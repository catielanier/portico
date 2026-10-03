package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/jokes"
	"github.com/catielanier/portico/internal/plan"
	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/ui"
	"github.com/catielanier/portico/internal/useflags"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [atom[@version]...]",
	Short: "Update the world set or one or more packages",
	Args:  validateZeroOrMorePackageTargetArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireRoot("update packages"); err != nil {
			return err
		}

		atoms, err := packageAtomsFromArgsOrWorld(args)
		if err != nil {
			return err
		}

		if err := syncRepositoriesForMutation(); err != nil {
			return err
		}

		var sandbox *portage.ConfigSandbox

		if err := ui.RunStep("Creating temporary Portage config sandbox", func() error {
			var err error
			sandbox, err = portage.NewConfigSandbox()
			return err
		}); err != nil {
			return err
		}
		defer sandbox.Cleanup()

		maskActions := NewInstallMaskActions()

		pretendResolution, err := resolveUpdatePretendProblemsInSandbox(atoms, sandbox, maskActions)
		if err != nil {
			if isRequiredUseResolutionCancelled(err) {
				fmt.Println("Update cancelled.")
				return nil
			}

			return err
		}

		t, err := i18n.New("en")
		if err != nil {
			return err
		}

		transaction := (*portage.MergeTransaction)(nil)
		if pretendResolution.Result != nil {
			transaction = portage.ParseMergeTransaction(pretendResolution.Result.Raw)
		}

		renderUpdatePlan(
			atoms,
			maskActions,
			pretendResolution.RequiredUseChanges,
			transaction,
			pretendResolution.Result,
			pretendResolution.Err,
			t,
		)

		if pretendResolution.Err != nil {
			fmt.Println()
			fmt.Println(ui.Error("Portico could not resolve the remaining update blocker safely."))
			fmt.Println("No real Portage configuration has been changed.")
			return pretendResolution.Err
		}

		confirmed, err := confirmDefaultNo("Persist these changes and run this update?")
		if err != nil {
			return err
		}

		if !confirmed {
			fmt.Println("Update cancelled.")
			return nil
		}

		if updateHasConfigurationChanges(maskActions, pretendResolution.RequiredUseChanges) {
			if err := ui.RunStep("Writing Portage configuration", func() error {
				_, err := applyInstallConfigToSystem(
					map[string][]string{},
					maskActions,
					pretendResolution.RequiredUseChanges,
				)
				return err
			}); err != nil {
				return err
			}
		}

		totalPackages := 0
		if transaction != nil {
			totalPackages = len(transaction.Packages)
		}

		if err := runPackageUpdate(atoms, totalPackages); err != nil {
			return err
		}

		fmt.Println()
		fmt.Println(ui.Success("Update complete."))

		return nil
	},
}

func packageAtomsFromArgsOrWorld(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}

	return packageAtomsFromArgs(args)
}

func resolveUpdatePretendProblemsInSandbox(
	atoms []string,
	sandbox *portage.ConfigSandbox,
	maskActions *InstallMaskActions,
) (*PretendResolution, error) {
	resolution := &PretendResolution{}
	attempt := 0

	for {
		attempt++

		var pretendResult *portage.PretendResult
		var pretendErr error

		label := "Running emerge --pretend --update --deep --newuse"
		if attempt > 1 {
			label = fmt.Sprintf("Running emerge --pretend --update --deep --newuse retry %d", attempt)
		}

		if err := ui.RunStep(label, func() error {
			pretendResult, pretendErr = portage.EmergePretendUpdateWithConfigRootForAtoms(atoms, sandbox.Root)

			if pretendErr != nil && pretendResult == nil {
				return pretendErr
			}

			return nil
		}); err != nil {
			return nil, err
		}

		resolution.Result = pretendResult
		resolution.Err = pretendErr
		resolution.RequiredUseChanges = dedupeRequiredUseChanges(resolution.RequiredUseChanges)

		if pretendErr == nil {
			return resolution, nil
		}

		if pretendResult == nil {
			return resolution, nil
		}

		before := updateResolutionFingerprint(resolution.RequiredUseChanges, maskActions)
		handled := false

		if requiredUseHandled, err := resolveRequiredUseFailureInSandbox(
			pretendResult.Raw,
			sandbox,
			atoms,
			&resolution.RequiredUseChanges,
		); requiredUseHandled || err != nil {
			if err != nil {
				return nil, err
			}

			handled = true
		} else if autounmaskHandled, err := resolveUpdateAutounmaskChangesInSandbox(
			pretendResult.Raw,
			sandbox,
			maskActions,
			&resolution.RequiredUseChanges,
		); autounmaskHandled || err != nil {
			if err != nil {
				return nil, err
			}

			handled = true
		} else {
			maskReport := portage.ParseMaskedPackageReport("", pretendResult.Raw)
			if maskReport != nil {
				if err := applyUpdateMaskedPackageReportInSandbox(maskReport, sandbox, maskActions, pretendErr); err != nil {
					return resolution, nil
				}

				handled = true
			}
		}

		resolution.RequiredUseChanges = dedupeRequiredUseChanges(resolution.RequiredUseChanges)

		if !handled {
			return resolution, nil
		}

		after := updateResolutionFingerprint(resolution.RequiredUseChanges, maskActions)
		if before == after {
			fmt.Println()
			fmt.Println(ui.Warning("Portico could not make further progress resolving this update."))
			fmt.Println("The remaining Portage failure is shown below in the final review.")
			return resolution, nil
		}
	}
}

