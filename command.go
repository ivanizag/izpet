package izpet

import "fmt"

const (
	// CommandToggleSpeed toggles cpu speed between full speed and actual PET speed
	CommandToggleSpeed = iota + 1
	// CommandShowSpeed prints the current speed in Mhz
	CommandShowSpeed
	// CommandToggleCPUTrace toggles tracing of the CPU execution
	CommandToggleCPUTrace
	// CommandKill stops the cpu execution loop
	CommandKill
	// CommandReset executes a 6502 reset
	CommandReset
	// CommandPauseUnpause toggles the pause
	CommandPauseUnpause
	// CommandPause pauses the emulator
	CommandPause
	// CommandStart restarts the emulator
	CommandStart
	// CommandComplex for commands that use a struct with parameters
	CommandComplex
)

type command interface {
	getId() int
}

type commandSimple struct {
	id int
}

func (c *commandSimple) getId() int {
	return c.id
}

type commandLoadProgram struct {
	path string
}

func (c *commandLoadProgram) getId() int {
	return CommandComplex
}

// SendCommand enqueues a command to the emulator goroutine
func (p *PET) SendCommand(commandId int) {
	p.commandChannel <- &commandSimple{commandId}
}

// SendLoadProgram loads a program from another goroutine, reporting the
// errors on the console
func (p *PET) SendLoadProgram(path string) {
	p.commandChannel <- &commandLoadProgram{path}
}

// drainCommands executes the queued commands. It returns if the emulation
// has to stop and if the clock reference has to be reset.
func (p *PET) drainCommands() (kill bool, resync bool) {
	for {
		select {
		case c := <-p.commandChannel:
			switch c.getId() {
			case CommandKill:
				return true, resync
			case CommandPause:
				p.paused.Store(true)
			case CommandStart:
				if p.paused.Load() {
					p.paused.Store(false)
					resync = true
				}
			case CommandPauseUnpause:
				p.paused.Store(!p.paused.Load())
				resync = true
			case CommandToggleSpeed:
				if p.cycleDurationNs == 0 {
					p.cycleDurationNs = 1000.0 / CPUClockMhz
				} else {
					p.cycleDurationNs = 0
				}
				resync = true
			case CommandShowSpeed:
				fmt.Printf("Speed: %.2f MHz\n", p.GetCurrentFreqMHz())
			case CommandToggleCPUTrace:
				p.cpuTrace = !p.cpuTrace
				p.cpu.SetTrace(p.cpuTrace)
			case CommandReset:
				p.reset()
			case CommandComplex:
				if load, ok := c.(*commandLoadProgram); ok {
					if err := p.LoadProgram(load.path); err != nil {
						fmt.Printf("Error loading %v: %v\n", load.path, err)
					}
				}
			}
		default:
			return false, resync
		}
	}
}
