package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/catielanier/portico/internal/i18n"
	"github.com/catielanier/portico/internal/jokes"
	"github.com/catielanier/portico/internal/plan"
	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/ui"
	"github.com/catielanier/portico/internal/useflags"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <atom>",
	Short: "Configure USE flags and install a package",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireRoot("install packages"); err != nil {
			return err
		}

		atom := args[0]

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

		maskActions, err := resolveInstallMasksInSandbox(atom, sandbox)
		if err != nil {
			return err
		}

		querier := portage.NewEqueryQuerier()

		queryResult, err := querier.Query(atom)
		if err != nil {
			return err
		}

		selections := useflags.FromQuery(queryResult)

		selected, ok, err := ui.RunUsePicker(atom, selections)
		if err != nil {
			return err
		}

		if !ok {
			fmt.Println("Install cancelled.")
			return nil
		}

		selectedFlags := useflags.SelectedFlags(selected)

		packageUsePath, err := portage.WritePackageUseEntry(
			sandbox.PortageConfigPath,
			atom,
			selectedFlags,
		)
		if err != nil {
			return err
		}

		pretendResolution, err := resolvePretendAutounmaskInSandbox(atom, sandbox)
		if err != nil {
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

		renderInstallPrototype(
			queryResult,
			selected,
			packageUsePath,
			maskActions,
			pretendResolution.RequiredUseChanges,
			transaction,
			pretendResolution.Result,
			pretendResolution.Err,
			t,
		)

		if pretendResolution.Err != nil {
			return pretendResolution.Err
		}

		confirmed, err := confirmDefaultNo("Apply configuration and install this package?")
		if err != nil {
			return err
		}

		if !confirmed {
			fmt.Println("Install cancelled.")
			return nil
		}

		if err := ui.RunStep("Writing Portage configuration", func() error {
			_, err := applyInstallConfigToSystem(
				atom,
				selectedFlags,
				maskActions,
				pretendResolution.RequiredUseChanges,
			)
			return err
		}); err != nil {
			return err
		}

		totalPackages := 0
		if transaction != nil {
			totalPackages = len(transaction.Packages)
		}

		if err := runPackageInstall(atom, totalPackages); err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("Install complete.")

		return nil
	},
}

type InstallMaskActions struct {
	AcceptKeywordsPath string
	PackageLicensePath string
	AcceptedKeyword    string
	AcceptedLicenses   []string
}

type AppliedInstallConfig struct {
	PackageUsePath     string
	AcceptKeywordsPath string
	PackageLicensePath string
	SelectedFlags      []string
	RequiredUseChanges []portage.RequiredUseChange
	AcceptedKeyword    string
	AcceptedLicenses   []string
}

type PretendResolution struct {
	Result             *portage.PretendResult
	Err                error
	RequiredUseChanges []portage.RequiredUseChange
}

