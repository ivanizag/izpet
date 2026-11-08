package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.design/x/clipboard"

	"github.com/ivanizag/izpet"
)

/*
Pasting the text of the clipboard of the host on the PET. It is read on its
own goroutine, the clipboard may take a while to answer, and typed by the
emulator, see izpet.PET.SendPaste.
*/

const clipboardTimeout = 2 * time.Second

var (
	clipboardOnce  sync.Once
	clipboardError error
)

// pasteClipboard types the text of the clipboard on the PET. It returns at
// once, the errors are reported on the console.
func pasteClipboard(p *izpet.PET) {
	go func() {
		clipboardOnce.Do(func() {
			clipboardError = clipboard.Init()
		})
		if clipboardError != nil {
			fmt.Printf("No clipboard to paste from: %v\n", clipboardError)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout)
		defer cancel()
		text, err := clipboard.Read(ctx, clipboard.FmtText)
		if err != nil {
			fmt.Printf("Error reading the clipboard: %v\n", err)
			return
		}
		p.SendPaste(string(text))
	}()
}
