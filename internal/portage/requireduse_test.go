package portage

import (
	"reflect"
	"testing"
)

func TestRequiredUseWineWow64Conflict(t *testing.T) {
	expression, err := ParseRequiredUseExpression("wow64? ( !arm64? ( abi_x86_64 !abi_x86_32 ) )")
	if err != nil {
		t.Fatalf("ParseRequiredUseExpression returned error: %v", err)
	}

	enabled := map[string]bool{
		"wow64":      true,
		"arm64":      false,
		"abi_x86_64": true,
		"abi_x86_32": true,
	}

	violations := expression.Violations(enabled)
	if len(violations) != 1 {
		t.Fatalf("expected one violation, got %d: %#v", len(violations), violations)
	}

	violation := violations[0]
	if violation.Kind != RequiredUseFlag {
		t.Fatalf("expected flag violation, got %q", violation.Kind)
	}
	if violation.Flag != "abi_x86_32" || !violation.Negated {
		t.Fatalf("expected abi_x86_32 to be required disabled, got %#v", violation)
	}

	expectedContext := []RequiredUseCondition{
		{Flag: "wow64", Enabled: true},
		{Flag: "arm64", Enabled: false},
	}
	if !reflect.DeepEqual(violation.Context, expectedContext) {
		t.Fatalf("unexpected context: got %#v want %#v", violation.Context, expectedContext)
	}

	enabled["wow64"] = false
	if got := expression.Violations(enabled); len(got) != 0 {
		t.Fatalf("expected disabling wow64 to satisfy expression, got %#v", got)
	}
}

func TestRequiredUseExactlyOne(t *testing.T) {
	expression, err := ParseRequiredUseExpression("^^ ( foo bar baz )")
	if err != nil {
		t.Fatalf("ParseRequiredUseExpression returned error: %v", err)
	}

	violations := expression.Violations(map[string]bool{
		"foo": true,
		"bar": true,
	})
	if len(violations) != 1 {
		t.Fatalf("expected one violation, got %#v", violations)
	}
	if violations[0].Kind != RequiredUseExactlyOne {
		t.Fatalf("expected exactly-one violation, got %#v", violations[0])
	}
	if !violations[0].SimpleFlags {
		t.Fatalf("expected simple flags, got %#v", violations[0])
	}
}

func TestRequiredUseAnyOf(t *testing.T) {
	expression, err := ParseRequiredUseExpression("|| ( foo bar )")
	if err != nil {
		t.Fatalf("ParseRequiredUseExpression returned error: %v", err)
	}

	if expression.Satisfied(map[string]bool{}) {
		t.Fatal("expected expression to be unsatisfied with neither flag enabled")
	}

	if !expression.Satisfied(map[string]bool{"bar": true}) {
		t.Fatal("expected expression to be satisfied with bar enabled")
	}
}

func TestRequiredUseAtMostOne(t *testing.T) {
	expression, err := ParseRequiredUseExpression("?? ( foo bar baz )")
	if err != nil {
		t.Fatalf("ParseRequiredUseExpression returned error: %v", err)
	}

	if !expression.Satisfied(map[string]bool{"foo": true}) {
		t.Fatal("expected one enabled flag to satisfy at-most-one")
	}

	if expression.Satisfied(map[string]bool{"foo": true, "baz": true}) {
		t.Fatal("expected two enabled flags to violate at-most-one")
	}
}

func TestRequiredUsePortageHumanReadableOperators(t *testing.T) {
	expression, err := ParseRequiredUseExpression("exactly-one-of ( foo bar ) at-most-one-of ( baz qux )")
	if err != nil {
		t.Fatalf("ParseRequiredUseExpression returned error: %v", err)
	}

	if !expression.Satisfied(map[string]bool{"foo": true, "baz": true}) {
		t.Fatal("expected Portage human-readable operators to parse and evaluate")
	}
}

func TestParseRequiredUseFailure(t *testing.T) {
	raw := `!!! The ebuild selected to satisfy "app-emulation/wine-vanilla" has unmet requirements.
- app-emulation/wine-vanilla-10.15::gentoo USE="abi_x86_32 abi_x86_64 wow64 -arm64"

  The following REQUIRED_USE flag constraints are unsatisfied:
    wow64? ( !arm64? ( abi_x86_64 !abi_x86_32 ) )

  The above constraints are a subset of the following complete expression:
    wow64? ( !arm64? ( abi_x86_64 !abi_x86_32 ) )

(dependency required by "app-emulation/wine-vanilla" [argument])`

	failure := ParseRequiredUseFailure(raw)
	if failure == nil {
		t.Fatal("expected REQUIRED_USE failure")
	}

	if failure.Package != "app-emulation/wine-vanilla" {
		t.Fatalf("unexpected package: %q", failure.Package)
	}
	if failure.ExactAtom != "=app-emulation/wine-vanilla-10.15::gentoo" {
		t.Fatalf("unexpected exact atom: %q", failure.ExactAtom)
	}
	if failure.Expression() != "wow64? ( !arm64? ( abi_x86_64 !abi_x86_32 ) )" {
		t.Fatalf("unexpected expression: %q", failure.Expression())
	}
	if !failure.CurrentUse["abi_x86_32"] || failure.CurrentUse["arm64"] {
		t.Fatalf("unexpected USE state: %#v", failure.CurrentUse)
	}
	if len(failure.RequiredBy) != 1 || failure.RequiredBy[0] != "app-emulation/wine-vanilla" {
		t.Fatalf("unexpected required-by chain: %#v", failure.RequiredBy)
	}
}

func TestPackageNameFromSpec(t *testing.T) {
	cases := map[string]string{
		"app-emulation/wine-vanilla":                    "app-emulation/wine-vanilla",
		"=app-emulation/wine-vanilla-10.15::gentoo":     "app-emulation/wine-vanilla",
		">=media-video/ffmpeg-8.1.2:0/60.62.62::gentoo": "media-video/ffmpeg",
		"mail-client/mailspring-bin-1.23.0::guru":       "mail-client/mailspring-bin",
	}

	for input, expected := range cases {
		if got := PackageNameFromSpec(input); got != expected {
			t.Errorf("PackageNameFromSpec(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestParseRequiredUseFailureWithoutCompleteExpression(t *testing.T) {
	raw := `!!! The ebuild selected to satisfy "app-alternatives/yacc" has unmet requirements.
- app-alternatives/yacc-1-r2::gentoo USE="-bison -byacc -reference" ABI_X86="(64)"

  The following REQUIRED_USE flag constraints are unsatisfied:
    exactly-one-of ( bison byacc reference )`

	failure := ParseRequiredUseFailure(raw)
	if failure == nil {
		t.Fatal("expected REQUIRED_USE failure")
	}
	if failure.Expression() != "exactly-one-of ( bison byacc reference )" {
		t.Fatalf("unexpected expression: %q", failure.Expression())
	}

	expression, err := ParseRequiredUseExpression(failure.Expression())
	if err != nil {
		t.Fatalf("failed to parse Portage human-readable REQUIRED_USE: %v", err)
	}
	if expression.Satisfied(failure.CurrentUse) {
		t.Fatal("expected reported USE state to violate exactly-one-of")
	}
}
