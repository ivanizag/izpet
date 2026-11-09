# izpet - Commodore PET emulator

izpet is a Commodore PET and CBM emulator written in Go, in the style of
[izapple2](https://github.com/ivanizag/izapple2) and
[izmac](https://github.com/ivanizag/izmac). The 6502 is emulated by
[iz6502](https://github.com/ivanizag/iz6502).

## Things to do

The [activities](doc/activities/README.md) walk through what using a PET was
like, step by step, with the pictures of what the machine shows on the way.

## Models

| Model | Description |
|---|---|
| `2001` | PET 2001, BASIC 1, chiclet keyboard, 8 KB |
| `3032` | PET 2001-N or CBM 3032, BASIC 2, graphics keyboard, 32 KB. The default |
| `3032b` | PET 2001-B or CBM 3032, BASIC 2, business keyboard, 32 KB |
| `4016` | CBM 4016, BASIC 4 on the 9" board, graphics keyboard, 16 KB |
| `4032b` | CBM 4032, BASIC 4 on the 9" board, business keyboard, 32 KB |
| `4032` | CBM 4032, BASIC 4 on the 12" CRTC board, graphics keyboard, 32 KB |
| `8032` | CBM 8032, BASIC 4, 80 columns, business keyboard, 32 KB |

The ROMs are embedded in the program, there is nothing to download.

## Features

- The two PIAs, the VIA and the 6545 CRTC, with the I/O page decoded by address bits as
  on the real machine
- The video of the discrete logic boards and of the CRTC boards, 40 and 80 columns, 50 and
  60 Hz, text and graphics character sets
- The graphics and business keyboard matrices, read from the editor ROMs
- Sound on CB2
- Loading and running `.prg` and `.p00` files, and the first file of `.t64` files
- Two cassette decks playing TAP and WAV files and recording TAP files
- The IEEE-488 bus with a dual disk drive on unit 8, on D64, D80 and D82 images
  and folders of the host, and a printer on unit 4 writing to a text file
- Option ROMs at `$9000` and `$a000`
- A drawing of the keyboard of the model below the screen, to type by clicking on the keys

Not yet: the 8096 and 8296.
See [doc/plan.md](doc/plan.md).

The [`activities`](activities) package makes the pictures of the
[activities](doc/activities/README.md):

```bash
IZPET_ACTIVITY_SCREENSHOTS=1 go test -count=1 -run TestActivityScreenshots ./activities
```

## Running

```bash
# The ebiten frontend
go run ./frontend/petebiten -model 8032

# Load and run a program
go run ./frontend/petebiten -model 3032 game.prg

# A disk image in the drive 0 of the unit 8, and a folder of the host in the drive 1
go run ./frontend/petebiten -model 8032 -disk work.d80 -disk1 ~/petfiles

# A tape in the deck #1 and a printer on the unit 4
go run ./frontend/petebiten -tape games.tap -printer printer.txt

# The headless frontend: run, type and show the screen as text
go run ./frontend/headless -model 4032 -type 'print 6*7\n'
```

The printable characters are typed by what they are, whatever the layout of the
host keyboard. In the ebiten frontend:

| Key | Action |
|---|---|
| F1 | Show or hide the help |
| Escape | RUN/STOP |
| F3 | RVS |
| F9 | ESC, on the business keyboard |
| Insert | INST |
| Home | HOME, with shift CLR |
| Ctrl-V | Paste the text of the clipboard, also Cmd-V and Shift-Insert |
| Arrows | the cursor keys |
| Ctrl-F2 | Reset, F2 alone while the help is shown |
| F4 | Show or hide the processor trace |
| F5 | Full speed on and off, Ctrl-F5 to report the speed |
| F7 | Show or hide the keyboard |
| F12 | Save the screen to a PNG file |
| Pause | Pause |

A text pasted is typed on the PET: a BASIC listing pasted at READY. is entered
line by line. In the graphics mode the letters are typed unshifted, uppercase,
whatever their case, as BASIC wants them; in the lowercase mode they are typed
as they are.

Drop a file on the window to use it: a program is loaded and run, a tape goes
in the deck #1 and a disk in the drive.

The keys of the keyboard below the screen can be clicked. A key is held while the mouse
button is. SHIFT stays down after a click and comes up with the next key, so
shifted keys are typed with two clicks. SHIFT LOCK goes down with a click and
up with the next one. The keys down are shown pressed, framed in green, also the ones typed on the host
keyboard.

## Disks, tapes and the printer

The files named after the options, or dropped on the window, go where they
belong by their type:

| Type | Goes |
|---|---|
| `.prg`, `.p00` | loaded once BASIC is ready. A BASIC program is run, and moved to `$0401` if it was saved elsewhere, by a C64 for instance; machine code is put where it was saved, to be started with `SYS`. A program that does not fit in the RAM of the model is refused |
| `.t64` | its first file is loaded and run |
| `.tap`, `.wav` | in the deck #1 |
| `.d64`, `.d80`, `.d82`, a folder | in the drive 0 of the unit 8, or the drive 1 if the 0 has a disk |

The drive answers the Commodore DOS commands: `LOAD "$",8` and `DIRECTORY`,
`LOAD`, `SAVE`, `DLOAD`, `DSAVE`, sequential files with `OPEN`, and the command
channel 15 with `S` scratch, `R` rename, `C` copy, `N` format, `I`, `V` and `UJ`,
and the error status. There are no relative files. A disk image that does not
exist is created, blank, when the PET writes on it. In a folder of the host the
files are named with their PET name and type, `hello.prg`, `data.seq`.

BASIC 1, the original ROM of the 2001, can't use the disks, as on the real
machine. BASIC 2 waits for a file that is not on the disk until STOP is pressed,
only BASIC 4 reports it.

With a tape in the deck, PLAY is pressed. The tape moves with the motor and the
emulator runs at full speed meanwhile. What the PET saves is recorded on the
tape and the tape is saved as a TAP file when the motor stops. A TAP file that
does not exist is a blank tape.

The printer writes what is printed on the unit 4 to a text file, in the
graphics mode, or with the lowercase letters after `OPEN 4,4,7` or `CHR$(17)`.

## Command line options

```
  -50hz         use the 50 Hz editor ROM, for the CRTC models
  -disk string  disk image, D64, D80 or D82, or folder for the drive 0 of the unit 8
  -disk1 string disk image or folder for the drive 1 of the unit 8
  -model string model to emulate (default "3032")
  -printer string text file to write what is printed on the unit 4, no printer if empty
  -ram int      RAM in KB, up to 32. The default depends on the model
  -rom9 string  option ROM at $9000
  -roma string  option ROM at $a000
  -speed string 'real', 'full' or the speed in MHz (default "real")
  -tape string  tape, TAP or WAV, for the deck #1
  -trace string comma separated list of tracers: cpu, drive
```

## ROMs and fonts

The ROM images in `resources/` are from the
[zimmers.net PET firmware archive](https://www.zimmers.net/anonftp/pub/cbm/firmware/computers/pet/).

The keyboards are drawn with the [Michroma](https://github.com/googlefonts/Michroma-font)
font, close to the Microgramma of the Commodore keys, under the SIL Open Font License,
see `frontend/shared/fonts/OFL.txt`. The graphics symbols on the keys come from the
character ROM. The colors of the 2001 keyboard follow the drawing
[PET Keyboard improved.svg](https://commons.wikimedia.org/wiki/File:PET_Keyboard_improved.svg)
by E5frog, CC0.
