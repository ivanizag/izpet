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

type commandLoadFile struct {
	path string
}

func (c *commandLoadFile) getId() int {
	return CommandComplex
}

type commandPaste struct {
	text string
}

func (c *commandPaste) getId() int {
	return CommandComplex
}

// SendCommand enqueues a command to the emulator goroutine
func (p *PET) SendCommand(commandId int) {
	p.commandChannel <- &commandSimple{commandId}
}

// SendLoadFile loads a program, a tape or a disk from another goroutine,
// see LoadFile, reporting the errors on the console
func (p *PET) SendLoadFile(path string) {
	p.commandChannel <- &commandLoadFile{path}
}

// SendPaste types a text from another goroutine, see paste
func (p *PET) SendPaste(text string) {
	p.commandChannel <- &commandPaste{text}
}

// drainCommands executes the queued commands. It returns if the emulation
// has to stop and if the clock reference has to be reset.
func (p *PET) drainCommands() (kill bool, resync bool) {
	for {
		select {
		case c := <-p.commandChannel:
			switch c.getId() {
			case CommandKill:
				if err := p.Close(); err != nil {
					fmt.Println(err)
				}
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
				switch c := c.(type) {
				case *commandLoadFile:
					if err := p.LoadFile(c.path); err != nil {
						fmt.Printf("Error loading %v: %v\n", c.path, err)
					}
				case *commandPaste:
					p.paste(c.text)
				}
			}
		default:
			return false, resync
		}
	}
}
