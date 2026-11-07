# izpet plan

A first pass over the Commodore PET family: which models there are, what is inside them,
and what izpet has to emulate. It follows the shape of `izmac/doc/plan.md` and will turn
into `doc/plan.md`. The scope decisions are at the end, in "Decisions taken".

Facts marked *(verify)* are from memory and must be checked against the schematics, the
ROM disassemblies or VICE's `xpet` before code depends on them. Facts marked *(verified)*
were checked against the ROMs, the zimmers.net notes (`PET-Interfaces.txt`) or the VICE
sources while writing milestone 1.

## The family

Every PET is the same machine at heart: a 1 MHz 6502, RAM from `$0000`, a character
screen at `$8000`, ROMs from `$B000`/`$C000` up, and two 6520 PIAs plus one 6522 VIA at
`$E8xx`. The models differ in five axes, and the emulator should be built around them
rather than around model names:

| Axis | Values |
|---|---|
| ROM set | BASIC 1 (original), BASIC 2 (upgrade), BASIC 4; each with its editor ROM |
| Video | discrete TTL 40 columns (no CRTC); 6545 CRTC 40 columns; 6545 CRTC 80 columns |
| Keyboard | 2001 chiclet; graphics (N); business (B); US layouts only |
| RAM | 4, 8, 16, 32 KB; 96 KB (8096, banked); 128 KB (8296, banked) |
| Extras | built in cassette; built in speaker on CB2; built in drives (8296-D) |

| Model | Year | ROM | Video | Keyboard | RAM | Notes |
|---|---|---|---|---|---|---|
| PET 2001-4/-8 | 1977 | BASIC 1 | TTL 40, 9" | chiclet | 4/8 KB | built in Datasette; screen "snow"; ROM waits for retrace |
| PET 2001-N/-B, CBM 3008/3016/3032 | 1979 | BASIC 2 | TTL 40, 9"/12" | graphics or business | 8/16/32 KB | external cassette |
| CBM 4016/4032 "thin 40" | 1980 | BASIC 4 | TTL 40 | graphics or business | 16/32 KB | 2001-N board with BASIC 4 |
| CBM 4032 "fat 40" | 1981 | BASIC 4 | CRTC 40, 12" | graphics or business | 32 KB | 8032 board strapped for 40 columns; speaker |
| CBM 8032 / 8032-SK | 1980 | BASIC 4 | CRTC 80, 12" | business | 32 KB | speaker |
| CBM 8096 | 1981 | BASIC 4 | CRTC 80 | business | 32 + 64 KB | 8032 + 64 KB board, banked by `$FFF0` |
| CBM 8296 / 8296-D | 1984 | BASIC 4 | CRTC 80 | business | 128 KB | new board, banked by `$FFF0`; -D has an 8250LP inside |
| SuperPET SP9000 / MMF 9000 | 1981 | BASIC 4 + Waterloo | CRTC 80 | business (APL) | 32 + 64 KB | second CPU 6809, 6551 ACIA, 6702 dongle — **out of scope** |

Out of scope: the **SuperPET**, whose 6809 side would need a new CPU core (its 6502
side is close to an 8096, but the Waterloo software is the point of the machine). The **CBM-II** line (B128, 600/700, P500) shares the
"CBM" name and the business look but is another architecture (6509, 6525 TPI, CIA, SID);
VICE gives it its own `xcbm2`. The **Educator 64 / PET 64** is a C64 in a PET case.

## The processor

A plain NMOS **6502 at 1 MHz** in every model, so `iz6502` does it as is, including the
undocumented opcodes some PET software uses. Interrupts come only on **IRQ**: the 60 Hz
(or 50 Hz) vertical retrace through PIA1 CB1 is the heartbeat that drives the jiffy clock,
the cursor blink and the keyboard scan. NMI is on the expansion connector and unused by
the stock machine.


## Memory map

