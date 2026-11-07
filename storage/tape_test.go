package storage

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestTapRoundTrip(t *testing.T) {
	pulses := []uint32{352, 512, 672, 100_000}
	data := MakeTap(pulses)
	if !IsTapFile(data) {
		t.Fatal("not recognized as a TAP file")
	}
	got, err := LoadTape(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(pulses) {
		t.Fatalf("got %v pulses", len(got))
	}
	for i, p := range pulses {
		if math.Abs(float64(got[i])-float64(p)) > 4 {
			t.Errorf("pulse %v is %v, expected %v", i, got[i], p)
		}
	}
}

func TestTapVersion0Pause(t *testing.T) {
	data := MakeTap(nil)
	data[12] = 0
	data = append(data, 0x30, 0, 0x30)
	binary.LittleEndian.PutUint32(data[16:20], 3)
	got, err := LoadTape(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != 0x30*8 || got[1] != tapLongPause {
		t.Errorf("got %v", got)
	}
}

// testWav makes a WAV file with a square wave of the pulses
func testWav(pulses []uint32, rate int) []byte {
	var samples []int16
	for _, p := range pulses {
		half := int(p) * rate / TapeCyclesPerSecond / 2
		for i := 0; i < half; i++ {
			samples = append(samples, -20000)
		}
		for i := 0; i < half; i++ {
			samples = append(samples, 20000)
		}
	}
	data := []byte("RIFF....WAVEfmt ")
	format := make([]byte, 20)
	binary.LittleEndian.PutUint32(format[0:], 16)
	binary.LittleEndian.PutUint16(format[4:], 1)
	binary.LittleEndian.PutUint16(format[6:], 1)
	binary.LittleEndian.PutUint32(format[8:], uint32(rate))
	binary.LittleEndian.PutUint32(format[12:], uint32(rate*2))
	binary.LittleEndian.PutUint16(format[16:], 2)
	binary.LittleEndian.PutUint16(format[18:], 16)
	data = append(data, format...)
	data = append(data, "data...."...)
	binary.LittleEndian.PutUint32(data[len(data)-4:], uint32(len(samples)*2))
	for _, s := range samples {
		data = binary.LittleEndian.AppendUint16(data, uint16(s))
	}
	return data
}

func TestWavTape(t *testing.T) {
	var pulses []uint32
	for i := 0; i < 50; i++ {
		pulses = append(pulses, 352, 512, 672)
	}
	got, err := LoadTape(testWav(pulses, 44100))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 140 {
		t.Fatalf("got %v pulses", len(got))
	}
	// Away from the start, where the filter settles
	for i := 10; i < len(got)-10; i++ {
		expected := []uint32{352, 512, 672}
		ok := false
		for _, e := range expected {
			if math.Abs(float64(got[i])-float64(e)) < 50 {
				ok = true
			}
		}
		if !ok {
			t.Errorf("pulse %v is %v", i, got[i])
		}
	}
}
