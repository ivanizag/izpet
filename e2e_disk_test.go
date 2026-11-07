package izpet

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestPETWithDisk(t *testing.T, model string, disk string) *PET {
	t.Helper()
	p := newTestPETWith(t, model, func(c *Configuration) { c.Disk = disk })
	testWaitText(t, p, "READY.")
	return p
}

func TestDiskSaveLoad(t *testing.T) {
	for _, model := range []string{"3032", "4032b", "8032"} {
		t.Run(model, func(t *testing.T) {
			disk := filepath.Join(t.TempDir(), "test.d64")
			p := newTestPETWithDisk(t, model, disk)
			testType(t, p, "10 PRINT \"IZPET\";6*7\r")
			testRun(t, p, "SAVE \"PROG\",8", "READY.")
			if _, err := os.Stat(disk); err != nil {
				t.Fatalf("the disk image was not created: %v", err)
			}

			testRun(t, p, "NEW", "READY.")
			testRun(t, p, "LOAD \"PROG\",8", "READY.")
			testRun(t, p, "RUN", "IZPET 42")

			testRun(t, p, "LOAD \"$\",8", "READY.")
			testRun(t, p, "LIST", "BLOCKS FREE.")
			screen := p.GetScreenText()
			upper := strings.ToUpper(screen)
			if !strings.Contains(upper, " 1    \"PROG\"") || !strings.Contains(upper, "PRG") ||
				!strings.Contains(upper, "663 BLOCKS FREE.") {
				t.Errorf("unexpected directory:\n%v", screen)
			}
		})
	}
}

func TestDiskBasic4Commands(t *testing.T) {
	disk := filepath.Join(t.TempDir(), "test.d80")
	p := newTestPETWithDisk(t, "8032", disk)
	testRun(t, p, "HEADER \"WORK\",D0,IAB", "ARE YOU SURE")
	testType(t, p, "Y\r")
	testWaitText(t, p, "READY.")
	testType(t, p, "10 PRINT \"DSAVED\"\r")
	testRun(t, p, "DSAVE \"DS\"", "READY.")
	testRun(t, p, "DIRECTORY", "BLOCKS FREE.")
	screen := strings.ToUpper(p.GetScreenText())
	if !strings.Contains(screen, "\"WORK            \" AB 2C") ||
		!strings.Contains(screen, "\"DS\"") || !strings.Contains(screen, "2051 BLOCKS FREE.") {
		t.Errorf("unexpected directory:\n%v", screen)
	}
	testRun(t, p, "NEW", "READY.")
	testRun(t, p, "DLOAD \"DS\"", "READY.")
	testRun(t, p, "RUN", "DSAVED")

	testRun(t, p, "DLOAD \"NOPE\"", "READY.")
	testRun(t, p, "PRINT DS$", "62,FILE NOT FOUND,00,00")

	testRun(t, p, "SCRATCH \"DS\",D0", "ARE YOU SURE")
	testType(t, p, "Y\r")
	testWaitText(t, p, "READY.")
	testRun(t, p, "DIRECTORY", "BLOCKS FREE.")
	if strings.Contains(strings.ToUpper(p.GetScreenText()), "\"DS\"") {
		t.Errorf("the file was not scratched:\n%v", p.GetScreenText())
	}
}

func TestDiskErrors(t *testing.T) {
	// BASIC 2 waits for a file not found until STOP is pressed, only BASIC 4
	// reports it
	p := newTestPETWithDisk(t, "8032", filepath.Join(t.TempDir(), "test.d64"))
	testRun(t, p, "LOAD \"NOPE\",8", "FILE NOT FOUND")
	testRun(t, p, "PRINT DS$", "62,FILE NOT FOUND,00,00")
}

func TestDiskDataFiles(t *testing.T) {
	p := newTestPETWithDisk(t, "3032", filepath.Join(t.TempDir(), "test.d64"))
	testRun(t, p, "OPEN 2,8,2,\"0:DATA,S,W\":PRINT#2,\"SEQUENTIAL\":CLOSE 2", "READY.")

	// INPUT# is not allowed in direct mode
	testType(t, p, "10 OPEN 2,8,2,\"0:\"+F$+\",S,R\":INPUT#2,A$:CLOSE 2:PRINT \"READ \";A$\r")
	testType(t, p, "15 END\r")
	testType(t, p, "20 OPEN 15,8,15:INPUT#15,E,E$,T:CLOSE 15:PRINT \"STATUS\";E;E$;T\r")
	testRun(t, p, "F$=\"DATA\":GOTO 10", "READ SEQUENTIAL")
	testRun(t, p, "GOTO 20", "STATUS 0 OK 0")

	// The command channel
	testRun(t, p, "OPEN 15,8,15,\"R0:INFO=DATA\":CLOSE 15", "READY.")
	testRun(t, p, "F$=\"INFO\":GOTO 10", "READ SEQUENTIAL")
	testRun(t, p, "OPEN 15,8,15,\"S0:INFO\":CLOSE 15", "READY.")
	testRun(t, p, "GOTO 20", "STATUS 1 FILES SCRATCHED 1")
	testRun(t, p, "F$=\"INFO\":GOTO 10", "READY.")
	testRun(t, p, "GOTO 20", "STATUS 62 FILE NOT FOUND 0")
}

func TestDiskFolder(t *testing.T) {
	dir := t.TempDir()
	// 10 PRINT "FROM HOST"
	program := []uint8{0x01, 0x04, 0x11, 0x04, 10, 0, 0x99, ' ', '"', 'F', 'R', 'O', 'M', ' ', 'H', 'O', 'S', 'T', '"', 0, 0, 0}
	if err := os.WriteFile(filepath.Join(dir, "host.prg"), program, 0644); err != nil {
		t.Fatal(err)
	}

	p := newTestPETWithDisk(t, "3032", dir)
	testRun(t, p, "LOAD \"HOST\",8", "READY.")
	testRun(t, p, "RUN", "FROM HOST")
	testRun(t, p, "SAVE \"SAVED\",8", "READY.")
	if _, err := os.Stat(filepath.Join(dir, "saved.prg")); err != nil {
		t.Errorf("the program was not saved in the folder: %v", err)
	}
	testRun(t, p, "LOAD \"$\",8", "READY.")
	testRun(t, p, "LIST", "BLOCKS FREE.")
	screen := p.GetScreenText()
	if !strings.Contains(screen, "\"HOST\"") || !strings.Contains(screen, "\"SAVED\"") {
		t.Errorf("unexpected directory:\n%v", screen)
	}
}

func TestPrinter(t *testing.T) {
	output := filepath.Join(t.TempDir(), "printer.txt")
	p := newTestPETWith(t, "3032", func(c *Configuration) { c.Printer = output })
	testWaitText(t, p, "READY.")
	testRun(t, p, "OPEN 4,4:PRINT#4,\"HELLO PRINTER\";6*7:CLOSE 4", "READY.")
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "HELLO PRINTER 42 \n" {
		t.Errorf("printed %q", data)
	}
}