func resolveInstallMasksInSandbox(atom string, sandbox *portage.ConfigSandbox) (*InstallMaskActions, error) {
	var initialPretendResult *portage.PretendResult
	var initialPretendErr error

	if err := ui.RunStep("Checking package availability", func() error {
		initialPretendResult, initialPretendErr = portage.EmergePretendWithConfigRoot(atom, sandbox.Root)

		if initialPretendErr != nil && initialPretendResult == nil {
			return initialPretendErr
		}

		return nil
	}); err != nil {
		return nil, err
	}

	if initialPretendErr == nil {
		return &InstallMaskActions{}, nil
	}

	if initialPretendResult == nil {
		return &InstallMaskActions{}, initialPretendErr
	}

	maskReport := portage.ParseMaskedPackageReport(atom, initialPretendResult.Raw)
	if maskReport == nil {
		return &InstallMaskActions{}, nil
	}

	candidate := portage.BestMaskedCandidate(maskReport)
	if candidate == nil {
		return nil, initialPretendErr
	}

	fmt.Println()
	fmt.Println("Portico detected that this package is masked.")
	fmt.Println()
	fmt.Println("Best candidate:")
	fmt.Printf("  %s::%s\n", candidate.Atom, candidate.Repository)
	fmt.Printf("  masked by: %s\n", candidate.RawReason)
	fmt.Println()

	if candidate.HasUnsupportedReasons() {
		fmt.Println("Portico does not automate this mask type yet.")
		fmt.Println()
		fmt.Print(initialPretendResult.Raw)
		return nil, initialPretendErr
	}

	actions := &InstallMaskActions{}

	if candidate.HasReason(portage.MaskReasonTestingKeyword) {
		keyword := candidate.RequiredKeyword
		if keyword == "" {
			keyword = "~amd64"
		}

		fmt.Println("Portico can allow this specific package keyword in the sandbox:")
		fmt.Printf("  %s %s\n", atom, keyword)
		fmt.Println()

		confirmed, err := confirmDefaultNo("Allow this package keyword in the sandbox?")
		if err != nil {
			return nil, err
		}

		if !confirmed {
			fmt.Println("Install cancelled.")
			return nil, fmt.Errorf("package keyword was not accepted")
		}

		path, err := portage.WriteAcceptKeywordEntry(
			sandbox.PortageConfigPath,
			atom,
			keyword,
		)
		if err != nil {
			return nil, err
		}

		actions.AcceptKeywordsPath = path
		actions.AcceptedKeyword = keyword
	}

	if candidate.HasReason(portage.MaskReasonLicense) {
		if len(candidate.RequiredLicenses) == 0 {
			fmt.Println("Portico detected a license mask, but could not determine the required license tokens.")
			fmt.Println()
			fmt.Print(initialPretendResult.Raw)
			return nil, initialPretendErr
		}

		fmt.Println("Portico can accept these licenses for this specific package in the sandbox:")
		fmt.Printf("  %s %s\n", atom, strings.Join(candidate.RequiredLicenses, " "))
		fmt.Println()

		confirmed, err := confirmDefaultNo("Accept these licenses for this package in the sandbox?")
		if err != nil {
			return nil, err
		}

		if !confirmed {
			fmt.Println("Install cancelled.")
			return nil, fmt.Errorf("package license was not accepted")
		}

		path, err := portage.WritePackageLicenseEntry(
			sandbox.PortageConfigPath,
			atom,
			candidate.RequiredLicenses,
		)
		if err != nil {
			return nil, err
		}

		actions.PackageLicensePath = path
		actions.AcceptedLicenses = candidate.RequiredLicenses
	}

	return actions, nil
}

func resolvePretendAutounmaskInSandbox(atom string, sandbox *portage.ConfigSandbox) (*PretendResolution, error) {
	const maxAttempts = 5

	resolution := &PretendResolution{}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var pretendResult *portage.PretendResult
		var pretendErr error

		label := "Running emerge --pretend"
		if attempt > 1 {
			label = fmt.Sprintf("Running emerge --pretend retry %d", attempt)
		}

		if err := ui.RunStep(label, func() error {
			pretendResult, pretendErr = portage.EmergePretendWithConfigRoot(atom, sandbox.Root)

			if pretendErr != nil && pretendResult == nil {
				return pretendErr
			}

			return nil
		}); err != nil {
			return nil, err
		}

		resolution.Result = pretendResult
		resolution.Err = pretendErr

		if pretendErr == nil {
			return resolution, nil
		}

		if pretendResult == nil {
			return resolution, nil
		}

		report := portage.ParseAutounmaskReport(pretendResult.Raw)
		if report == nil || len(report.RequiredUseChanges) == 0 {
			return resolution, nil
		}

		fmt.Println()
		fmt.Println("Portage requires additional USE changes to proceed:")
		fmt.Println()

		for _, change := range report.RequiredUseChanges {
			fmt.Printf("  %s %s\n", change.Atom, strings.Join(change.Flags, " "))

			for _, requiredBy := range change.RequiredBy {
				fmt.Printf("    required by: %s\n", requiredBy)
			}
		}

		fmt.Println()
		fmt.Println("Portico will apply these changes to the temporary sandbox and retry.")

		for _, change := range report.RequiredUseChanges {
			if _, err := portage.WritePackageUseEntry(
				sandbox.PortageConfigPath,
				change.Atom,
				change.Flags,
			); err != nil {
				return nil, err
			}

			resolution.RequiredUseChanges = append(resolution.RequiredUseChanges, change)
		}
	}

	return resolution, fmt.Errorf("emerge --pretend did not resolve after %d attempts", maxAttempts)
}

