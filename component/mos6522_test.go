package component

import (
	"testing"
)

func TestMOS6522Timer1OneShot(t *testing.T) {
	var v MOS6522
	v.Write(14, 0x80|0x40) // Enable the T1 interrupt
	v.Write(4, 100)        // T1 latch low
	v.Write(5, 0)          // T1 counter high, starts the timer

	v.Tick(100)
	if v.InterruptAsserted() {
		t.Error("The interrupt should not fire before the timer underflows")
	}

	v.Tick(2)
	if !v.InterruptAsserted() {
		t.Error("The interrupt should fire when the timer underflows")
	}

	// In one shot mode it must fire only once
	v.Write(13, 0x40) // Clear the T1 flag
	v.Tick(1000)
	if v.InterruptAsserted() {
		t.Error("The one shot timer should not fire again")
	}
}

func TestMOS6522CA2Output(t *testing.T) {
	var v MOS6522
	v.Write(12, 12) // POKE 59468,12: the graphics character set
	if v.GetCA2() {
		t.Error("CA2 should be low with 12")
	}
	v.Write(12, 14) // POKE 59468,14: the lowercase character set
	if !v.GetCA2() {
		t.Error("CA2 should be high with 14")
	}
}

func TestMOS6522ShiftOutFreeRunning(t *testing.T) {
	var v MOS6522
	v.Write(11, 16) // POKE 59467,16: shift out free running
	v.Write(10, 15) // POKE 59466,15: the pattern 00001111
	v.Write(8, 98)  // POKE 59464,98: a bit every 100 cycles

	levels := ""
	for i := 0; i < 16; i++ {
		v.Tick(100)
		if v.GetCB2() {
			levels += "1"
		} else {
			levels += "0"
		}
	}
	if levels != "0000111100001111" {
		t.Errorf("CB2 levels: got %v", levels)
	}
	if v.InterruptAsserted() {
		t.Error("the free running mode should not interrupt")
	}
}