func applyUpdateMaskedPackageReportInSandbox(
	report *portage.MaskedPackageReport,
	sandbox *portage.ConfigSandbox,
	maskActions *InstallMaskActions,
	originalErr error,
) error {
	if report == nil {
		return nil
	}

	candidate := portage.BestMaskedCandidate(report)
	if candidate == nil {
		return originalErr
	}

	requestedAtom := strings.TrimSpace(report.RequestedAtom)
	if requestedAtom == "" {
		requestedAtom = candidate.Atom
	}

	fmt.Println()
	fmt.Println(ui.Warning("Portico detected that a package required by this update is masked."))
	fmt.Println()
	fmt.Println(ui.Accent("Best candidate:"))
	fmt.Printf("  %s::%s\n", candidate.Atom, candidate.Repository)
	fmt.Printf("  masked by: %s\n", ui.Warning(candidate.RawReason))
	fmt.Println()

	if candidate.HasUnsupportedReasons() {
		fmt.Println(ui.Error("Portico does not automate this mask type yet."))
		fmt.Println("The update will stop without modifying real Portage configuration.")
		return originalErr
	}

	if candidate.HasReason(portage.MaskReasonTestingKeyword) {
		keyword := candidate.RequiredKeyword
		if keyword == "" {
			fmt.Println(ui.Error("Portico detected a keyword mask, but could not determine the required keyword token."))
			return originalErr
		}

		if !maskActions.AcceptedKeywords[keyword] {
			fmt.Println(ui.Info("Portico can allow this keyword for packages required by the update:"))
			fmt.Printf("  %s\n", ui.Info(keyword))
			fmt.Println()

			confirmed, err := confirmDefaultNo("Allow this keyword for this update?")
			if err != nil {
				return err
			}
			if !confirmed {
				return errRequiredUseResolutionCancelled
			}

			maskActions.AcceptedKeywords[keyword] = true
		}

		if err := writeKeywordMaskEntryInSandbox(sandbox, maskActions, requestedAtom, keyword); err != nil {
			return err
		}
	}

	if candidate.HasReason(portage.MaskReasonLicense) {
		if len(candidate.RequiredLicenses) == 0 {
			fmt.Println(ui.Error("Portico detected a license mask, but could not determine the required license tokens."))
			return originalErr
		}

		newLicenses := newLicenseTokens(maskActions, candidate.RequiredLicenses)
		if len(newLicenses) > 0 {
			fmt.Println(ui.Info("Portico can accept these licenses for packages required by the update:"))
			fmt.Printf("  %s\n", renderInfoTokens(newLicenses))
			fmt.Println()

			confirmed, err := confirmDefaultNo("Accept these licenses for this update?")
			if err != nil {
				return err
			}
			if !confirmed {
				return errRequiredUseResolutionCancelled
			}

			for _, license := range newLicenses {
				maskActions.AcceptedLicenses[license] = true
			}
		}

		if err := writeLicenseMaskEntryInSandbox(sandbox, maskActions, requestedAtom, candidate.RequiredLicenses); err != nil {
			return err
		}
	}

	fmt.Println(ui.Success("Portico applied the confirmed mask changes to the temporary sandbox and will retry."))
	return nil
}

