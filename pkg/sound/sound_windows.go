//go:build windows

package sound

import (
	"sync"
	"syscall"
)

var (
	kernel32   = syscall.NewLazyDLL("kernel32.dll")
	procBeep   = kernel32.NewProc("Beep")
	soundChan  = make(chan string, 16)
	soundOnce  sync.Once
)

func initSoundWorker() {
	go func() {
		for name := range soundChan {
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
			case SoundPlayerHurt:
				procBeep.Call(220, 80)
			default:
				procBeep.Call(400, 20)
			}
		}
	}()
}

// PlayHardwareFeedback emits non-blocking sound feedback on Windows via a dedicated single worker.
func PlayHardwareFeedback(name string) {
	soundOnce.Do(initSoundWorker)
	select {
	case soundChan <- name:
	default:
		// Queue full: skip to prevent thread explosion and syscall deadlocks
	}
}
