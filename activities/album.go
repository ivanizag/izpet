/*
Package activities makes the pictures of izpet's activities, the pages of
doc/activities that walk through what a PET owner did: an Album writes
screenshots and recordings of a machine into a folder, in the frame they all
have, and a Recording is a GIF of the screen taken as the machine is driven.

The activities of izpet's own documentation are in this package's tests, and
are made with:

	IZPET_ACTIVITY_SCREENSHOTS=1 go test -run TestActivityScreenshots ./activities

Anyone writing their own does it the same way: a machine from izpet.NewPET,
driven with TypeText and PutKey, and an Album for what it shows on the way.
*/
package activities

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/ivanizag/izpet"
	"github.com/ivanizag/izpet/frontend/shared"
)

/*
An Album is the folder of the pictures of one activity. Screenshots are PNG
files and recordings GIF files, named after what they show, in the frame of
Frame. The folder is made with the first picture.
*/
type Album struct {
	folder string
}

// NewAlbum is an album in a folder
func NewAlbum(folder string) *Album {
	return &Album{folder: folder}
}

// Path is where a picture of the album is, by its name and extension
func (a *Album) Path(file string) string {
	return filepath.Join(a.folder, file)
}

// create makes a file of the album, and the folder when it is not there yet
func (a *Album) create(file string) (*os.File, error) {
	if err := os.MkdirAll(a.folder, 0o755); err != nil {
		return nil, err
	}
	return os.Create(a.Path(file))
}

// Screenshot writes the screen of a machine as it is now
func (a *Album) Screenshot(p *izpet.PET, name string) error {
	return a.Write(Frame(screenOf(p), ""), name)
}

// ScreenshotOf writes the screen of a machine with a label in its frame
func (a *Album) ScreenshotOf(p *izpet.PET, name string, label string) error {
	return a.Write(Frame(screenOf(p), label), name)
}

// machineGap is the space between the screen and the keyboard of a machine
const machineGap = 12

/*
Machine writes the screen of a machine, in its frame with a label, over the
drawing of its keyboard, as wide as the frame: the picture of a model, where
what tells it from the others is seen at once.
*/
func (a *Album) Machine(p *izpet.PET, name string, label string) error {
	framed := Frame(screenOf(p), label)
	width := framed.Bounds().Dx()
	diagram, err := shared.NewKeyboardDiagram(p)
	if err != nil {
		return err
	}
	keyboard := diagram.Render(width)

	height := framed.Bounds().Dy() + machineGap + keyboard.Bounds().Dy()
	out := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(out, framed.Bounds(), framed, image.Point{}, draw.Src)
	at := image.Rect(0, framed.Bounds().Dy()+machineGap, width, height)
	draw.Draw(out, at, keyboard, image.Point{}, draw.Src)
	return a.Write(out, name)
}

// Write writes a picture as it is
func (a *Album) Write(picture image.Image, name string) error {
	f, err := a.create(name + ".png")
	if err != nil {
		return err
	}
	if err := png.Encode(f, picture); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// SaveRecording writes a recording as a GIF, its last picture held for
// longer, by hold hundredths of a second
func (a *Album) SaveRecording(r *Recording, name string, hold int) error {
	f, err := a.create(name + ".gif")
	if err != nil {
		return err
	}
	if err := r.Encode(f, hold); err != nil {
		f.Close()
		return fmt.Errorf("the recording %v: %w", name, err)
	}
	return f.Close()
}
