package component

import "testing"

func TestMOS6520DirectionAndPorts(t *testing.T) {
	var p MOS6520
	p.SetInputA(0xff)
	p.Write(0, 0x0f) // DDRA, CRA bit 2 is 0 after reset
	p.Write(1, 0x04) // Select the port register
	p.Write(0, 0x05)
	if got := p.GetPortA(); got != 0xf5 {
		t.Errorf("port A pins: got $%02x, want $f5", got)
	}

	p.SetInputB(0xa5)
	p.Write(3, 0x04)
	if got := p.Read(2); got != 0xa5 {
		t.Errorf("port B read: got $%02x, want $a5", got)
	}
}

func TestMOS6520InterruptOnC1(t *testing.T) {
	var p MOS6520
	p.SetCB1(true)
	p.Write(3, 0x05) // IRQ on the falling edge of CB1, port register selected
	p.SetCB1(false)
	if !p.InterruptAsserted() {
		t.Fatal("the falling edge of CB1 should interrupt")
	}
	if p.Read(3)&0x80 == 0 {
		t.Error("the CB1 flag should be set")
	}
	p.Read(2)
	if p.InterruptAsserted() {
		t.Error("reading port B should clear the interrupt")
	}
	p.SetCB1(true)
	if p.InterruptAsserted() {
		t.Error("the rising edge should not interrupt")
	}
}

func TestMOS6520C2Output(t *testing.T) {
	var p MOS6520
	p.Write(1, 0x34) // CA2 manual output low, as the PET writes 52
	if p.GetCA2() {
		t.Error("CA2 should be low")
	}
	p.Write(1, 0x3c) // 60, CA2 high
	if !p.GetCA2() {
		t.Error("CA2 should be high")
	}
}
