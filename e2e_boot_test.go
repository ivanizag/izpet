package izpet

import (
	"testing"
)

func TestModelsBoot(t *testing.T) {
	cases := map[string]string{
		"2001":  "*** COMMODORE BASIC ***",
		"3032":  "### COMMODORE BASIC ###",
		"3032b": "### COMMODORE BASIC ###",
		"4016":  "*** COMMODORE BASIC 4.0 ***",
		"4032b": "*** COMMODORE BASIC 4.0 ***",
		"4032":  "*** COMMODORE BASIC 4.0 ***",
		"8032":  "*** COMMODORE BASIC 4.0 ***",
	}
	if len(cases) != len(petModels()) {
		t.Errorf("there are %v models and %v tested", len(petModels()), len(cases))
	}

	for model, banner := range cases {
		t.Run(model, func(t *testing.T) {
			t.Parallel()
			p := newTestPET(t, model)
			testWaitText(t, p, banner)
			testWaitText(t, p, "READY.")
			testType(t, p, "PRINT 6*7\r")
			testWaitText(t, p, " 42")
		})
	}
}

func TestModels50Hz(t *testing.T) {
	for _, model := range []string{"4032", "8032"} {
		t.Run(model, func(t *testing.T) {
			config := NewConfiguration()
			config.Model = model
			config.Hz50 = true
			p, err := NewPET(config)
			if err != nil {
				t.Fatal(err)
			}
			testWaitText(t, p, "READY.")

			// Count the frames in a second
			start := p.GetFrames()
			cycles := p.GetCycles()
			p.RunUntil(100, func() bool { return p.GetCycles()-cycles >= 1_000_000 })
			frames := p.GetFrames() - start
			if frames < 49 || frames > 51 {
				t.Errorf("got %v frames per second, expected 50", frames)
			}
		})
	}
}

func TestFrameRate60Hz(t *testing.T) {
	for _, model := range []string{"3032", "4032", "8032"} {
		p := newTestPET(t, model)
		testWaitText(t, p, "READY.")
		start := p.GetFrames()
		cycles := p.GetCycles()
		p.RunUntil(100, func() bool { return p.GetCycles()-cycles >= 1_000_000 })
		frames := p.GetFrames() - start
		if frames < 59 || frames > 61 {
			t.Errorf("%v: got %v frames per second, expected 60", model, frames)
		}
	}
}

func TestShiftedKeys(t *testing.T) {
	// The quotes and the dollar are shifted keys on the business keyboard
	// and not on the graphics one
	for _, model := range []string{"3032", "8032"} {
		t.Run(model, func(t *testing.T) {
			p := newTestPET(t, model)
			testWaitText(t, p, "READY.")
			testType(t, p, "A$=\"AB\"+CHR$(67):PRINT A$;LEN(A$)\r")
			testWaitText(t, p, "ABC 3")
		})
	}
}

func TestLoadBasicProgram(t *testing.T) {
	// 10 PRINT "IZPET";6*7
	program := []uint8{
		0x13, 0x04, // Next line
		10, 0, // Line number
		0x99, ' ', '"', 'I', 'Z', 'P', 'E', 'T', '"', ';', '6', 0xac, '7', 0,
		0, 0, // End of program
	}
	for _, model := range []string{"2001", "3032", "8032"} {
		t.Run(model, func(t *testing.T) {
			p := newTestPET(t, model)
			err := p.LoadProgram(testWritePrg(t, basicStart, program))
			if err != nil {
				t.Fatal(err)
			}
			testWaitText(t, p, "IZPET 42")
			testType(t, p, "LIST\r")
			testWaitText(t, p, "10 PRINT \"IZPET\";6*7")
		})
	}
}

func TestSound(t *testing.T) {
	p := newTestPET(t, "8032")
	var sink testAudioSink
	p.SetAudioSink(&sink)
	testWaitText(t, p, "READY.")
	sink.reset()
	p.RunFrames(10)
	if sink.peak() > 0.01 {
		t.Errorf("there should be silence, got a peak of %v", sink.peak())
	}

	testType(t, p, "POKE 59467,16:POKE 59466,15:POKE 59464,100\r")
	p.RunFrames(5)
	sink.reset()
	p.RunFrames(10)
	if sink.peak() < 0.1 {
		t.Errorf("there should be sound, got a peak of %v", sink.peak())
	}
}

type testAudioSink struct {
	samples []float32
}

func (s *testAudioSink) PushSample(sample float32) {
	s.samples = append(s.samples, sample)
}

func (s *testAudioSink) reset() {
	s.samples = nil
}

func (s *testAudioSink) peak() float32 {
	peak := float32(0)
	for _, sample := range s.samples {
		peak = max(peak, sample, -sample)
	}
	return peak
}
