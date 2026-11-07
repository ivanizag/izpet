package main

import (
	"fmt"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ivanizag/izpet"
)

// helpMessage is the text shown with F1
const helpMessage = `
          F1: Show/Hide help
     Ctrl-F2: Reset
      F1, F2: Reset
          F4: Show/Hide CPU trace
          F5: Fast/Normal speed
     Ctrl-F5: Show speed
          F7: Show/Hide the keyboard
         F12: Save screen snapshot
       Pause: Pause the emulation

      Escape: RUN/STOP
          F3: RVS
          F9: ESC, business keyboard
      Insert: INST
        Home: HOME, with shift CLR

 Click on the keys of the keyboard
 Drop a .prg file to load and run it

    Run izpet -h for more options
   https://github.com/ivanizag/izpet
`

// game drives the ebiten main loop. The emulation runs on its own goroutine,
// this one only reads the screen and pushes the input.
type game struct {
	p        *izpet.PET
	image    *ebiten.Image
	keyboard *ebitenKeyboard
	diagram  *ebitenKeyboardDiagram
	title    string

	width        int // Size of the image of the PET screen
	height       int
	scaleX       float64
	updates      uint64
	showHelp     bool
	showKeyboard bool

	// The window in pixels of the device, the contents are drawn scaled
	// to fit and centered
	canvasWidth  int
	canvasHeight int
}

func (g *game) Update() error {
	g.keyboard.update()
	if g.showKeyboard {
		x, y := ebiten.CursorPosition()
		scale, offsetX, offsetY := g.contentPlacement()
		top := offsetY + float64(g.screenHeight())*scale
		g.diagram.update(float64(x)-offsetX, float64(y)-top, int(float64(g.screenWidth())*scale))
	}

	for _, f := range droppedFiles() {
		g.p.SendLoadProgram(f)
	}

	if g.showHelp {
		g.image.WritePixels(g.p.GetMessageImage(helpMessage).Pix)
	} else if !g.p.IsPaused() {
		g.image.WritePixels(g.p.GetImage().Pix)
	}

	if g.updates%60 == 0 {
		ebiten.SetWindowTitle(g.windowTitle())
	}
	g.updates++
	return nil
}

func (g *game) Draw(dst *ebiten.Image) {
	scale, offsetX, offsetY := g.contentPlacement()

	var op ebiten.DrawImageOptions
	op.GeoM.Scale(g.scaleX*scale, 2*scale)
	op.GeoM.Translate(offsetX, offsetY)
	dst.DrawImage(g.image, &op)

	if g.showKeyboard {
		g.diagram.draw(dst, offsetX, offsetY+float64(g.screenHeight())*scale,
			int(float64(g.screenWidth())*scale))
	}
}

// Layout works in the pixels of the device, for the keyboard to be drawn
// sharp at any size
func (g *game) Layout(outsideWidth int, outsideHeight int) (int, int) {
	factor := ebiten.Monitor().DeviceScaleFactor()
	g.canvasWidth = int(float64(outsideWidth) * factor)
	g.canvasHeight = int(float64(outsideHeight) * factor)
	return g.canvasWidth, g.canvasHeight
}

// contentPlacement returns the scale and the position of the contents to fit
// in the window
func (g *game) contentPlacement() (scale, offsetX, offsetY float64) {
	width, height := g.contentSize()
	scale = min(float64(g.canvasWidth)/float64(width), float64(g.canvasHeight)/float64(height))
	offsetX = (float64(g.canvasWidth) - float64(width)*scale) / 2
	offsetY = (float64(g.canvasHeight) - float64(height)*scale) / 2
	return scale, offsetX, offsetY
}

/*
contentSize is the size of the window contents: the screen with the pixels in
their shape, each scan line drawn twice and the pixels of the 40 columns
models twice as wide as the ones of the 80 columns models; and below it the
keyboard, as wide as the screen.
*/
func (g *game) contentSize() (int, int) {
	width := g.screenWidth()
	height := g.screenHeight()
	if g.showKeyboard {
		height += int(float64(width) * g.diagram.aspect())
	}
	return width, height
}

func (g *game) screenWidth() int {
	return int(float64(g.width) * g.scaleX)
}

func (g *game) screenHeight() int {
	return g.height * 2
}

// toggleKeyboard shows or hides the keyboard, the window grows or shrinks to
// keep the size of the screen
func (g *game) toggleKeyboard() {
	_, oldHeight := g.contentSize()
	g.showKeyboard = !g.showKeyboard
	_, height := g.contentSize()
	windowWidth, windowHeight := ebiten.WindowSize()
	ebiten.SetWindowSize(windowWidth, windowHeight*height/oldHeight)
}

func (g *game) windowTitle() string {
	if g.p.IsPaused() {
		return g.title + " - PAUSED"
	}
	speed := fmt.Sprintf("%0.2f MHz", g.p.GetCurrentFreqMHz())
	if g.p.IsFullSpeed() {
		speed += ", full speed"
	}
	return fmt.Sprintf("%v - %v (F1 for help)", g.title, speed)
}

// droppedFiles returns the paths of the files dropped on the window
func droppedFiles() []string {
	dropped := ebiten.DroppedFiles()
	if dropped == nil {
		return nil
	}
	entries, err := fs.ReadDir(dropped, ".")
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		file, err := dropped.Open(entry.Name())
		if err != nil {
			continue
		}
		if real, ok := file.(*os.File); ok {
			names = append(names, real.Name())
		}
		file.Close()
	}
	return names
}

// saveScreenshot writes the screen to a PNG file on the working directory
func saveScreenshot(p *izpet.PET) {
	name := fmt.Sprintf("izpet_%v.png", time.Now().Format("20060102_150405"))
	f, err := os.Create(name)
	if err != nil {
		fmt.Printf("Error saving the screen: %v\n", err)
		return
	}
	defer f.Close()
	err = png.Encode(f, p.GetImage())
	if err != nil {
		fmt.Printf("Error saving the screen: %v\n", err)
		return
	}
	abs, _ := filepath.Abs(name)
	fmt.Printf("Screen saved to %v\n", abs)
}

func main() {
	config := izpet.NewConfiguration()
	err := config.ParseFlags(os.Args[0], os.Args[1:], os.Stderr)
	if izpet.IsHelpRequested(err) {
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	p, err := izpet.NewPET(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	err = ebitenRun(p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func ebitenRun(p *izpet.PET) error {
	sound, err := newEbitenAudio(p)
	if err != nil {
		return err
	}

	diagram, err := newEbitenKeyboardDiagram(p)
	if err != nil {
		return err
	}

	size := p.GetImage().Bounds().Size()
	g := &game{
		diagram:      diagram,
		showKeyboard: true,
		p:            p,
		image:        ebiten.NewImage(size.X, size.Y),
		title:        "izpet - " + strings.Split(p.Name, ",")[0],
		width:        size.X,
		height:       size.Y,
		scaleX:       2,
	}
	if p.Is80Columns() {
		g.scaleX = 1
	}
	g.keyboard = newEbitenKeyboard(p, g)

	w, h := g.contentSize()
	ebiten.SetWindowSize(w, h)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle(g.title + " (F1 for help)")

	sound.start()

	stopped := make(chan struct{})
	go func() {
		p.Run()
		close(stopped)
	}()
	defer func() {
		p.SendCommand(izpet.CommandKill)
		<-stopped
	}()

	return ebiten.RunGame(g)
}
