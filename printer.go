package izpet

import (
	"fmt"
	"os"

	"github.com/ivanizag/izpet/ieee488"
)

/*
A printer on the IEEE-488 bus, device 4 as the Commodore 4022 and 8023, that
writes what it is sent to a text file of the host, as UTF-8.

The printer starts in the graphics mode, with the unshifted letters
uppercase. The secondary address 7, or CHR$(17), select the business mode,
with the unshifted letters lowercase and the shifted ones uppercase, and
CHR$(145) goes back to the graphics mode. The graphics characters are
printed as '·'.
*/

const printerAddress = 4

type printer struct {
	*ieee488.Peripheral
	filename  string
	file      *os.File
	lowercase bool
}

func newPrinter(filename string) *printer {
	pr := &printer{filename: filename}
	pr.Peripheral = ieee488.NewPeripheral(printerAddress, pr)
	return pr
}

func (pr *printer) Listen(secondary uint8) {
	switch secondary & 0x0f {
	case 7:
		pr.lowercase = true
	case 0:
		pr.lowercase = false
	}
}

func (pr *printer) Talk(secondary uint8) {}
func (pr *printer) Unlisten()            {}
func (pr *printer) Untalk()              {}
func (pr *printer) Next()                {}

func (pr *printer) Peek() (uint8, bool, bool) {
	return 0, false, false
}

func (pr *printer) Receive(b uint8, eoi bool) {
	switch b {
	case 0x11:
		pr.lowercase = true
		return
	case 0x91:
		pr.lowercase = false
		return
	}
	text := petsciiToText(b, pr.lowercase)
	if text == "" {
		return
	}
	if pr.file == nil {
		f, err := os.OpenFile(pr.filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Printer: %v\n", err)
			return
		}
		pr.file = f
	}
	if _, err := pr.file.WriteString(text); err != nil {
		fmt.Printf("Printer: %v\n", err)
	}
}

func (pr *printer) close() error {
	if pr.file == nil {
		return nil
	}
	err := pr.file.Close()
	pr.file = nil
	return err
}

// petsciiToText returns the text for a PETSCII character sent to the printer
func petsciiToText(b uint8, lowercase bool) string {
	switch {
	case b == 0x0d:
		return "\n"
	case b == 0x5e:
		return "↑"
	case b == 0x5f:
		return "←"
	case b >= 0x41 && b <= 0x5a:
		if lowercase {
			return string(rune(b - 0x41 + 'a'))
		}
		return string(rune(b))
	case b >= 0xc1 && b <= 0xda:
		if lowercase {
			return string(rune(b - 0xc1 + 'A'))
		}
		return "·"
	case b >= 0x20 && b < 0x60:
		return string(rune(b))
	case b >= 0x60 && b < 0x80, b >= 0xa0:
		return "·"
	}
	return "" // Control codes
}
