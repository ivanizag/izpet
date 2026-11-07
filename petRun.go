package izpet

import (
	"math"
	"time"
)

const (
	// CPUClockMhz is the clock of all the PET models
	CPUClockMhz = 1.0

	maxWaitDuration = 100 * time.Millisecond
	cpuSpinLoops    = 100
)

// Run starts the emulation, until CommandKill is sent
func (p *PET) Run() {
	p.reset()

	referenceTime := time.Now()
	speedReferenceTime := referenceTime
	speedReferenceCycles := uint64(0)

	for {
		if !p.paused.Load() {
			for i := 0; i < cpuSpinLoops; i++ {
				p.stepInstruction()
			}
		} else {
			time.Sleep(200 * time.Millisecond)
		}

		kill, resync := p.drainCommands()
		if kill {
			return
		}
		if resync {
			referenceTime = time.Now().Add(-time.Duration(float64(p.cycles) * p.cycleDurationNs))
		}

		if p.cycleDurationNs != 0 && !p.isTapeMoving() {
			// Wait until the next steps have to run
			clockDuration := time.Since(referenceTime)
			simulatedDuration := time.Duration(float64(p.cycles) * p.cycleDurationNs)
			waitDuration := simulatedDuration - clockDuration
			if waitDuration > maxWaitDuration || -waitDuration > maxWaitDuration {
				// Too much ahead or behind, resynchronize
				referenceTime = referenceTime.Add(-waitDuration)
				waitDuration = 0
			}
			if waitDuration > 0 {
				time.Sleep(waitDuration)
			}
		}

		if p.cycles-speedReferenceCycles > 1_000_000 {
			// Calculate speed in MHz every million cycles
			newTime := time.Now()
			elapsedCycles := float64(p.cycles - speedReferenceCycles)
			freq := 1000.0 * elapsedCycles / float64(newTime.Sub(speedReferenceTime).Nanoseconds())
			p.currentFreqMHz.Store(math.Float64bits(freq))
			speedReferenceTime = newTime
			speedReferenceCycles = p.cycles
		}
	}
}

// RunFrames runs the emulation for a number of frames as fast as it can. The
// first call resets the machine, the next ones continue. It is used by the
// headless frontend and the tests.
func (p *PET) RunFrames(frames uint64) {
	if p.cycles == 0 {
		p.reset()
	}
	target := p.frames + frames
	for p.frames < target {
		kill, _ := p.drainCommands()
		if kill {
			return
		}
		p.runFrame()
	}
}

// RunUntil runs frames until the condition is true, up to a number of
// frames. It returns true if the condition was met.
func (p *PET) RunUntil(maxFrames uint64, condition func() bool) bool {
	for i := uint64(0); i < maxFrames; i++ {
		p.RunFrames(1)
		if condition() {
			return true
		}
	}
	return false
}

func (p *PET) runFrame() {
	frame := p.frames
	for frame == p.frames {
		p.stepInstruction()
	}
}

// GetCurrentFreqMHz returns the speed the emulation is reaching
func (p *PET) GetCurrentFreqMHz() float64 {
	return math.Float64frombits(p.currentFreqMHz.Load())
}

func (p *PET) stepInstruction() {
	startCycles := p.cpu.GetCycles()
	p.cpu.ExecuteInstruction()
	cycles := p.cpu.GetCycles() - startCycles
	p.cycles += cycles

	p.io.tick(cycles)
	p.sound.tick(cycles, p.io.via.GetCB2())
	if p.cassettes[0].inserted || p.cassettes[1].inserted {
		p.tickCassettes(cycles)
	}
	if p.bus.IsWaiting() {
		p.bus.Tick(cycles)
	}
	if p.video.tick(cycles) {
		p.endFrame()
	}
	p.io.updateRetrace()
	p.cpu.SetIRQ(p.io.irq())
}

func (p *PET) tickCassettes(cycles uint64) {
	motor1, motor2, write := p.io.cassetteLines()
	p.cassettes[0].tick(cycles, motor1, write)
	p.cassettes[1].tick(cycles, motor2, write)
	p.io.pia1.SetCA1(p.cassettes[0].level)
	p.io.via.SetCB1(p.cassettes[1].level)
}

func (p *PET) endFrame() {
	p.frames++
	p.video.takeSnapshot(p.mmu.screenRAM(), p.io.isLowerCase())
	p.keyboard.frame()
	p.loadPendingPrograms()
}

func (p *PET) reset() {
	p.io.reset()
	p.bus.Reset()
	p.cpu.Reset()
}
