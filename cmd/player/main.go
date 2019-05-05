package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gizak/termui/widgets"

	ui "github.com/gizak/termui"

	"github.com/BurntSushi/toml"
	"github.com/lynges/fisymup"
	"github.com/lynges/susimup"
)

type config struct {
	Dest   string
	Source string
}

func absDirPath(slug string) (string, error) {
	var err error
	var basepath string
	basepath, err = filepath.Abs(slug)
	if err != nil {
		return "", errors.New(slug + " caused: " + err.Error())
	}
	err = os.MkdirAll(basepath, 0755)
	if err != nil {
		return "", errors.New("Could not create path for: " + basepath + " caused: " + err.Error())
	}
	return basepath, nil
}

func runSyncOp(grid *ui.Grid) {
	var conf config

	_, err := toml.DecodeFile("fisymup.conf", &conf)
	if err != nil {
		log.Fatal(err)
	}
	dest, err := absDirPath(conf.Dest)
	if err != nil {
		log.Fatal(err)
	}

	fisymup.StartSync(grid, conf.Source, dest)

}

func createGrid(x int, y int, width int, height int) *ui.Grid {
	resgrid := ui.NewGrid()
	resgrid.SetRect(x, y, width, height)
	return resgrid
}

func app() {
	var err error
	var basepath string

	err = ui.Init()

	if err != nil {
		panic(err)
	}
	defer ui.Close()

	if len(os.Args) > 1 {
		basepath, err = absDirPath(os.Args[1])
		if err != nil {
			log.Println(err)
			log.Println("Using cwd as basepath.")
		}
	} else {
		basepath, err = os.Getwd()
		if err != nil {
			fmt.Println(err)
			log.Fatal(err)
		}
	}

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	explain := widgets.NewParagraph()
	explain.Text = "You may now choose one of the actions below by pressing their corresponding hotkey as shown below."
	options := widgets.NewList()
	options.Rows = []string{
		"Action                             key",
		" ",
		"Sync files from dropbox              h",
		"Start the musicplayer                j",
	}

	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(1.0/4,
			ui.NewCol(1.0, explain),
		),
		ui.NewRow(1.0/4,
			ui.NewCol(1.0, options),
		),
	)

	ui.Render(grid)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			switch e.ID {
			case "h":
				ui.Clear()
				runSyncOp(createGrid(0, 0, termWidth, termHeight))
			case "j":
				ui.Clear()
				susimup.Start(basepath, createGrid(0, 0, termWidth, termHeight))
			case "½":
				ui.Clear()
				return
			}
		}
		ui.Render(grid)
	}

}

func main() {
	app()
	os.Exit(0)
}