| Range | Contents |
|---|---|
| `$0000`–`$7FFF` | RAM, as much as fitted; the rest reads as open bus |
| `$8000`–`$8FFF` | screen RAM: 1 KB mirrored 4 times on 40 columns, 2 KB mirrored twice on 80 |
| `$9000`–`$9FFF` | option ROM socket (empty, or a toolkit, Wordpro, an assembler...) |
| `$A000`–`$AFFF` | option ROM socket |
| `$B000`–`$DFFF` | BASIC 4 (12 KB); BASIC 1 and 2 only start at `$C000` and leave `$B000` empty |
| `$E000`–`$E7FF` | editor ROM: screen, keyboard, the 50/60 Hz and 40/80 column personality |
| `$E800`–`$E8FF` | I/O, see below |
| `$E900`–`$EFFF` | unused on most boards, open bus in izpet *(verify what each board decodes there)* |
| `$F000`–`$FFFF` | kernal ROM, with the reset and IRQ vectors |

### The I/O page

The chips are selected by single address bits, not by a decoder, so an address with more
than one of them set selects more than one chip, and the registers repeat all over the page:

| Chip | Selected by | Base | Registers |
|---|---|---|---|
| PIA1 (6520) | A4 | `$E810` | 4, on A0–A1 |
| PIA2 (6520) | A5 | `$E820` | 4, on A0–A1 |
| VIA (6522) | A6 | `$E840` | 16, on A0–A3 |
| CRTC (6545) | A7 | `$E880` | 2, on A0 (address, data) — CRTC boards only |

A write to `$E830` lands in both PIAs; a read returns the AND of what drives the bus
*(verified, VICE does the same)*. Some programs depend on the mirrors, so decode by bit, like izmac decodes
loosely.

### The wiring

What each pin does is the core of the machine, and it is the same on every model except
where noted *(verified against `PET-Interfaces.txt` and VICE's `petpia1.c`, `petpia2.c`,
`petvia.c`; not yet against the schematics)*:

**PIA1 `$E810`** — keyboard, cassette, retrace

| Pin | Use |
|---|---|
| PA0–PA3 | keyboard row select, through a 74145 to 10 rows |
| PA4 | cassette #1 switch sense (0 = a key pressed) |
| PA5 | cassette #2 switch sense |
| PA6 | IEEE EOI in |
| PA7 | diagnostic sense: low at reset sends the kernal to the diagnostics/monitor |
| PB0–PB7 | keyboard columns in (0 = key down) |
| CA1 | cassette #1 read |
| CA2 | IEEE EOI out; on the 2001 it also blanks the screen |
| CB1 | **vertical retrace in, the 60 Hz IRQ** |
| CB2 | cassette #1 motor |

**PIA2 `$E820`** — IEEE-488 data

| Pin | Use |
|---|---|
| PA0–PA7 | IEEE DI1–DI8 in |
| PB0–PB7 | IEEE DO1–DO8 out |
| CA1 | ATN in |
| CA2 | NDAC out |
| CB1 | SRQ in |
| CB2 | DAV out |

**VIA `$E840`** — IEEE handshake, user port, sound, character set

| Pin | Use |
|---|---|
| PA0–PA7 | user port, parallel |
| PB0 | NDAC in |
| PB1 | NRFD out |
| PB2 | ATN out |
| PB3 | cassette write (both decks) |
| PB4 | cassette #2 motor |
| PB5 | vertical retrace in (polled: BASIC 1 waits on it before touching the screen) |
| PB6 | NRFD in |
| PB7 | DAV in |
| CA1 | user port CA1 |
| CA2 | **character set**: graphics or lower case (`POKE 59468,12/14`) |
| CB1 | cassette #2 read |
| CB2 | user port, and **the sound**: the shift register in free running mode makes the tone |

## Video

Two very different generators, one per board family.

**The TTL boards (2001, 2001-N, 3000, thin 40 4000).** Fixed timing, 40×25 characters of
8×8 pixels, 320×200, read from `$8000` through a 2 KB character ROM holding two sets of 128
glyphs; bit 7 of the screen code inverts the cell in hardware, and VIA CA2 picks the set.
64 cycles per line, 260 lines, 16 640 cycles per frame, 60.1 Hz; the blanking signal goes
low after the 200 text lines and stays low for 60 lines *(verified, VICE uses the same
numbers)*. On the
original 2001 the CPU and the video share the screen RAM without arbitration: writes during
the visible area produce **snow**, which is why BASIC 1 waits for retrace and why the
"killer poke" (`POKE 59458,62`) sped up printing on the boards that do not snow. Snow is an
optional nicety; the retrace wait is not.

