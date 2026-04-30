package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/adammcgrogan/dub/internal/ui"
)

func main() {
	// Create a cancellable context so that pressing Esc or Ctrl-C in the UI
	// can cleanly terminate the yt-dlp subprocess before the program exits.
	downloadCtx, cancelDownload := context.WithCancel(context.Background())
	defer cancelDownload()

	program := tea.NewProgram(ui.InitialModel(downloadCtx, cancelDownload))

	// ui.Program must be set before program.Run() is called so that the
	// yt-dlp background worker can send messages to the UI.
	ui.Program = program

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
