package cli

import (
	"testing"

	"github.com/catielanier/portico/internal/portage"
	"github.com/catielanier/portico/internal/useflags"
)

func TestFilterNewRequiredUseChanges(t *testing.T) {
	existing := []portage.RequiredUseChange{
		{Atom: ">=app/example-1", Flags: []string{"foo", "-bar"}},
	}

	incoming := []portage.RequiredUseChange{
		{Atom: ">=app/example-1", Flags: []string{"foo", "-bar"}},
		{Atom: ">=app/other-2", Flags: []string{"baz"}},
	}

	got := filterNewRequiredUseChanges(existing, incoming)
	if len(got) != 1 {
		t.Fatalf("expected 1 new USE change, got %d", len(got))
	}

	if got[0].Atom != ">=app/other-2" {
		t.Fatalf("expected app/other change, got %q", got[0].Atom)
	}
}

func TestUpdateResolutionFingerprintIgnoresOrdering(t *testing.T) {
	first := []portage.RequiredUseChange{
		{Atom: ">=app/a-1", Flags: []string{"foo"}},
		{Atom: ">=app/b-2", Flags: []string{"-bar"}},
	}
	second := []portage.RequiredUseChange{
		{Atom: ">=app/b-2", Flags: []string{"-bar"}},
		{Atom: ">=app/a-1", Flags: []string{"foo"}},
	}

	if updateResolutionFingerprint(first, NewInstallMaskActions()) != updateResolutionFingerprint(second, NewInstallMaskActions()) {
		t.Fatal("expected equivalent resolution state to have the same fingerprint")
	}
}

func TestUpdateResolutionFingerprintChangesWithConfiguration(t *testing.T) {
	actions := NewInstallMaskActions()
	before := updateResolutionFingerprint(nil, actions)

	actions.KeywordEntries = append(actions.KeywordEntries, InstallKeywordEntry{
		Atom:    "=app/example-1",
		Keyword: "~amd64",
	})

	after := updateResolutionFingerprint(nil, actions)
	if before == after {
		t.Fatal("expected keyword configuration to change resolution fingerprint")
	}
}

func TestUpdateHasConfigurationChanges(t *testing.T) {
	if updateHasConfigurationChanges(NewInstallMaskActions(), nil) {
		t.Fatal("expected empty update configuration to report no changes")
	}

	if !updateHasConfigurationChanges(NewInstallMaskActions(), []portage.RequiredUseChange{
		{Atom: "app/example", Flags: []string{"foo"}},
	}) {
		t.Fatal("expected USE change to count as update configuration")
	}
}

func TestApplyRequestedUseFlagsToSelections(t *testing.T) {
	selections := []useflags.FlagSelection{
		{Name: "foo", CurrentEnabled: false},
		{Name: "bar", CurrentEnabled: true},
		{Name: "baz", CurrentEnabled: true},
	}

	applyRequestedUseFlagsToSelections(selections, []string{"foo", "-bar"})

	if selections[0].Selection != useflags.SelectionEnabled {
		t.Fatalf("expected foo to be explicitly enabled, got %q", selections[0].Selection)
	}
	if selections[1].Selection != useflags.SelectionDisabled {
		t.Fatalf("expected bar to be explicitly disabled, got %q", selections[1].Selection)
	}
	if selections[2].Selection != useflags.SelectionUnset {
		t.Fatalf("expected baz to remain unset, got %q", selections[2].Selection)
	}
}
