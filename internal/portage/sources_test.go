package portage

import "testing"

func TestParsePackageSourceReportFromMaskedEmergeOutput(t *testing.T) {
	raw := `
These are the packages that would be merged, in order:

!!! All ebuilds that could satisfy "mail-client/mailspring-bin" have been masked.
!!! One of the following masked packages is required to complete your request:
- mail-client/mailspring-bin-1.23.0::guru (masked by: ~amd64 keyword)
- mail-client/mailspring-bin-1.23.0::edgets (masked by: ~amd64 keyword)
- mail-client/mailspring-bin-1.22.0::edgets (masked by: ~amd64 keyword)
- mail-client/mailspring-bin-1.21.1::edgets (masked by: ~amd64 keyword)
- mail-client/mailspring-bin-1.21.0::edgets (masked by: ~amd64 keyword)
`

	report := ParsePackageSourceReport("mail-client/mailspring-bin", raw)
	if report == nil {
		t.Fatal("expected source report")
	}

	if len(report.Candidates) != 2 {
		t.Fatalf("expected 2 latest-per-repo candidates, got %d", len(report.Candidates))
	}

	assertPackageSourceCandidate(t, report.Candidates[0], PackageSourceCandidate{
		Package:       "mail-client/mailspring-bin",
		Version:       "1.23.0",
		Repository:    "guru",
		InstallTarget: "=mail-client/mailspring-bin-1.23.0::guru",
		Masked:        true,
		MaskReason:    "~amd64 keyword",
		Live:          false,
	})

	assertPackageSourceCandidate(t, report.Candidates[1], PackageSourceCandidate{
		Package:       "mail-client/mailspring-bin",
		Version:       "1.23.0",
		Repository:    "edgets",
		InstallTarget: "=mail-client/mailspring-bin-1.23.0::edgets",
		Masked:        true,
		MaskReason:    "~amd64 keyword",
		Live:          false,
	})
}

func TestParsePackageSourceReportKeepsLatestPerRepositoryFromPortageOrder(t *testing.T) {
	raw := `
!!! All ebuilds that could satisfy "app-misc/example" have been masked.
- app-misc/example-3.0.0::local (masked by: ~amd64 keyword)
- app-misc/example-2.0.0::local (masked by: ~amd64 keyword)
- app-misc/example-1.0.0::local (masked by: ~amd64 keyword)
`

	report := ParsePackageSourceReport("app-misc/example", raw)
	if report == nil {
		t.Fatal("expected source report")
	}

	if len(report.Candidates) != 1 {
		t.Fatalf("expected 1 latest-per-repo candidate, got %d", len(report.Candidates))
	}

	got := report.Candidates[0]
	if got.Version != "3.0.0" {
		t.Fatalf("expected latest version from Portage order to be 3.0.0, got %q", got.Version)
	}
}

func TestNeedsPackageSourceSelection(t *testing.T) {
	report := &PackageSourceReport{
		RequestedAtom: "app-misc/example",
		Candidates: []PackageSourceCandidate{
			{Repository: "gentoo"},
			{Repository: "guru"},
		},
	}

	if !NeedsPackageSourceSelection(report) {
		t.Fatal("expected source selection to be needed")
	}
}

func TestDoesNotNeedPackageSourceSelectionForSingleRepository(t *testing.T) {
	report := &PackageSourceReport{
		RequestedAtom: "app-misc/example",
		Candidates: []PackageSourceCandidate{
			{Repository: "gentoo"},
		},
	}

	if NeedsPackageSourceSelection(report) {
		t.Fatal("expected source selection to be unnecessary")
	}
}

func assertPackageSourceCandidate(
	t *testing.T,
	got PackageSourceCandidate,
	want PackageSourceCandidate,
) {
	t.Helper()

	if got.Package != want.Package {
		t.Fatalf("package: expected %q, got %q", want.Package, got.Package)
	}

	if got.Version != want.Version {
		t.Fatalf("version: expected %q, got %q", want.Version, got.Version)
	}

	if got.Repository != want.Repository {
		t.Fatalf("repository: expected %q, got %q", want.Repository, got.Repository)
	}

	if got.InstallTarget != want.InstallTarget {
		t.Fatalf("install target: expected %q, got %q", want.InstallTarget, got.InstallTarget)
	}

	if got.Masked != want.Masked {
		t.Fatalf("masked: expected %v, got %v", want.Masked, got.Masked)
	}

	if got.MaskReason != want.MaskReason {
		t.Fatalf("mask reason: expected %q, got %q", want.MaskReason, got.MaskReason)
	}

	if got.Live != want.Live {
		t.Fatalf("live: expected %v, got %v", want.Live, got.Live)
	}
}