func resolveUpdateAutounmaskChangesInSandbox(
	raw string,
	sandbox *portage.ConfigSandbox,
	maskActions *InstallMaskActions,
	requiredUseChanges *[]portage.RequiredUseChange,
) (bool, error) {
	report := portage.ParseAutounmaskReport(raw)
	if report == nil {
		return false, nil
	}

	handled := false

	newUseChanges := filterNewRequiredUseChanges(*requiredUseChanges, report.RequiredUseChanges)
	if len(newUseChanges) > 0 {
		if err := confirmAndApplyUpdateUseChangesInSandbox(newUseChanges, sandbox, requiredUseChanges); err != nil {
			return true, err
		}
		handled = true
	}

	if len(report.RequiredKeywordChanges) > 0 {
		before := len(maskActions.KeywordEntries)
		if err := applyRequiredKeywordChangesInSandbox(report.RequiredKeywordChanges, sandbox, maskActions); err != nil {
			return true, err
		}
		if len(maskActions.KeywordEntries) > before {
			handled = true
		}
	}

	if len(report.RequiredLicenseChanges) > 0 {
		before := len(maskActions.LicenseEntries)
		if err := applyRequiredLicenseChangesInSandbox(report.RequiredLicenseChanges, sandbox, maskActions); err != nil {
			return true, err
		}
		if len(maskActions.LicenseEntries) > before {
			handled = true
		}
	}

	return handled, nil
}

func confirmAndApplyUpdateUseChangesInSandbox(
	changes []portage.RequiredUseChange,
	sandbox *portage.ConfigSandbox,
	requiredUseChanges *[]portage.RequiredUseChange,
) error {
	if len(changes) == 0 {
		return nil
	}

	fmt.Println()
	fmt.Println(ui.Warning("Portage requires additional USE changes to complete this update:"))
	fmt.Println()

	for _, change := range changes {
		fmt.Printf("  %s %s\n", change.Atom, renderUSEChangeTokens(change.Flags))
		for _, requiredBy := range change.RequiredBy {
			fmt.Printf("    required by: %s\n", requiredBy)
		}
	}

	fmt.Println()
	fmt.Println("These changes will be applied only to Portico's temporary sandbox first.")
	fmt.Println()

	confirmed, err := confirmDefaultNo("Apply these USE changes to the temporary sandbox?")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Update cancelled.")
		return errRequiredUseResolutionCancelled
	}

	for _, change := range changes {
		resolvedChange, err := resolveUpdateUseCollisionBeforeWrite(change, sandbox)
		if err != nil {
			return err
		}

		cleanedFlags := cleanStringList(resolvedChange.Flags)
		if len(cleanedFlags) == 0 {
			continue
		}

		if _, err := portage.WritePackageUseEntry(
			sandbox.PortageConfigPath,
			resolvedChange.Atom,
			cleanedFlags,
		); err != nil {
			return err
		}

		*requiredUseChanges = appendRequiredUseChangeUnique(*requiredUseChanges, portage.RequiredUseChange{
			Atom:       resolvedChange.Atom,
			Flags:      cleanedFlags,
			RequiredBy: append([]string(nil), resolvedChange.RequiredBy...),
			Raw:        resolvedChange.Raw,
		})
	}

	fmt.Println()
	fmt.Println(ui.Success("Portico applied the confirmed USE changes to the temporary sandbox and will retry."))

	return nil
}

