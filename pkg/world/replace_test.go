package world

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceFileReplacesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "settings.json")
	src := filepath.Join(dir, "settings.json.tmp")
	if err := os.WriteFile(dst, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceFile(src, dst); err != nil {
		t.Fatalf("replace existing destination: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("destination contents = %q, want new", got)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("replacement source still exists or stat failed unexpectedly: %v", err)
	}
}