**The CRTC boards (fat 40 4000, 8000).** A 6545, a 6845 subset, does the timing.
The 80 column board fetches two characters per CRTC clock, so the CRTC is programmed for 40
and the board doubles it. The editor ROM programs it, and the programming differs between
text mode (10 scan lines per row, a gap between rows) and graphics mode (8 lines, glyphs
touching), and between the 50 Hz and 60 Hz editors. Some software reprograms it to change
the screen geometry. The top bits of the start address are wired to board functions such
as whole screen inversion and character ROM selection: MA12 low reverses the screen (not
on the 8296), MA13 selects the upper half of a 4K character ROM *(verified in VICE's
`pet.c`)*. The 6545 vertical sync is the interrupt, the display enable is PB5.

With a CRTC the whole frame timing falls out of the registers:
`cycles per frame = (R0+1) × ((R4+1) × (R9+1) + R5)` at one character clock per CPU cycle.
Like izmac's "every 352 cycles is a line", that one number drives the retrace IRQ, the
retrace bit on VIA PB5 and the frame the frontend draws. izapple2 has an `MC6845` in
`component/` (written for the Videx cards) to start from; it needs the 6545 differences
and the vertical sync output.

Display: green (or white, on some 12" monitors) on black, no colour. The character ROMs
come in several variants; izpet needs two, the original 2001 one and the later standard
one, both US.

## Keyboard

A matrix of 10 rows × 8 columns scanned by the editor ROM every IRQ. There are three
physical layouts with **different matrices**, and the editor ROM must match:

- the 2001 chiclet keyboard, with its numeric pad;
- the graphics keyboard (2001-N, 3000, 4000 "G"), with the PETSCII graphics on the keys;
- the business keyboard (2001-B, 4000 "B", 8000), typewriter style, no graphics
  symbols, with TAB, ESC, REPEAT.

*(verified)* The chiclet and the graphics keyboards share **one matrix**: the decode tables of
the BASIC 1 and BASIC 2 graphics editors are identical. So there are two matrices, both read
from the 80 byte decode tables of the editor ROMs (`$E6F8` in `edit-2-n`, `$E6D1` in the 8032
editor), scanned from row 0, bit 0 at the end of the table. On the business keyboard a code
with bit 7 set is not affected by shift (the numeric pad, `@`, `[`, `]`...), and shift turns
`,`–`;` into `<`–`+` and `1`–`9` into `!`–`)`.

Only the US layouts are emulated; the national variants (German, Swedish...) with their
editor ROMs and character sets are out of scope. The host side needs both a positional mapping
(for games) and a symbolic one (for typing), the way izapple2's `frontend/shared/keys.go`
does. RUN/STOP, the shift lock, REVERSE and the cursor keys need choices on a PC keyboard.

## Sound

There is no sound chip. The VIA's shift register, set to free running mode, shifts a byte
pattern out of CB2 at the rate of timer 2, and the machine is "programmed" with three
POKEs (`59467` mode, `59466` pattern, `59464` rate). The 4000/8000 12" machines have a
small speaker on CB2 and the 80 column editor beeps on `CHR$(7)`; on the 2001 a speaker had
to be wired to the user port. Emulation: sample the CB2 level per cycle, like izapple2's
speaker click, into the same audio stream code.

## Storage and peripherals

### Cassette

Two decks: #1 on PIA1 (read CA1, sense PA4, motor CB2), #2 mostly on the VIA (read CB1,
motor PB4, but sense on PIA1 PA5), writing shared on VIA PB3. The data is pulse length encoded
(short, medium, long pulses) and the kernal times the edges. The common files:

- **TAP**: the pulse lengths, so it plays through the real kernal routines exactly as
  izapple2's `cassette.go` plays its list of transitions. The sense line must show "PLAY
  pressed" while playing.
- **T64**: a container of program files, with no pulses; read by generating the pulses, or
  by a kernal trap.
- **WAV**: decoded to edges the way izapple2's `storage/cassette.go` does.

### IEEE-488

