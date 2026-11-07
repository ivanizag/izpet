package izpet

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	defaultModel = "3032"

	// SpeedReal runs at the speed of the PET, SpeedFull as fast as possible
	SpeedReal = "real"
	SpeedFull = "full"
)

// Configuration holds the options to build a PET
type Configuration struct {
	Model string
	RamKB int
	Hz50  bool
	Speed string
	Trace string
	Rom9  string // Option ROM at $9000
	RomA  string // Option ROM at $a000

	cycleDurationNs float64
	optionRoms      []romChip
	programs        []string
}

// NewConfiguration returns the default configuration
func NewConfiguration() *Configuration {
	return &Configuration{
		Model:           defaultModel,
		Speed:           SpeedReal,
		cycleDurationNs: 1000.0 / CPUClockMhz,
	}
}

// ParseFlags fills the configuration from the command line arguments. The
// arguments left after the options are programs to load.
func (c *Configuration) ParseFlags(name string, args []string, output io.Writer) error {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(output)
	c.AddFlags(fs)

	err := fs.Parse(args)
	if err != nil {
		return err
	}

	c.AddFiles(fs.Args())
	return c.Validate()
}

// IsHelpRequested tells whether parsing the flags failed only because the
// help was asked for
func IsHelpRequested(err error) bool {
	return errors.Is(err, flag.ErrHelp)
}

// AddFlags adds the options to a flag set, for frontends with options of
// their own
func (c *Configuration) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Model, "model", c.Model,
		"model to emulate, one of:\n"+ModelsHelp())
	fs.IntVar(&c.RamKB, "ram", c.RamKB,
		"RAM in KB, up to 32. The default depends on the model")
	fs.BoolVar(&c.Hz50, "50hz", c.Hz50,
		"use the 50 Hz editor ROM, for the CRTC models")
	fs.StringVar(&c.Speed, "speed", c.Speed,
		"'real', 'full' or the speed in MHz")
	fs.StringVar(&c.Trace, "trace", c.Trace,
		"comma separated list of tracers: cpu")
	fs.StringVar(&c.Rom9, "rom9", c.Rom9,
		"option ROM at $9000")
	fs.StringVar(&c.RomA, "roma", c.RomA,
		"option ROM at $a000")
}

// AddFiles adds the files given without option, programs to load
func (c *Configuration) AddFiles(filenames []string) {
	c.programs = append(c.programs, filenames...)
}

// Validate checks the options and prepares them
func (c *Configuration) Validate() error {
	if _, err := getModel(c.Model); err != nil {
		return err
	}

	switch c.Speed {
	case SpeedReal, "":
		c.cycleDurationNs = 1000.0 / CPUClockMhz
	case SpeedFull:
		c.cycleDurationNs = 0
	default:
		mhz, err := strconv.ParseFloat(c.Speed, 64)
		if err != nil || mhz <= 0 {
			return fmt.Errorf("invalid speed '%v', use '%v', '%v' or a number of MHz",
				c.Speed, SpeedReal, SpeedFull)
		}
		c.cycleDurationNs = 1000.0 / mhz
	}

	c.optionRoms = nil
	if c.Rom9 != "" {
		c.optionRoms = append(c.optionRoms, romChip{0x9000, c.Rom9})
	}
	if c.RomA != "" {
		c.optionRoms = append(c.optionRoms, romChip{0xa000, c.RomA})
	}
	return nil
}

// hasTracer returns true when the given tracer is listed in the trace option
func (c *Configuration) hasTracer(name string) bool {
	for _, v := range strings.Split(c.Trace, ",") {
		if strings.TrimSpace(v) == name {
			return true
		}
	}
	return false
}
