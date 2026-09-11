package portage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAcceptKeywordEntryPreservesKeywordToken(t *testing.T) {
	dir := t.TempDir()

	path, err := WriteAcceptKeywordEntry(dir, "=dev-libs/foo-1.2.3", "~arm64")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedPath := filepath.Join(dir, "package.accept_keywords", PorticoAcceptKeywordsFile)
	if path != expectedPath {
		t.Fatalf("expected path %q, got %q", expectedPath, path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected to read accept_keywords file, got %v", err)
	}

	want := "=dev-libs/foo-1.2.3 ~arm64\n"
	got := string(content)

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestWriteAcceptKeywordEntryPreservesStableKeywordToken(t *testing.T) {
	dir := t.TempDir()

	path, err := WriteAcceptKeywordEntry(dir, "=dev-libs/foo-1.2.3", "riscv")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected to read accept_keywords file, got %v", err)
	}

	want := "=dev-libs/foo-1.2.3 riscv\n"
	got := string(content)

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