func applyInstallConfigToSystem(
	atom string,
	selectedFlags []string,
	maskActions *InstallMaskActions,
	requiredUseChanges []portage.RequiredUseChange,
) (*AppliedInstallConfig, error) {
	applied := &AppliedInstallConfig{
		SelectedFlags:      selectedFlags,
		RequiredUseChanges: requiredUseChanges,
	}

	if len(selectedFlags) > 0 {
		path, err := portage.WritePackageUseEntry(
			portage.SystemPortageConfigPath,
			atom,
			selectedFlags,
		)
		if err != nil {
			return nil, err
		}

		applied.PackageUsePath = path
	}

	for _, change := range requiredUseChanges {
		path, err := portage.WritePackageUseEntry(
			portage.SystemPortageConfigPath,
			change.Atom,
			change.Flags,
		)
		if err != nil {
			return nil, err
		}

		applied.PackageUsePath = path
	}

	if maskActions != nil && maskActions.AcceptedKeyword != "" {
		path, err := portage.WriteAcceptKeywordEntry(
			portage.SystemPortageConfigPath,
			atom,
			maskActions.AcceptedKeyword,
		)
		if err != nil {
			return nil, err
		}

		applied.AcceptKeywordsPath = path
		applied.AcceptedKeyword = maskActions.AcceptedKeyword
	}

	if maskActions != nil && len(maskActions.AcceptedLicenses) > 0 {
		path, err := portage.WritePackageLicenseEntry(
			portage.SystemPortageConfigPath,
			atom,
			maskActions.AcceptedLicenses,
		)
		if err != nil {
			return nil, err
		}

		applied.PackageLicensePath = path
		applied.AcceptedLicenses = maskActions.AcceptedLicenses
	}

	return applied, nil
}

