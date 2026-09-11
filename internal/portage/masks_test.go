package portage

import "testing"

func TestExtractRequiredKeywordPreservesTestingKeyword(t *testing.T) {
	got := extractRequiredKeyword("~arm64 keyword")
	want := "~arm64"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestExtractRequiredKeywordPreservesStableKeyword(t *testing.T) {
	got := extractRequiredKeyword("riscv keyword")
	want := "riscv"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestExtractRequiredKeywordIgnoresMissingKeyword(t *testing.T) {
	got := extractRequiredKeyword("missing keyword")

	if got != "" {
		t.Fatalf("expected empty keyword, got %q", got)
	}
}

func TestParseMaskedPackageReportPreservesNonAmd64Keyword(t *testing.T) {
	raw := `
!!! All ebuilds that could satisfy "media-video/example" have been masked.
!!! One of the following masked packages is required to complete your request:
- media-video/example-1.2.3::gentoo (masked by: ~arm64 keyword)
`

	report := ParseMaskedPackageReport("media-video/example", raw)
	if report == nil {
		t.Fatal("expected masked package report")
	}

	if len(report.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(report.Candidates))
	}

	got := report.Candidates[0].RequiredKeyword
	want := "~arm64"

	if got != want {
		t.Fatalf("expected required keyword %q, got %q", want, got)
	}
}

func TestParseMaskedPackageReportPreservesStableKeyword(t *testing.T) {
	raw := `
!!! All ebuilds that could satisfy "media-video/example" have been masked.
!!! One of the following masked packages is required to complete your request:
- media-video/example-1.2.3::gentoo (masked by: riscv keyword)
`

	report := ParseMaskedPackageReport("media-video/example", raw)
	if report == nil {
		t.Fatal("expected masked package report")
	}

	if len(report.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(report.Candidates))
	}

	got := report.Candidates[0].RequiredKeyword
	want := "riscv"

	if got != want {
		t.Fatalf("expected required keyword %q, got %q", want, got)
	}
}

func TestClassifyStableKeywordMaskAsSupportedKeywordMask(t *testing.T) {
	reasons := classifyMaskReasons("riscv keyword")
	if len(reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d", len(reasons))
	}

	if reasons[0] != MaskReasonTestingKeyword {
		t.Fatalf("expected %q, got %q", MaskReasonTestingKeyword, reasons[0])
	}
}