The disks and printers all hang on the IEEE-488 bus, driven bit by bit from PIA2 and the
VIA. Eight data lines and DAV, NRFD, NDAC, ATN, EOI, SRQ, IFC, REN, all open collector,
so the bus is the wired AND of every device.

Devices that matter: the 2040/3040 (DOS 1), **4040** (DOS 2, 170 KB, the `.d64` geometry),
**8050** (DOS 2.5, 500 KB, `.d80`), **8250** and SFD-1001 (1 MB, `.d82`), the 2031, the
D9060/D9090 hard disks, and the 2022/4022/8023 printers. The two ways to emulate a drive:

1. **At the protocol level**: a device on the bus that understands TALK, LISTEN, the
   secondary addresses, OPEN/CLOSE, the command channel 15 and the error channel, and
   serves the files of a disk image or of a host folder. Small, fast and enough for
   almost all PET software, which barely copy protects. This is VICE's "virtual drive".
2. **At the chip level**: the drive's own computer, two 6502s (6502 + 6504) sharing RAM,
   two 6532 RIOTs, a 6522 and the GCR read/write logic, running the real DOS ROM. Exact,
   and the only way for software that sends code to the drive, but ten times the work and
   needs the drive ROMs.

izpet does 1 only. The bus is still modelled at the line level, which is what the
kernal drives and what keeps the devices honest about the handshake. The kernal has timeouts on the handshake (and BASIC 4 polls the error
channel), so a protocol level device still has to answer within emulated time rather than
instantly across a frame.

A **printer** device on address 4 can turn what it is sent into text or a PDF, like
izmac's `imagewriter/`.

### Loading programs directly

Most PET software circulates as `.prg` files. Putting one in memory at its load address
(`$0401` for BASIC) and fixing the BASIC pointers (whose zero page addresses differ between
BASIC 1 and BASIC 2/4) is the first, cheapest way to run things, before any drive exists.
Typing `RUN` through the keyboard buffer finishes an autostart.

### Other expansion

- **Option ROMs** at `$9000` and `$A000`: a configuration value per socket.
- **8096 banking**: a write only control register at `$FFF0` maps two of four 16 KB banks
  of the extra 64 KB over `$8000`–`$FFFF`, with bits to let the screen (`$8xxx`) and I/O
  (`$E8xx`) show through and to write protect each half *(verify the bit assignment)*.
- **8296 banking**: the same idea on a new board, plus the ROMs themselves can be mapped
  out, the 8296 having RAM under everything *(verify; the jumpers matter)*.
- **User port** devices: a parallel printer, a joystick adapter for games.
- High resolution boards (MTU Visible Memory, the 8296 HSG) are rare: no.

## Timing

Everything runs at 1 MHz and the 6502 is cycle counted by iz6502, so the machine can be
cycle exact cheaply. The retrace (from the TTL timing or the CRTC registers) is the only
periodic event; the VIA timers, the cassette edges and the CB2 sound fall out of the cycle
count. 50 Hz versus 60 Hz is decided by the editor ROM on CRTC boards, and by the board on
the TTL ones.

## What izpet can take from its siblings

| Need | Source | Work |
|---|---|---|
| 6502 | `iz6502` | none |
| 6522 VIA | `izmac/component/mos6522.go` | add CA2/CB2 as outputs (character set, IEEE, motor) and the free running shift out on CB2 (sound) |
| 6520 PIA | — | new, small: two ports, DDR selected by CRx bit 2, CA1/CA2/CB1/CB2 |
| 6545 CRTC | `izapple2/component/mc6845.go` | timing out of the registers, vertical sync |
| Cassette | `izapple2/cassette.go`, `storage/cassette.go`, `wavFile.go` | TAP reader, pulse widths instead of zero crossings |
| Audio | `izmac/audio/` or `izapple2/audio/` | none |
| Frontends | `izmac/frontend/macebiten`, `headless` | keyboard mapping, drop targets |
| Configuration | `izmac/configuration.go` (own `flag.FlagSet`) | a `-model` that sets the five axes, each overridable |
| Tests | izapple2 e2e style: run N cycles, read the screen as text | PETSCII screen codes to text |
| Embedded ROMs | `izapple2/resources.go`: `//go:embed resources`, `<internal>/` names | none |