func resolveUpdateUseCollisionBeforeWrite(
	change portage.RequiredUseChange,
	sandbox *portage.ConfigSandbox,
) (portage.RequiredUseChange, error) {
	change.Atom = strings.TrimSpace(change.Atom)
	change.Flags = cleanStringList(change.Flags)
	if change.Atom == "" || len(change.Flags) == 0 {
		return change, nil
	}

	querier := portage.NewEqueryQuerierWithConfigRoot(sandbox.Root)
	queryResult, err := querier.Query(change.Atom)
	if err != nil || strings.TrimSpace(queryResult.RequiredUse) == "" || len(queryResult.Uses) == 0 {
		// Portage's pretend pass remains authoritative. If metadata inspection is
		// unavailable here, apply the requested change only to the sandbox and
		// let the next pretend classify any resulting REQUIRED_USE failure.
		return change, nil
	}

	expression, err := portage.ParseRequiredUseExpression(queryResult.RequiredUse)
	if err != nil {
		return change, nil
	}

	selections := useflags.FromQuery(queryResult)
	applyRequestedUseFlagsToSelections(selections, change.Flags)

	if expression.Satisfied(useflags.EffectiveEnabledMap(selections)) {
		return change, nil
	}

	fmt.Println()
	fmt.Println(ui.Warning("The requested update-time USE change conflicts with REQUIRED_USE:"))
	fmt.Printf("  %s %s\n", change.Atom, renderUSEChangeTokens(change.Flags))
	fmt.Println()
	fmt.Println(ui.Info("Resolve the conflict below. Portico will keep the result in the temporary sandbox until the final update review."))
	fmt.Println()

	selected, ok, err := ui.RunUsePicker(change.Atom, selections, queryResult.RequiredUse)
	if err != nil {
		return change, err
	}
	if !ok {
		return change, errRequiredUseResolutionCancelled
	}

	change.Flags = useflags.SelectedFlags(selected)
	if len(change.Flags) == 0 {
		return change, fmt.Errorf("REQUIRED_USE resolution for %s produced no scoped USE changes", change.Atom)
	}

	return change, nil
}

func applyRequestedUseFlagsToSelections(selections []useflags.FlagSelection, flags []string) {
	requested := make(map[string]useflags.SelectionState)

	for _, flag := range cleanStringList(flags) {
		state := useflags.SelectionEnabled
		name := flag
		if strings.HasPrefix(flag, "-") {
			state = useflags.SelectionDisabled
			name = strings.TrimPrefix(flag, "-")
		}

		name = strings.TrimSpace(name)
		if name != "" {
			requested[name] = state
		}
	}

	for i := range selections {
		if state, ok := requested[selections[i].Name]; ok {
			selections[i].Selection = state
		}
	}
}

func filterNewRequiredUseChanges(
	existing []portage.RequiredUseChange,
	incoming []portage.RequiredUseChange,
) []portage.RequiredUseChange {
	seen := make(map[string]bool)
	for _, change := range dedupeRequiredUseChanges(existing) {
		seen[requiredUseChangeKey(change)] = true
	}

	var out []portage.RequiredUseChange
	for _, change := range dedupeRequiredUseChanges(incoming) {
		key := requiredUseChangeKey(change)
		if key == "" || seen[key] {
			continue
		}

		seen[key] = true
		out = append(out, change)
	}

	return out
}

func updateResolutionFingerprint(
	requiredUseChanges []portage.RequiredUseChange,
	maskActions *InstallMaskActions,
) string {
	var parts []string

	for _, change := range dedupeRequiredUseChanges(requiredUseChanges) {
		parts = append(parts, "use:"+requiredUseChangeKey(change))
	}

	if maskActions != nil {
		for _, entry := range maskActions.KeywordEntries {
			parts = append(parts, "keyword:"+strings.TrimSpace(entry.Atom)+" "+strings.TrimSpace(entry.Keyword))
		}
		for _, entry := range maskActions.LicenseEntries {
			parts = append(parts, "license:"+strings.TrimSpace(entry.Atom)+" "+strings.Join(cleanStringList(entry.Licenses), " "))
		}
	}

	sort.Strings(parts)
	return strings.Join(parts, "\n")
}

func updateHasConfigurationChanges(
	maskActions *InstallMaskActions,
	requiredUseChanges []portage.RequiredUseChange,
) bool {
	if len(dedupeRequiredUseChanges(requiredUseChanges)) > 0 {
		return true
	}

	if maskActions == nil {
		return false
	}

	return len(maskActions.KeywordEntries) > 0 || len(maskActions.LicenseEntries) > 0
}

