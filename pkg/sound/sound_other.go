//go:build !windows

package sound

func PlayHardwareFeedback(name string) {
	// No-op on non-Windows headless environments
}