func runPackageInstall(atom string, totalPackages int) error {
	installer := portage.NewEmergeInstaller()

	return ui.RunInstallProgress("Installing "+atom, totalPackages, func(ctx context.Context, events chan<- ui.InstallProgressEvent) error {
		return installer.InstallContext(ctx, atom, totalPackages, func(progress portage.InstallProgress) {
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

func renderInstallPrototype(
	queryResult *portage.PackageQuery,
	selections []useflags.FlagSelection,
	packageUsePath string,
	maskActions *InstallMaskActions,
	requiredUseChanges []portage.RequiredUseChange,
	transaction *portage.MergeTransaction,
	pretendResult *portage.PretendResult,
	pretendErr error,
	t *i18n.Translator,
) {
	atom := queryResult.Atom
	selectedFlags := useflags.SelectedFlags(selections)

	p := plan.Plan{
		TitleKey: "plan_title",
		Action:   "Install " + atom,
		Will: []plan.Item{
			{
				Key: "will_inspect_use_flags",
				Data: map[string]any{
					"Atom": atom,
				},
			},
			{
				Key: "will_create_config_sandbox",
			},
			{
				Key: "will_write_sandbox_package_use",
			},
			{
				Key: "will_run_emerge_pretend",
				Data: map[string]any{
					"Atom": atom,
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
				Key: "will_not_overwrite_package_use",
			},
			{
				Key: jokes.RandomKey(jokes.Context{
					Atom:    atom,
					Command: "install",
				}),
			},
		},
	}

	fmt.Println("Portico Install")
	fmt.Println()
	fmt.Println("Package:")
	fmt.Printf("  %s\n", atom)
	fmt.Println()

	renderUseFlagSummary(queryResult)

	if maskActions != nil {
		renderInstallMaskActions(atom, maskActions)
	}

	renderRequiredUseChanges(requiredUseChanges)

	fmt.Println()
	fmt.Println("Selected package.use entry:")

	if len(selectedFlags) > 0 {
		fmt.Printf("  %s %s\n", atom, strings.Join(selectedFlags, " "))
	} else {
		fmt.Println("  No explicit USE flag changes selected.")
	}

	if packageUsePath != "" {
		fmt.Println()
		fmt.Println("Sandbox package.use path:")
		fmt.Printf("  %s\n", packageUsePath)
	}

	fmt.Println()

	if transaction != nil {
		renderMergeTransaction(transaction)
	}

	if transaction == nil && pretendResult != nil && pretendResult.Raw != "" {
		fmt.Println("emerge --pretend output:")
		fmt.Println()
		fmt.Print(pretendResult.Raw)

		if !strings.HasSuffix(pretendResult.Raw, "\n") {
			fmt.Println()
		}
	}

	if transaction == nil && pretendResult != nil && pretendResult.Raw == "" {
		fmt.Println("emerge --pretend output:")
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

func renderInstallMaskActions(atom string, actions *InstallMaskActions) {
	if actions.AcceptedKeyword != "" {
		fmt.Println()
		fmt.Println("Sandbox package.accept_keywords entry:")
		fmt.Printf("  %s %s\n", atom, actions.AcceptedKeyword)

		if actions.AcceptKeywordsPath != "" {
			fmt.Println("Sandbox package.accept_keywords path:")
			fmt.Printf("  %s\n", actions.AcceptKeywordsPath)
		}
	}

	if len(actions.AcceptedLicenses) > 0 {
		fmt.Println()
		fmt.Println("Sandbox package.license entry:")
		fmt.Printf("  %s %s\n", atom, strings.Join(actions.AcceptedLicenses, " "))

		if actions.PackageLicensePath != "" {
			fmt.Println("Sandbox package.license path:")
			fmt.Printf("  %s\n", actions.PackageLicensePath)
		}
	}
}

func renderRequiredUseChanges(changes []portage.RequiredUseChange) {
	if len(changes) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Sandbox dependency USE changes:")

	for _, change := range changes {
		fmt.Printf("  %s %s\n", change.Atom, strings.Join(change.Flags, " "))

		for _, requiredBy := range change.RequiredBy {
			fmt.Printf("    required by: %s\n", requiredBy)
		}
	}
}

func renderMergeTransaction(transaction *portage.MergeTransaction) {
	if transaction == nil {
		return
	}

	atoms := transaction.PackageAtoms()

	if len(atoms) > 0 {
		fmt.Println("Dependencies to install:")
		fmt.Println("  " + wrapCommaList(atoms, 2, 88))
		fmt.Println()
	}

	if transaction.TotalLine != "" {
		fmt.Println("Summary:")
		fmt.Printf("  %s\n", transaction.TotalLine)
	}
}

func wrapCommaList(items []string, indentLevel int, maxWidth int) string {
	if len(items) == 0 {
		return ""
	}

	indent := strings.Repeat("  ", indentLevel)
	var b strings.Builder

	lineLength := 0

	for i, item := range items {
		part := item
		if i < len(items)-1 {
			part += ","
		}

		if lineLength > 0 && lineLength+1+len(part) > maxWidth {
			b.WriteString("\n")
			b.WriteString(indent)
			b.WriteString(part)
			lineLength = len(indent) + len(part)
			continue
		}

		if lineLength > 0 {
			b.WriteString(" ")
			lineLength++
		}

		b.WriteString(part)
		lineLength += len(part)
	}

	return b.String()
}
