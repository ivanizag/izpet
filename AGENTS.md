# AGENTS.md - Developer Guide for izpet

izpet is a Commodore PET emulator written in Go, built on the
[iz6502](https://github.com/ivanizag/iz6502) processor library. It follows the
conventions of [izapple2](https://github.com/ivanizag/izapple2), and of
[izmac](https://github.com/ivanizag/izmac) where they fit better.

See `doc/plan.md` for the hardware notes, the decisions and the milestones. Read it
before changing the machine.

## Build and test

```bash
go build ./...
go test ./...

go run ./frontend/petebiten -model 8032
go run ./frontend/headless -model 3032 -frames 200 -type 'print 2+2\n' -png screen.png

# The keyboard diagrams as PNG images, to check a change to them
IZPET_PREVIEW=/tmp go test -run TestKeyboardDiagramPreview ./frontend/shared/
```

The tests of a package that imports ebiten do not run without a display, keep them
out of `frontend/petebiten`.

The ROMs are embedded from `resources/`, so the tests need nothing else. The end to end
tests (`e2e_boot_test.go`) boot every model, type on the keyboard matrix and read the
screen back as text with `GetScreenText`. The helpers are in `petTester_test.go`.

## Code style

- `gofmt`, tabs, standard library imports first, then third party, then local.
- Exported types, functions and methods carry doc comments starting with the name.
- CamelCase for exported names, camelCase for unexported ones. `New<Type>` for exported
  constructors, `new<Type>` for unexported ones.
- Hardware addresses in hex: `0xe810`, `0x8000`.
- Errors are returned, not panicked. Wrap with `fmt.Errorf`.
- The chips in `component/` know nothing of the PET. They were copied from izapple2 and
  izmac and changed for the PET, not shared.

## Scope

- All the PET models with a 6502, US keyboards and character sets only.
- No SuperPET, no CBM-II.
- Drives at the protocol level, no emulation of the drive hardware.
- Frontends: ebiten and headless.

## Things that will bite

- **The I/O chips are selected by address bits**, A4 PIA1, A5 PIA2, A6 VIA, A7 CRTC. An
  address with several bits set reaches several chips, a read is the AND of them.
- **The keyboard row is read when PIA1 port B is read.** The row select is the low
  nibble of PIA1 port A, rows 10 to 15 select nothing.
- **The 60 Hz interrupt is the retrace on PIA1 CB1.** On the 9" boards it is the end of
  the text area, on the CRTC boards the 6545 vertical sync. BASIC 1 also polls VIA PB5
  and hangs if it never goes low.
- **The CRTC is not programmed at power on.** Until the editor ROM sets it up the video
  runs with the timing of the 9" boards, see `isCrtcProgrammed`.
- **The business keyboard starts in the lowercase mode.** Compare the screen text without
  case, and type BASIC with `TypeText`, that types letters unshifted.
- **The zero page of BASIC 1 is not the one of BASIC 2 and 4**, see `basicVarTab`.

## Layout

| Path | Contents |
|---|---|
| `pet.go`, `petRun.go` | the machine and its run loop |
| `models.go` | the model table |
| `memoryManager.go` | the memory map |
| `ioE8Page.go` | the I/O page and the wiring of the chips |
| `video.go` | the video timing, the retrace signals and the image |
| `keyboard.go` | the keyboard matrices and the typing of characters |
| `sound.go` | the CB2 sound |
| `program.go` | loading `.prg` files |
| `screenText.go` | the screen as text |
| `resources.go`, `resources/` | the embedded ROMs |
| `configuration.go`, `command.go` | options and the command channel |
| `component/` | the 6520 PIA, the 6522 VIA and the 6545 CRTC |
| `audio/` | the sample stream for the frontends |
| `frontend/petebiten/` | the ebiten frontend |
| `frontend/shared/` | what the frontends share and the emulator has no place for: the keyboard diagrams, their layouts in key units and their drawing |
| `frontend/headless/` | the headless frontend |
| `doc/` | the plan and the hardware notes |