func renderUpdatePlan(
	atoms []string,
	maskActions *InstallMaskActions,
	requiredUseChanges []portage.RequiredUseChange,
	transaction *portage.MergeTransaction,
	pretendResult *portage.PretendResult,
	pretendErr error,
	t *i18n.Translator,
) {
	target := updateTargetLabel(atoms)

	p := plan.Plan{
		TitleKey: "plan_title",
		Action:   "Update " + target,
		Will: []plan.Item{
			{
				Key: "will_run_emerge_update_pretend",
				Data: map[string]any{
					"Target": target,
				},
			},
			{
				Key: "will_show_portage_transaction",
			},
		},
		WillNot: []plan.Item{
			{
				Key: "will_not_modify_global_use",
				Data: map[string]any{
					"Path": "/etc/portage/make.conf",
				},
			},
			{
				Key: jokes.RandomKey(jokes.Context{
					Atom:    firstUpdateJokeAtom(atoms),
					Command: "update",
				}),
			},
		},
	}

	fmt.Println(ui.Selected("Portico Update"))
	fmt.Println()
	fmt.Println(ui.Accent("Target:"))
	fmt.Printf("  %s\n", target)
	fmt.Println()

	renderUpdateConfigurationChanges(requiredUseChanges, maskActions)

	if transaction != nil {
		renderMergeTransaction(transaction)
	}

	if transaction == nil && pretendResult != nil && pretendResult.Raw != "" {
		fmt.Println(ui.Accent("emerge --pretend output:"))
		fmt.Println()
		fmt.Print(pretendResult.Raw)

		if !strings.HasSuffix(pretendResult.Raw, "\n") {
			fmt.Println()
		}
	}

	if transaction == nil && pretendResult != nil && pretendResult.Raw == "" {
		fmt.Println(ui.Accent("emerge --pretend output:"))
		fmt.Println()
		fmt.Println("  No output returned.")
	}

	if pretendErr != nil {
		fmt.Println()
		fmt.Printf("Pretend result: %v\n", pretendErr)
	}

	fmt.Println()
	fmt.Print(ui.RenderPlanWithoutConfirmation(p, t))
}

func renderUpdateConfigurationChanges(
	requiredUseChanges []portage.RequiredUseChange,
	maskActions *InstallMaskActions,
) {
	requiredUseChanges = dedupeRequiredUseChanges(requiredUseChanges)

	if !updateHasConfigurationChanges(maskActions, requiredUseChanges) {
		return
	}

	fmt.Println(ui.Accent("Additional configuration required for this update:"))
	fmt.Println()

	if len(requiredUseChanges) > 0 {
		fmt.Println(ui.Warning("USE changes:"))
		for _, change := range requiredUseChanges {
			fmt.Printf("  %s %s\n", change.Atom, renderUSEChangeTokens(change.Flags))
			for _, requiredBy := range change.RequiredBy {
				fmt.Printf("    required by: %s\n", requiredBy)
			}
		}
		fmt.Println()
	}

	if maskActions != nil && len(maskActions.LicenseEntries) > 0 {
		fmt.Println(ui.Warning("License changes:"))
		for _, entry := range maskActions.LicenseEntries {
			fmt.Printf("  %s %s\n", entry.Atom, renderInfoTokens(entry.Licenses))
		}
		fmt.Println()
	}

	if maskActions != nil && len(maskActions.KeywordEntries) > 0 {
		fmt.Println(ui.Warning("Keyword / mask changes:"))
		for _, entry := range maskActions.KeywordEntries {
			fmt.Printf("  %s %s\n", entry.Atom, ui.Info(entry.Keyword))
		}
		fmt.Println()
	}

	fmt.Println(ui.Success("These changes were validated in Portico's temporary sandbox."))
	fmt.Println()
}

func runPackageUpdate(atoms []string, totalPackages int) error {
	installer := portage.NewEmergeInstaller()

	label := "Updating " + updateTargetLabel(atoms)

	return ui.RunInstallProgress(label, totalPackages, func(ctx context.Context, events chan<- ui.InstallProgressEvent) error {
		return installer.UpdateAtomsContext(ctx, atoms, totalPackages, func(progress portage.InstallProgress) {
			event := ui.InstallProgressEvent{
				CurrentPackage: progress.CurrentPackage,
				CurrentIndex:   progress.CurrentIndex,
				Total:          progress.Total,
			}

			select {
			case events <- event:
			case <-ctx.Done():
			}
		})
	})
}

func updateTargetLabel(atoms []string) string {
	cleanAtoms := cleanInstallArgs(atoms)
	if len(cleanAtoms) == 0 {
		return "@world"
	}

	return strings.Join(cleanAtoms, " ")
}

func firstUpdateJokeAtom(atoms []string) string {
	cleanAtoms := cleanInstallArgs(atoms)
	if len(cleanAtoms) == 0 {
		return "@world"
	}

	return cleanAtoms[0]
}
