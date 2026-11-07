package izpet

/*
The sound of the PET is the CB2 line of the VIA. Programs set the shift
register to shift out freely at the rate of timer 2, and the bit pattern
written to it goes round making a tone. The 12" models have a speaker on it,
the 2001 needed one wired to the user port.

The level of CB2 is averaged over each sample. A DC blocker removes the
constant level of an idle line.

See PET-Interfaces.txt for the three POKEs: 59467 the mode, 59466 the
pattern and 59464 the rate.
*/

const (
	soundCyclesPerSample = 20

	// SoundSampleRateHz is the rate the samples are produced at
	SoundSampleRateHz = CPUClockMhz * 1_000_000 / soundCyclesPerSample

	soundVolume      = 0.3
	soundDCBlockPole = 0.995
)

// AudioSink receives the samples of the sound, on the emulation goroutine
type AudioSink interface {
	PushSample(sample float32)
}

type sound struct {
	sink AudioSink

	cycles   uint64 // Cycles accumulated for the next sample
	highTime uint64 // Of them, with CB2 high

	previousIn  float32
	previousOut float32
}

// SetAudioSink sets the receiver of the sound samples
func (p *PET) SetAudioSink(sink AudioSink) {
	p.sound.sink = sink
}

func (s *sound) tick(cycles uint64, level bool) {
	if s.sink == nil {
		return
	}

	for cycles > 0 {
		step := min(cycles, soundCyclesPerSample-s.cycles)
		s.cycles += step
		if level {
			s.highTime += step
		}
		cycles -= step

		if s.cycles == soundCyclesPerSample {
			in := float32(s.highTime) / soundCyclesPerSample * soundVolume
			out := in - s.previousIn + soundDCBlockPole*s.previousOut
			s.previousIn = in
			s.previousOut = out
			s.sink.PushSample(out)
			s.cycles = 0
			s.highTime = 0
		}
	}
}
