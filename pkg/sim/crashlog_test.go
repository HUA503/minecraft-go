package sim

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAppendSimulationCrashLogUsesUserConfigDirectory(t *testing.T) {
	configRoot := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", configRoot)
	} else {
		t.Setenv("XDG_CONFIG_HOME", configRoot)
	}

	appendSimulationCrashLog("SIMULATION TICK LOOP PANIC: regression marker\nstack trace")

	path := filepath.Join(configRoot, "minecraft-go", "crash.log")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read crash log: %v", err)
	}
	for _, want := range []string{"SIMULATION TICK LOOP PANIC: regression marker", "stack trace"} {
		if !strings.Contains(string(contents), want) {
			t.Errorf("crash log missing %q: %s", want, contents)
		}
	}
}

