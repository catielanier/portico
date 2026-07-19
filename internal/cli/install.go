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
	Use:   "install <atom...>",
	Short: "Configure USE flags and install one or more packages",
	Args:  validateOneOrMoreAtomArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireRoot("install packages"); err != nil {
			return err
		}

		atoms := cleanInstallArgs(args)

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

		if err := resolveInitialInstallMasksInSandbox(atoms, sandbox, maskActions); err != nil {
			return err
		}

		queryResults := make([]*portage.PackageQuery, 0, len(atoms))
		selectedFlagsByAtom := make(map[string][]string)
		selectionsByAtom := make(map[string][]useflags.FlagSelection)
		packageUsePaths := make([]string, 0, len(atoms))

		querier := portage.NewEqueryQuerier()

		for _, atom := range atoms {
			queryResult, err := querier.Query(atom)
			if err != nil {
				return err
			}

			queryResults = append(queryResults, queryResult)

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

			selectedFlagsByAtom[atom] = selectedFlags
			selectionsByAtom[atom] = selected
			packageUsePaths = append(packageUsePaths, packageUsePath)
		}

		pretendResolution, err := resolvePretendProblemsInSandbox(atoms, sandbox, maskActions)
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
			atoms,
			queryResults,
			selectionsByAtom,
			selectedFlagsByAtom,
			packageUsePaths,
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

		confirmed, err := confirmDefaultNo("Apply configuration and install these packages?")
		if err != nil {
			return err
		}

		if !confirmed {
			fmt.Println("Install cancelled.")
			return nil
		}

		if err := ui.RunStep("Writing Portage configuration", func() error {
			_, err := applyInstallConfigToSystem(
				selectedFlagsByAtom,
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

		if err := runPackageInstall(atoms, totalPackages); err != nil {
			return err
		}

		fmt.Println()
		fmt.Println("Install complete.")

		return nil
	},
}

type InstallKeywordEntry struct {
	Atom    string
	Keyword string
	Path    string
}

type InstallLicenseEntry struct {
	Atom     string
	Licenses []string
	Path     string
}

type InstallMaskActions struct {
	AcceptedKeywords map[string]bool
	AcceptedLicenses map[string]bool
	KeywordEntries   []InstallKeywordEntry
	LicenseEntries   []InstallLicenseEntry
	writtenKeywords  map[string]bool
	writtenLicenses  map[string]bool
}

type AppliedInstallConfig struct {
	PackageUsePath      string
	AcceptKeywordsPath  string
	PackageLicensePath  string
	SelectedFlagsByAtom map[string][]string
	RequiredUseChanges  []portage.RequiredUseChange
	KeywordEntries      []InstallKeywordEntry
	LicenseEntries      []InstallLicenseEntry
}

type PretendResolution struct {
	Result             *portage.PretendResult
	Err                error
	RequiredUseChanges []portage.RequiredUseChange
}

func NewInstallMaskActions() *InstallMaskActions {
	return &InstallMaskActions{
		AcceptedKeywords: make(map[string]bool),
		AcceptedLicenses: make(map[string]bool),
		writtenKeywords:  make(map[string]bool),
		writtenLicenses:  make(map[string]bool),
	}
}

func resolveInitialInstallMasksInSandbox(
	atoms []string,
	sandbox *portage.ConfigSandbox,
	maskActions *InstallMaskActions,
) error {
	const maxAttempts = 8

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var initialPretendResult *portage.PretendResult
		var initialPretendErr error

		label := "Checking package availability"
		if attempt > 1 {
			label = fmt.Sprintf("Checking package availability retry %d", attempt)
		}

		if err := ui.RunStep(label, func() error {
			initialPretendResult, initialPretendErr = portage.EmergePretendWithConfigRootForAtoms(atoms, sandbox.Root)

			if initialPretendErr != nil && initialPretendResult == nil {
				return initialPretendErr
			}

			return nil
		}); err != nil {
			return err
		}

		if initialPretendErr == nil {
			return nil
		}

		if initialPretendResult == nil {
			return initialPretendErr
		}

		maskReport := portage.ParseMaskedPackageReport("", initialPretendResult.Raw)
		if maskReport == nil {
			return nil
		}

		if err := applyMaskedPackageReportInSandbox(maskReport, sandbox, maskActions, initialPretendErr); err != nil {
			return err
		}
	}

	return fmt.Errorf("package availability did not resolve after %d attempts", maxAttempts)
}

