package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	ui "github.com/gizak/termui"
	"github.com/gizak/termui/widgets"
	"github.com/lynges/susimup"
)

func main() {
	var err error
	var basepath string

	err = ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	if len(os.Args) > 1 {
		basepath, err = filepath.Abs(os.Args[1])
		if err != nil {
			log.Println(os.Args[1] + " caused: " + err.Error() + "\n Using cwd as basepath.")
		}
	} else {
		basepath, err = os.Getwd()
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
	}
	susimup.Start(basepath, false)
	tf := widgets.NewParagraph()
	tf.Text = "Noget tekst"
	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(1.0,
			ui.NewCol(1.0, tf),
		),
	)

	ui.Render(grid)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			break
		}
	}
}
