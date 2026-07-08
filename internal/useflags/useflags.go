package useflags

import "github.com/catielanier/portico/internal/portage"

type SelectionState string

const (
	SelectionUnset    SelectionState = "unset"
	SelectionEnabled  SelectionState = "enabled"
	SelectionDisabled SelectionState = "disabled"
)

type FlagSelection struct {
	Name           string
	Description    string
	CurrentEnabled bool
	Installed      *bool
	Selection      SelectionState
}

func FromQuery(query *portage.PackageQuery) []FlagSelection {
	selections := make([]FlagSelection, 0, len(query.Uses))

	for _, flag := range query.Uses {
		selections = append(selections, FlagSelection{
			Name:           flag.Name,
			Description:    flag.Description,
			CurrentEnabled: flag.EnabledForBuild,
			Installed:      flag.Installed,
			Selection:      SelectionUnset,
		})
	}

	return selections
}

func (s SelectionState) Next() SelectionState {
	switch s {
	case SelectionUnset:
		return SelectionEnabled
	case SelectionEnabled:
		return SelectionDisabled
	default:
		return SelectionUnset
	}
}

func (s SelectionState) Prefix(flagName string) string {
	switch s {
	case SelectionEnabled:
		return flagName
	case SelectionDisabled:
		return "-" + flagName
	default:
		return ""
	}
}

func SelectedFlags(selections []FlagSelection) []string {
	var out []string

	for _, selection := range selections {
		prefix := selection.Selection.Prefix(selection.Name)
		if prefix == "" {
			continue
		}

		out = append(out, prefix)
	}

	return out
}