func resolvePretendProblemsInSandbox(
	atoms []string,
	sandbox *portage.ConfigSandbox,
	maskActions *InstallMaskActions,
) (*PretendResolution, error) {
	const maxAttempts = 8

	resolution := &PretendResolution{}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var pretendResult *portage.PretendResult
		var pretendErr error

		label := "Running emerge --pretend"
		if attempt > 1 {
			label = fmt.Sprintf("Running emerge --pretend retry %d", attempt)
		}

		if err := ui.RunStep(label, func() error {
			pretendResult, pretendErr = portage.EmergePretendWithConfigRootForAtoms(atoms, sandbox.Root)

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

		useReport := portage.ParseAutounmaskReport(pretendResult.Raw)
		if useReport != nil && len(useReport.RequiredUseChanges) > 0 {
			fmt.Println()
			fmt.Println("Portage requires additional USE changes to proceed:")
			fmt.Println()

			for _, change := range useReport.RequiredUseChanges {
				fmt.Printf("  %s %s\n", change.Atom, strings.Join(change.Flags, " "))

				for _, requiredBy := range change.RequiredBy {
					fmt.Printf("    required by: %s\n", requiredBy)
				}
			}

			fmt.Println()
			fmt.Println("Portico will apply these changes to the temporary sandbox and retry.")

			for _, change := range useReport.RequiredUseChanges {
				if _, err := portage.WritePackageUseEntry(
					sandbox.PortageConfigPath,
					change.Atom,
					change.Flags,
				); err != nil {
					return nil, err
				}

				resolution.RequiredUseChanges = append(resolution.RequiredUseChanges, change)
			}

			continue
		}

		maskReport := portage.ParseMaskedPackageReport("", pretendResult.Raw)
		if maskReport != nil {
			if err := applyMaskedPackageReportInSandbox(maskReport, sandbox, maskActions, pretendErr); err != nil {
				return nil, err
			}

			continue
		}

		return resolution, nil
	}

	return resolution, fmt.Errorf("emerge --pretend did not resolve after %d attempts", maxAttempts)
}

func applyMaskedPackageReportInSandbox(
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
	fmt.Println("Portico detected that a package in this transaction is masked.")
	fmt.Println()
	fmt.Println("Best candidate:")
	fmt.Printf("  %s::%s\n", candidate.Atom, candidate.Repository)
	fmt.Printf("  masked by: %s\n", candidate.RawReason)
	fmt.Println()

	if candidate.HasUnsupportedReasons() {
		fmt.Println("Portico does not automate this mask type yet.")
		return originalErr
	}

	if candidate.HasReason(portage.MaskReasonTestingKeyword) {
		keyword := candidate.RequiredKeyword
		if keyword == "" {
			keyword = "~amd64"
		}

		if !maskActions.AcceptedKeywords[keyword] {
			fmt.Println("Portico can allow this keyword for packages required by this transaction:")
			fmt.Printf("  %s\n", keyword)
			fmt.Println()

			confirmed, err := confirmDefaultNo("Allow this keyword for this transaction?")
			if err != nil {
				return err
			}

			if !confirmed {
				fmt.Println("Install cancelled.")
				return fmt.Errorf("package keyword was not accepted")
			}

			maskActions.AcceptedKeywords[keyword] = true
		} else {
			fmt.Printf("Portico already has permission to apply keyword %s in this transaction.\n", keyword)
		}

		if err := writeKeywordMaskEntryInSandbox(sandbox, maskActions, requestedAtom, keyword); err != nil {
			return err
		}
	}

	if candidate.HasReason(portage.MaskReasonLicense) {
		if len(candidate.RequiredLicenses) == 0 {
			fmt.Println("Portico detected a license mask, but could not determine the required license tokens.")
			return originalErr
		}

		newLicenses := newLicenseTokens(maskActions, candidate.RequiredLicenses)
		if len(newLicenses) > 0 {
			fmt.Println("Portico can accept these licenses for packages required by this transaction:")
			fmt.Printf("  %s\n", strings.Join(newLicenses, " "))
			fmt.Println()

			confirmed, err := confirmDefaultNo("Accept these licenses for this transaction?")
			if err != nil {
				return err
			}

			if !confirmed {
				fmt.Println("Install cancelled.")
				return fmt.Errorf("package license was not accepted")
			}

			for _, license := range newLicenses {
				maskActions.AcceptedLicenses[license] = true
			}
		} else {
			fmt.Println("Portico already has permission to apply these license tokens in this transaction.")
		}

		if err := writeLicenseMaskEntryInSandbox(sandbox, maskActions, requestedAtom, candidate.RequiredLicenses); err != nil {
			return err
		}
	}

	return nil
}

func writeKeywordMaskEntryInSandbox(
	sandbox *portage.ConfigSandbox,
	maskActions *InstallMaskActions,
	atom string,
	keyword string,
) error {
	key := atom + " " + keyword
	if maskActions.writtenKeywords[key] {
		return nil
	}

	path, err := portage.WriteAcceptKeywordEntry(
		sandbox.PortageConfigPath,
		atom,
		keyword,
	)
	if err != nil {
		return err
	}

	maskActions.writtenKeywords[key] = true
	maskActions.KeywordEntries = append(maskActions.KeywordEntries, InstallKeywordEntry{
		Atom:    atom,
		Keyword: keyword,
		Path:    path,
	})

	return nil
}

func writeLicenseMaskEntryInSandbox(
	sandbox *portage.ConfigSandbox,
	maskActions *InstallMaskActions,
	atom string,
	licenses []string,
) error {
	cleanedLicenses := cleanStringList(licenses)
	if len(cleanedLicenses) == 0 {
		return nil
	}

	key := atom + " " + strings.Join(cleanedLicenses, " ")
	if maskActions.writtenLicenses[key] {
		return nil
	}

	path, err := portage.WritePackageLicenseEntry(
		sandbox.PortageConfigPath,
		atom,
		cleanedLicenses,
	)
	if err != nil {
		return err
	}

	maskActions.writtenLicenses[key] = true
	maskActions.LicenseEntries = append(maskActions.LicenseEntries, InstallLicenseEntry{
		Atom:     atom,
		Licenses: cleanedLicenses,
		Path:     path,
	})

	return nil
}

func newLicenseTokens(maskActions *InstallMaskActions, licenses []string) []string {
	var out []string

	for _, license := range cleanStringList(licenses) {
		if maskActions.AcceptedLicenses[license] {
			continue
		}

		out = append(out, license)
	}

	return out
}

func cleanStringList(values []string) []string {
	var out []string
	seen := make(map[string]bool)

	for _, value := range values {
		value = strings.TrimSpace(value)
		value = strings.Trim(value, ",")

		if value == "" {
			continue
		}

		if seen[value] {
			continue
		}

		seen[value] = true
		out = append(out, value)
	}

	return out
}

func applyInstallConfigToSystem(
	selectedFlagsByAtom map[string][]string,
	maskActions *InstallMaskActions,
	requiredUseChanges []portage.RequiredUseChange,
) (*AppliedInstallConfig, error) {
	applied := &AppliedInstallConfig{
		SelectedFlagsByAtom: selectedFlagsByAtom,
		RequiredUseChanges:  requiredUseChanges,
	}

	for atom, selectedFlags := range selectedFlagsByAtom {
		if len(selectedFlags) == 0 {
			continue
		}

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

	if maskActions != nil {
		for _, entry := range maskActions.KeywordEntries {
			path, err := portage.WriteAcceptKeywordEntry(
				portage.SystemPortageConfigPath,
				entry.Atom,
				entry.Keyword,
			)
			if err != nil {
				return nil, err
			}

			applied.AcceptKeywordsPath = path
			applied.KeywordEntries = append(applied.KeywordEntries, InstallKeywordEntry{
				Atom:    entry.Atom,
				Keyword: entry.Keyword,
				Path:    path,
			})
		}

		for _, entry := range maskActions.LicenseEntries {
			path, err := portage.WritePackageLicenseEntry(
				portage.SystemPortageConfigPath,
				entry.Atom,
				entry.Licenses,
			)
			if err != nil {
				return nil, err
			}

			applied.PackageLicensePath = path
			applied.LicenseEntries = append(applied.LicenseEntries, InstallLicenseEntry{
				Atom:     entry.Atom,
				Licenses: entry.Licenses,
				Path:     path,
			})
		}
	}

	return applied, nil
}

func runPackageInstall(atoms []string, totalPackages int) error {
	installer := portage.NewEmergeInstaller()

	label := "Installing " + strings.Join(atoms, " ")

	return ui.RunInstallProgress(label, totalPackages, func(ctx context.Context, events chan<- ui.InstallProgressEvent) error {
		return installer.InstallAtomsContext(ctx, atoms, totalPackages, func(progress portage.InstallProgress) {
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
	atoms []string,
	queryResults []*portage.PackageQuery,
	selectionsByAtom map[string][]useflags.FlagSelection,
	selectedFlagsByAtom map[string][]string,
	packageUsePaths []string,
	maskActions *InstallMaskActions,
	requiredUseChanges []portage.RequiredUseChange,
	transaction *portage.MergeTransaction,
	pretendResult *portage.PretendResult,
	pretendErr error,
	t *i18n.Translator,
) {
	action := "Install " + strings.Join(atoms, " ")

	p := plan.Plan{
		TitleKey: "plan_title",
		Action:   action,
		Will: []plan.Item{
			{
				Key: "will_inspect_use_flags",
				Data: map[string]any{
					"Atom": strings.Join(atoms, " "),
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
					"Atom": strings.Join(atoms, " "),
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
					Atom:    atoms[0],
					Command: "install",
				}),
			},
		},
	}

	fmt.Println("Portico Install")
	fmt.Println()
	fmt.Println("Packages:")

	for _, atom := range atoms {
		fmt.Printf("  %s\n", atom)
	}

	for _, queryResult := range queryResults {
		fmt.Println()
		renderUseFlagSummary(queryResult)
	}

	renderInstallMaskActions(maskActions)
	renderRequiredUseChanges(requiredUseChanges)

	fmt.Println()
	fmt.Println("Selected package.use entries:")

	hasSelectedFlags := false
	for _, atom := range atoms {
		selectedFlags := selectedFlagsByAtom[atom]
		if len(selectedFlags) == 0 {
			continue
		}

		hasSelectedFlags = true
		fmt.Printf("  %s %s\n", atom, strings.Join(selectedFlags, " "))
	}

	if !hasSelectedFlags {
		fmt.Println("  No explicit USE flag changes selected.")
	}

	if len(packageUsePaths) > 0 {
		fmt.Println()
		fmt.Println("Sandbox package.use paths:")

		for _, path := range dedupeStrings(packageUsePaths) {
			fmt.Printf("  %s\n", path)
		}
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

	_ = selectionsByAtom
}

func renderInstallMaskActions(actions *InstallMaskActions) {
	if actions == nil {
		return
	}

	if len(actions.KeywordEntries) > 0 {
		fmt.Println()
		fmt.Println("Sandbox package.accept_keywords entries:")

		for _, entry := range actions.KeywordEntries {
			fmt.Printf("  %s %s\n", entry.Atom, entry.Keyword)
		}

		if actions.KeywordEntries[0].Path != "" {
			fmt.Println("Sandbox package.accept_keywords path:")
			fmt.Printf("  %s\n", actions.KeywordEntries[0].Path)
		}
	}

	if len(actions.LicenseEntries) > 0 {
		fmt.Println()
		fmt.Println("Sandbox package.license entries:")

		for _, entry := range actions.LicenseEntries {
			fmt.Printf("  %s %s\n", entry.Atom, strings.Join(entry.Licenses, " "))
		}

		if actions.LicenseEntries[0].Path != "" {
			fmt.Println("Sandbox package.license path:")
			fmt.Printf("  %s\n", actions.LicenseEntries[0].Path)
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

func cleanInstallArgs(args []string) []string {
	var out []string
	seen := make(map[string]bool)

	for _, arg := range args {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}

		if seen[arg] {
			continue
		}

		seen[arg] = true
		out = append(out, arg)
	}

	return out
}

func dedupeStrings(values []string) []string {
	var out []string
	seen := make(map[string]bool)

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if seen[value] {
			continue
		}

		seen[value] = true
		out = append(out, value)
	}

	return out
}
