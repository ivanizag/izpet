# izpet - Commodore PET emulator

izpet is a Commodore PET and CBM emulator written in Go, in the style of
[izapple2](https://github.com/ivanizag/izapple2) and
[izmac](https://github.com/ivanizag/izmac). The 6502 is emulated by
[iz6502](https://github.com/ivanizag/iz6502).

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
- Loading and running `.prg` files
- Option ROMs at `$9000` and `$a000`
- A drawing of the keyboard of the model below the screen, to type by clicking on the keys

Not yet: cassette, IEEE-488 drives and printers, the 8096 and 8296.
See [doc/plan.md](doc/plan.md).

## Running

```bash
# The ebiten frontend
go run ./frontend/petebiten -model 8032

# Load and run a program
go run ./frontend/petebiten -model 3032 game.prg

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
| Arrows | the cursor keys |
| Ctrl-F2 | Reset, F2 alone while the help is shown |
| F4 | Show or hide the processor trace |
| F5 | Full speed on and off, Ctrl-F5 to report the speed |
| F7 | Show or hide the keyboard |
| F12 | Save the screen to a PNG file |
| Pause | Pause |

Drop a `.prg` file on the window to load and run it.

The keys of the keyboard below the screen can be clicked. A key is held while the mouse
button is. SHIFT stays down after a click and comes up with the next key, so
shifted keys are typed with two clicks. SHIFT LOCK goes down with a click and
up with the next one. The keys down are shown pressed, framed in green, also the ones typed on the host
keyboard.

## Command line options

```
  -50hz         use the 50 Hz editor ROM, for the CRTC models
  -model string model to emulate (default "3032")
  -ram int      RAM in KB, up to 32. The default depends on the model
  -rom9 string  option ROM at $9000
  -roma string  option ROM at $a000
  -speed string 'real', 'full' or the speed in MHz (default "real")
  -trace string comma separated list of tracers: cpu
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
