package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAutosaveFailureIsVisibleAndRecorded(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("APPDATA", configRoot)
	client := &Client{}
	client.recordAutosaveResult([]error{errors.New("disk full")})
	if !client.autosaveFailed.Load() {
		t.Fatal("autosave failure was not retained for the UI warning")
	}
	logPath := filepath.Join(configRoot, "minecraft-go", "crash.log")
	contents, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read crash log: %v", err)
	}
	if !strings.Contains(string(contents), "PERIODIC AUTOSAVE FAILED: disk full") {
		t.Fatalf("crash log missing autosave failure: %s", contents)
	}
	client.recordAutosaveResult(nil)
	if client.autosaveFailed.Load() {
		t.Fatal("successful autosave did not clear the warning")
	}
}

func TestAutosavePanicIsVisibleAndRecorded(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("APPDATA", configRoot)
	client := &Client{}
	client.recordAutosavePanic("simulated panic")
	if !client.autosaveFailed.Load() {
		t.Fatal("autosave panic was not retained for the UI warning")
	}
	contents, err := os.ReadFile(filepath.Join(configRoot, "minecraft-go", "crash.log"))
	if err != nil {
		t.Fatalf("read crash log: %v", err)
	}
	if !strings.Contains(string(contents), "PERIODIC AUTOSAVE PANIC: simulated panic") {
		t.Fatalf("crash log missing autosave panic: %s", contents)
	}
}
