package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"strings"

	"github.com/ivanizag/izpet"
)

func main() {
	err := run(os.Args[0], os.Args[1:])
	if izpet.IsHelpRequested(err) {
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(name string, args []string) error {
	config := izpet.NewConfiguration()

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	config.AddFlags(fs)
	frames := fs.Uint64("frames", 120,
		"frames to run before showing the screen")
	typeText := fs.String("type", "",
		"text to type once BASIC is ready, \\n for return")
	afterFrames := fs.Uint64("after", 120,
		"frames to run after typing the text")
	pngFile := fs.String("png", "",
		"file to write the screen to as a PNG image")

	err := fs.Parse(args)
	if err != nil {
		return err
	}
	config.AddFiles(fs.Args())
	err = config.Validate()
	if err != nil {
		return err
	}

	p, err := izpet.NewPET(config)
	if err != nil {
		return err
	}

	p.RunFrames(*frames)

	if *typeText != "" {
		p.TypeText(strings.ReplaceAll(*typeText, "\\n", "\r"))
		p.RunUntil(10_000, func() bool { return !p.IsTyping() })
		p.RunFrames(*afterFrames)
	}

	fmt.Print(p.GetScreenText())
	fmt.Printf("Ran %v frames, %v cycles. %v\n", p.GetFrames(), p.GetCycles(), p.DumpRegisters())

	if *pngFile != "" {
		f, err := os.Create(*pngFile)
		if err != nil {
			return err
		}
		defer f.Close()
		err = png.Encode(f, p.GetImage())
		if err != nil {
			return err
		}
	}
	return nil
}
