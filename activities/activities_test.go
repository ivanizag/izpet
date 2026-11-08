package activities

import (
	"os"
	"testing"
)

/*
The screenshots of the activities in doc/activities, made by taking a machine
through what each page tells its reader to do. It is not a test: it does
nothing unless asked for with IZPET_ACTIVITY_SCREENSHOTS, and then writes the
images of each page in doc/activities/images, so that they can be made again
when the emulator or the instructions change.

	IZPET_ACTIVITY_SCREENSHOTS=1 go test -run TestActivityScreenshots ./activities
*/

func TestActivityScreenshots(t *testing.T) {
	if os.Getenv("IZPET_ACTIVITY_SCREENSHOTS") == "" {
		t.Skip("this makes the screenshots of doc/activities")
	}
	t.Run("models", modelsScreenshots)
	t.Run("first-program", firstProgramScreenshots)
	t.Run("drawing", drawingScreenshots)
	t.Run("cassette", cassetteScreenshots)
	t.Run("disk", diskScreenshots)
	t.Run("monitor", monitorScreenshots)
	t.Run("wordpro", wordProScreenshots)
	t.Run("visicalc", visiCalcScreenshots)
	t.Run("dog-star", dogStarScreenshots)
}
