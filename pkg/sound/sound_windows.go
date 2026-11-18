//go:build windows

package sound

import (
	"syscall"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	procBeep = kernel32.NewProc("Beep")
)

// PlayHardwareFeedback emits non-blocking sound feedback on Windows.
func PlayHardwareFeedback(name string) {
	go func() {
		switch name {
		case SoundBlockStoneBreak:
			procBeep.Call(600, 25)
			procBeep.Call(350, 35)
		case SoundBlockWoodBreak:
			procBeep.Call(450, 40)
		case SoundBlockGrassBreak:
			procBeep.Call(320, 30)
		case SoundBlockStonePlace, SoundBlockWoodPlace, SoundBlockGrassPlace:
			procBeep.Call(520, 25)
		case SoundItemToolBreak:
			procBeep.Call(800, 40)
			procBeep.Call(220, 90)
		case "minecraft:block.hit":
			procBeep.Call(700, 15)
		case "minecraft:player.eat":
			procBeep.Call(580, 20)
			procBeep.Call(420, 25)
		case "minecraft:player.burp":
			procBeep.Call(260, 45)
			procBeep.Call(320, 30)
		case SoundBlockGrassStep:
			procBeep.Call(240, 15)
		case SoundBlockStoneStep:
			procBeep.Call(480, 15)
		case SoundBlockWoodStep:
			procBeep.Call(360, 20)
		default:
			procBeep.Call(400, 20)
		}
	}()
}
