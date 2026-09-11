package cli

const defaultVersion = "unstable"

var version = defaultVersion

func Version() string {
	if version == "" {
		return defaultVersion
	}

	return version
}