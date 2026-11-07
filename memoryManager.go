package izpet

import (
	"fmt"
)

/*
The memory map of the PET:

	$0000-$7fff RAM, as much as installed
	$8000-$8fff Screen RAM, 1K or 2K mirrored over the 4K
	$9000-$afff Option ROM sockets
	$b000-$dfff BASIC, BASIC 1 and 2 start at $c000
	$e000-$e7ff Editor ROM
	$e800-$e8ff I/O
	$e900-$efff Nothing
	$f000-$ffff Kernal ROM

Unused addresses read as the high byte of the address, the last byte on the
data bus for most of the absolute addressing modes.
*/

const (
	addressScreen  = 0x8000
	addressRomBase = 0x9000
	addressIO      = 0xe800
	addressIOEnd   = 0xe900
	addressKernal  = 0xf000
	romBlockSize   = 0x800
	romBlockCount  = (0x10000 - addressRomBase) / romBlockSize
)

type memoryManager struct {
	p          *PET
	ram        []uint8
	screen     [0x800]uint8
	screenMask uint16
	rom        [0x10000 - addressRomBase]uint8
	romPresent [romBlockCount]bool
}

func newMemoryManager(p *PET, ramKB int, screenSize int) *memoryManager {
	var mmu memoryManager
	mmu.p = p
	mmu.ram = make([]uint8, ramKB*1024)
	mmu.screenMask = uint16(screenSize - 1)
	return &mmu
}

// loadRom places a ROM image
func (mmu *memoryManager) loadRom(address uint16, data []uint8) error {
	if address < addressRomBase || int(address)+len(data) > 0x10000 ||
		address%romBlockSize != 0 || len(data)%romBlockSize != 0 {
		return fmt.Errorf("a ROM of %v bytes can't be placed at $%04x", len(data), address)
	}
	copy(mmu.rom[address-addressRomBase:], data)
	for i := 0; i < len(data)/romBlockSize; i++ {
		mmu.romPresent[int(address-addressRomBase)/romBlockSize+i] = true
	}
	return nil
}

func openBus(address uint16) uint8 {
	return uint8(address >> 8)
}

// Peek returns the data on the given address
func (mmu *memoryManager) Peek(address uint16) uint8 {
	switch {
	case address < addressScreen:
		if int(address) < len(mmu.ram) {
			return mmu.ram[address]
		}
		return openBus(address)
	case address < addressRomBase:
		return mmu.screen[address&mmu.screenMask]
	case address >= addressIO && address < addressIOEnd:
		return mmu.p.io.peek(uint8(address))
	default:
		if address >= addressIOEnd && address < addressKernal {
			return openBus(address)
		}
		offset := address - addressRomBase
		if mmu.romPresent[offset/romBlockSize] {
			return mmu.rom[offset]
		}
		return openBus(address)
	}
}

// PeekCode returns the data on the given address, for the CPU to fetch code
func (mmu *memoryManager) PeekCode(address uint16) uint8 {
	return mmu.Peek(address)
}

// Poke sets the data at the given address
func (mmu *memoryManager) Poke(address uint16, value uint8) {
	switch {
	case address < addressScreen:
		if int(address) < len(mmu.ram) {
			mmu.ram[address] = value
		}
	case address < addressRomBase:
		mmu.screen[address&mmu.screenMask] = value
	case address >= addressIO && address < addressIOEnd:
		mmu.p.io.poke(uint8(address), value)
	}
}

// screenRAM returns the screen memory, unmirrored
func (mmu *memoryManager) screenRAM() []uint8 {
	return mmu.screen[:mmu.screenMask+1]
}
