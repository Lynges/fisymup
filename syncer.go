package fisymup

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	ui "github.com/gizak/termui"
	"github.com/gizak/termui/widgets"

	// Must be imported this way for rclone to work
	_ "github.com/ncw/rclone/backend/dropbox"
	_ "github.com/ncw/rclone/backend/local"
	"github.com/ncw/rclone/fs"
	"github.com/ncw/rclone/fs/accounting"
	fslog "github.com/ncw/rclone/fs/log"
	rsync "github.com/ncw/rclone/fs/sync"
)

var (
	progressMu sync.Mutex
)

type progressStats struct {
	bytes          int64
	transfers      int64
	logMessages    []string
	statusWidget   *widgets.Paragraph
	bytesWidget    *widgets.Paragraph
	transferWidget *widgets.Paragraph
	logWidget      *widgets.Paragraph
}

func newProgressStats() *progressStats {
	ps := new(progressStats)

	ps.bytes = 0
	ps.transfers = 0
	ps.logMessages = []string{}

	ps.statusWidget = widgets.NewParagraph()
	ps.statusWidget.Title = "Status:"
	ps.statusWidget.Text = "Waiting for sync process to start..."

	ps.bytesWidget = widgets.NewParagraph()
	ps.bytesWidget.Title = "Bytes transferred:"
	ps.bytesWidget.Text = "0"

	ps.transferWidget = widgets.NewParagraph()
	ps.transferWidget.Title = "Files tranferred:"
	ps.transferWidget.Text = "0"

	ps.logWidget = widgets.NewParagraph()
	ps.logWidget.Title = "Log messages:"
	ps.logWidget.Text = ""
	ps.logWidget.BorderLeft = false
	ps.logWidget.BorderRight = false
	ps.logWidget.BorderBottom = false

	return ps
}

func (ps *progressStats) getStats() {
	progressMu.Lock()
	defer progressMu.Unlock()
	ps.bytes = accounting.Stats.GetBytes()
	ps.transfers = accounting.Stats.GetTransfers()
}

func (ps *progressStats) updateBytes() {
	ps.bytesWidget.Text = fmt.Sprintf("%.2f MB", float64(ps.bytes)/1000000)
}

func (ps *progressStats) updateTransfers() {
	ps.transferWidget.Text = fmt.Sprintf("%d files tranferred", ps.transfers)
}

func (ps *progressStats) updateLog() {
	ps.logWidget.Text = strings.Join(ps.logMessages, "\n")
}

func (ps *progressStats) update() {
	ps.getStats()
	ps.updateBytes()
	ps.updateTransfers()
	ps.updateLog()
}

func (ps *progressStats) addLogMessage(msg string) {
	ps.logMessages = append(ps.logMessages, msg)
	if len(ps.logMessages) > 4 {
		ps.logMessages = ps.logMessages[1:]
	}
}

const (
	// interval between progress prints
	defaultProgressInterval = 500 * time.Millisecond
	// time format for logging
	logTimeFormat = "2006-01-02 15:04:05"
)

func newEmptyParagraph() *widgets.Paragraph {
	p := widgets.NewParagraph()
	p.Border = false
	return p
}

func progressMonitor() func() {
	stats := newProgressStats()

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()

	grid.SetRect(0, 0, termWidth, termHeight)
	grid.Set(
		ui.NewRow(1.0/12*1,
			ui.NewCol(1.0, newEmptyParagraph()),
		),
		ui.NewRow(1.0/12*1,
			ui.NewCol(1.0/6, newEmptyParagraph()),
			ui.NewCol(1.0/3, stats.bytesWidget),
			ui.NewCol(1.0/3, stats.transferWidget),
			ui.NewCol(1.0/6, newEmptyParagraph()),
		),
		ui.NewRow(1.0/12*1,
			ui.NewCol(1.0, newEmptyParagraph()),
		),
		ui.NewRow(1.0/12*1,
			ui.NewCol(1.0/4, newEmptyParagraph()),
			ui.NewCol(1.0/2, stats.statusWidget),
			ui.NewCol(1.0/4, newEmptyParagraph()),
		),
		ui.NewRow(1.0/12*5,
			ui.NewCol(1.0, newEmptyParagraph()),
		),
		ui.NewRow(1.0/12*3,
			ui.NewCol(1.0, stats.logWidget),
		),
	)
	ui.Render(grid)
	stopStats := make(chan struct{})

	oldLogPrint := fs.LogPrint
	if !fslog.Redirected() {
		// Intercept the log calls if not logging to file or syslog
		fs.LogPrint = func(level fs.LogLevel, text string) {
			stats.addLogMessage(fmt.Sprintf("%s %-6s: %s", time.Now().Format(logTimeFormat), level, text))
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		stats.statusWidget.Text = "Downloading from source..."
		progressInterval := defaultProgressInterval

		ticker := time.NewTicker(progressInterval)
		for {
			select {
			case <-ticker.C:
				stats.update()
				ui.Render(grid)
			case <-stopStats:
				ticker.Stop()
				stats.update()
				fs.LogPrint = oldLogPrint
				fmt.Println("")
				ui.Render(grid)
				return
			}
		}
	}()
	return func() {
		newstyle := ui.NewStyle(ui.ColorBlack, ui.ColorGreen)
		if accounting.Stats.HadFatalError() {
			stats.statusWidget.Text = "Something went wrong. Check the log below and then press q to return to the menu"
			newstyle = ui.NewStyle(ui.ColorBlack, ui.ColorRed)
		} else {
			if stats.bytes > 0 {
				stats.statusWidget.Text = "Download done. Press q to return to the menu"
			} else {
				stats.statusWidget.Text = "All files up to date, nothing was downloaded. Press q to return to the menu"
			}
		}

		stats.statusWidget.TextStyle = newstyle

		close(stopStats)
		wg.Wait()
		for e := range ui.PollEvents() {
			if e.Type == ui.KeyboardEvent {
				switch e.ID {
				case "q":
					return
				}
			}
		}
	}
}

// StartSync starts the syncing process.
// uses the provided termui grid to render the progress.
// src and dst must be formatted according to remote and local for rclone
func StartSync(grid *ui.Grid, srcpath string, dstpath string) {
	var errsrc, errdst error
	fsrc, errsrc := fs.NewFs(srcpath)
	fdst, errdst := fs.NewFs(dstpath)

	if errdst != nil {
		log.Fatal(errdst)
	}
	if errsrc != nil {
		log.Fatal(errsrc)
	}
	stopStats := progressMonitor()

	rsync.Sync(fdst, fsrc, false)

	stopStats()

}

// TestDropboxConnection attempts to read from dropbox.com
func TestDropboxConnection() bool {
	_, err := http.Get("https://www.dropbox.com/")
	if err != nil {
		return false
	}
	return true
}
