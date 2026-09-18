package portage

import "testing"

func TestRepositoryQualifier(t *testing.T) {
	cases := map[string]string{
		"app-emulation/wine-vanilla":               "",
		"app-emulation/wine-vanilla::gentoo":       "gentoo",
		"=mail-client/mailspring-bin-1.23.0::guru": "guru",
		"media-video/ffmpeg:0/60.62.62::gentoo":    "gentoo",
		"media-video/ffmpeg::gentoo[opus,x264]":    "gentoo",
		"  app-portage/portico::guru  ":            "guru",
	}

	for input, expected := range cases {
		if got := repositoryQualifier(input); got != expected {
			t.Errorf("repositoryQualifier(%q) = %q, want %q", input, got, expected)
		}
	}
}
