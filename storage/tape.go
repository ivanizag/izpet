package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

/*
Cassette tapes, as the list of the pulses recorded on them. A pulse is a full
cycle of the signal, from a falling edge to the next, measured in cycles of
the 1 MHz clock of the PET.

The Commodore tapes are recorded with pulses of three lengths, about 350, 510
and 670 µs, and the kernal measures the time between edges to read them.

Formats:
  - TAP, the format of VICE: the header "C64-TAPE-RAW", the version, and a
    byte per pulse with its length in units of 8 cycles. A zero is a pause:
    in version 0 a long one, in version 1 its length in cycles in the next
    three bytes.
  - WAV, a recording of a tape. The pulses are the time between zero
    crossings of the signal going down.

See: https://vice-emu.sourceforge.io/vice_17.html#SEC392
*/

const (
	tapSignature  = "C64-TAPE-RAW"
	tapHeaderSize = 20
	tapUnit       = 8
	tapPetMachine = 3

	// tapLongPause is a version 0 pause, longer than any pulse
	tapLongPause = 256 * tapUnit

	// TapeCyclesPerSecond is the clock the pulse lengths are counted in
	TapeCyclesPerSecond = 1_000_000
)

// IsTapFile returns true for the data of a TAP file
func IsTapFile(data []uint8) bool {
	return len(data) >= tapHeaderSize && string(data[:len(tapSignature)]) == tapSignature
}

// IsWavFile returns true for the data of a WAV file
func IsWavFile(data []uint8) bool {
	return len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WAVE"
}

// LoadTape returns the pulses of a TAP or WAV file
func LoadTape(data []uint8) ([]uint32, error) {
	switch {
	case IsTapFile(data):
		return parseTap(data)
	case IsWavFile(data):
		return parseWavTape(data)
	}
	return nil, errors.New("not a TAP or WAV file")
}

func parseTap(data []uint8) ([]uint32, error) {
	version := data[12]
	if version > 2 {
		return nil, fmt.Errorf("unsupported TAP version %v", version)
	}
	size := int(binary.LittleEndian.Uint32(data[16:20]))
	content := data[tapHeaderSize:]
	if size < len(content) {
		content = content[:size]
	}

	pulses := make([]uint32, 0, len(content))
	for i := 0; i < len(content); i++ {
		v := content[i]
		if v != 0 {
			pulses = append(pulses, uint32(v)*tapUnit)
			continue
		}
		if version == 0 || i+3 >= len(content) {
			pulses = append(pulses, tapLongPause)
			continue
		}
		cycles := uint32(content[i+1]) | uint32(content[i+2])<<8 | uint32(content[i+3])<<16
		pulses = append(pulses, max(cycles, 1))
		i += 3
	}
	return pulses, nil
}

// MakeTap returns a TAP file, version 1, with the pulses
func MakeTap(pulses []uint32) []uint8 {
	data := make([]uint8, tapHeaderSize, tapHeaderSize+len(pulses))
	copy(data, tapSignature)
	data[12] = 1
	data[13] = tapPetMachine
	for _, p := range pulses {
		v := (p + tapUnit/2) / tapUnit
		if v > 0 && v < 256 {
			data = append(data, uint8(v))
		} else {
			data = append(data, 0, uint8(p), uint8(p>>8), uint8(p>>16))
		}
	}
	binary.LittleEndian.PutUint32(data[16:20], uint32(len(data)-tapHeaderSize))
	return data
}

func parseWavTape(data []uint8) ([]uint32, error) {
	wav, err := parseWavFile(data)
	if err != nil {
		return nil, err
	}
	transitions := extractTransitions(wav, TapeCyclesPerSecond)
	if len(transitions) < 3 {
		return nil, errors.New("no signal found in the tape recording")
	}

	// A pulse goes from a falling edge to the next one. The transitions
	// start with a rising one, the falling ones are the odd ones.
	pulses := make([]uint32, 0, len(transitions)/2)
	for i := 3; i < len(transitions); i += 2 {
		pulses = append(pulses, uint32(transitions[i]-transitions[i-2]))
	}
	return pulses, nil
}

/*
The zero crossings of the signal, cleaned up: a high pass filter removes the
DC offset and the crossings are detected with hysteresis relative to the
envelope, to ignore the noise of old recordings. From izapple2.
*/

const (
	tapeHysteresisRatio = 0.1
	tapeHysteresisFloor = 0.02
	tapeHighPassHz      = 20.0
	tapeEnvelopeSeconds = 0.020
)

func extractTransitions(wav *wavFile, cyclesPerSecond float64) []uint64 {
	channel := wav.bestChannel()
	frames := wav.frames()
	dt := 1.0 / float64(wav.sampleRate)
	cyclesPerFrame := cyclesPerSecond / float64(wav.sampleRate)

	rc := 1.0 / (2 * math.Pi * tapeHighPassHz)
	alpha := rc / (rc + dt)
	decay := math.Exp(-dt / tapeEnvelopeSeconds)

	transitions := make([]uint64, 0, 1000)
	prevX := 0.0
	prevY := 0.0
	envelope := 0.0
	high := false
	lastCross := 0
	for frame := 0; frame < frames; frame++ {
		x := wav.sample(frame, channel)
		y := alpha * (prevY + x - prevX)
		if (y > 0) != (prevY > 0) {
			lastCross = frame
		}
		prevX = x
		prevY = y

		envelope = max(math.Abs(y), envelope*decay)
		threshold := max(envelope*tapeHysteresisRatio, tapeHysteresisFloor)
		if (!high && y > threshold) || (high && y < -threshold) {
			high = !high
			transitions = append(transitions, uint64(float64(lastCross)*cyclesPerFrame))
		}
	}
	return transitions
}
