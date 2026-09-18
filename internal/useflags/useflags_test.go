package useflags

import "testing"

func TestEffectiveEnabledMapUsesCurrentStateForUnsetSelections(t *testing.T) {
	selections := []FlagSelection{
		{Name: "foo", CurrentEnabled: true, Selection: SelectionUnset},
		{Name: "bar", CurrentEnabled: false, Selection: SelectionUnset},
	}

	enabled := EffectiveEnabledMap(selections)
	if !enabled["foo"] {
		t.Fatal("expected current enabled state for foo")
	}
	if enabled["bar"] {
		t.Fatal("expected current disabled state for bar")
	}
}

func TestEffectiveEnabledMapAppliesExplicitSelections(t *testing.T) {
	selections := []FlagSelection{
		{Name: "foo", CurrentEnabled: true, Selection: SelectionDisabled},
		{Name: "bar", CurrentEnabled: false, Selection: SelectionEnabled},
	}

	enabled := EffectiveEnabledMap(selections)
	if enabled["foo"] {
		t.Fatal("expected explicit disable to override current state")
	}
	if !enabled["bar"] {
		t.Fatal("expected explicit enable to override current state")
	}
}