The components are **copied**, as izmac copied them from izapple2, not shared through a
common module: each emulator evolves its chips for its own machine.

## Layout

One package per thing that knows nothing of the machine. Done so far:

| Path | Contents |
|---|---|
| `pet.go`, `petRun.go` | the machine and its run loop |
| `models.go` | the model table: the ROMs, the video board, the keyboard and the RAM of each model |
| `memoryManager.go` | the map, the ROMs, the screen mirrors, open bus |
| `ioE8Page.go` | the I/O page decoding and the wiring of the three chips and the CRTC |
| `resources.go`, `resources/` | the embedded ROMs, loaded by `<internal>/` name, as in izapple2 |
| `video.go` | the TTL and CRTC timing, the retrace signals, the snapshot of each frame and its image |
| `keyboard.go` | the two matrices, keys by position and characters typed |
| `sound.go` | CB2 to samples |
| `program.go` | loading `.prg` files once BASIC is ready |
| `screenText.go` | the screen as text, for the tests and the headless frontend |
| `configuration.go`, `command.go` | options and the command channel |
| `component/` | 6520, 6522, 6545 |
| `audio/` | the sample stream to the frontends, from izmac |
| `frontend/petebiten/`, `frontend/headless/` | the frontends |

To come:

| Path | Contents |
|---|---|
| `cassette.go` | the two decks, their sense and motor lines |
| `ieee488/` | the bus as wired lines, devices attach to it |
| `cbmdos/` | a protocol level drive: the command channel over a disk image or a host folder |
| `storage/` | D64, D80, D82, TAP, T64, WAV |

## Milestones

1. **Done.** A 2001-N/BASIC 2 to the `READY.` prompt on the headless frontend, with the
   screen read back as text in a test.
2. **Done.** Keyboard and the ebiten frontend; `.prg` loading.
3. **Done.** CRTC video: 4032 fat 40 and 8032, with 50/60 Hz and text/graphics mode.
4. **Done, but snow.** BASIC 1 and the original 2001: chiclet keyboard, the retrace wait.
5. Cassette: TAP, then WAV and T64.
6. IEEE-488 and a protocol level drive on D64/D80/D82 and host folders; a printer.
7. **Done.** Sound on CB2. Option ROMs are also in, with `-rom9` and `-roma`.
8. 8096 and 8296 banking.
9. The snow of the 2001, the 6545 status and light pen, CRTC changes within a frame.

## Decisions taken

- **ROMs are embedded**, as in izapple2: a `resources/` folder under `//go:embed`, loaded
  through `LoadResource` with `<internal>/` names, so the program runs with no download and
  the tests need no `test_images/` copy. The set: BASIC 1, 2 and 4; kernal 1, 2 and 4; the
  US editors (graphics/business × 40/80 columns × 50/60 Hz, as each BASIC had them); the two
  US character ROMs. A ROM can still be replaced by a file named on the command line.
  `//go:embed` needs a package level variable, the one exception to izmac's rule against
  them.
- **No SuperPET.** No 6809 core, no 6551, no 6702.
- **Drives at the protocol level only**: a CBM DOS device on the IEEE-488 bus serving
  D64/D80/D82 images and host folders. No drive ROMs, no 6504, no GCR.
- **Frontends: ebiten and headless**, like izmac.
- **Components are copied** from izapple2 and izmac, not shared.
- **US keyboards and character sets only.** 50 Hz stays, as a timing option of the editor
  ROMs, not as a national variant.

## References

- VICE `xpet` sources and documentation: the most complete account of every model, ROM and
  wiring detail, and the reference to compare against.
- *PET/CBM schematics* (2001, 2001-N 320349, 8032 8032087, 8296) on zimmers.net.
- *The PET Revealed*, Nick Hampshire; *Programming the PET/CBM*, Raeto West: the memory
  map, the zero page and the I/O.
- *All About Your 64K* / the 8096 and 8296 user guides for the banking register.
- ROM disassemblies of BASIC 1/2/4, the kernals and the editors.
- The IEEE-488 / CBM DOS manuals for the 4040 and 8050.
