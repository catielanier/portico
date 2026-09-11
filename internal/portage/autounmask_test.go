package portage

import "testing"

func TestParseAutounmaskReportPreservesKeywordTokens(t *testing.T) {
	raw := `
The following keyword changes are necessary to proceed:
# required by app-editors/example-1.0::gentoo
=dev-libs/foo-1.2.3 ~arm64
=dev-libs/bar-4.5.6 riscv
=dev-libs/baz-7.8.9 ppc64
=dev-libs/qux-1.0.0 ~x86
`

	report := ParseAutounmaskReport(raw)
	if report == nil {
		t.Fatal("expected autounmask report")
	}

	got := report.RequiredKeywordChanges
	if len(got) != 4 {
		t.Fatalf("expected 4 keyword changes, got %d", len(got))
	}

	expected := []struct {
		atom    string
		keyword string
	}{
		{"=dev-libs/foo-1.2.3", "~arm64"},
		{"=dev-libs/bar-4.5.6", "riscv"},
		{"=dev-libs/baz-7.8.9", "ppc64"},
		{"=dev-libs/qux-1.0.0", "~x86"},
	}

	for i, want := range expected {
		if got[i].Atom != want.atom {
			t.Fatalf("change %d atom: expected %q, got %q", i, want.atom, got[i].Atom)
		}

		if len(got[i].Keywords) != 1 {
			t.Fatalf("change %d keywords: expected 1 keyword, got %d", i, len(got[i].Keywords))
		}

		if got[i].Keywords[0] != want.keyword {
			t.Fatalf("change %d keyword: expected %q, got %q", i, want.keyword, got[i].Keywords[0])
		}
	}
}
