package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/liubog2008/tmux-agent/internal/ui"
)

func main() {
	program := tea.NewProgram(ui.NewModel())
	if _, err := program.Run(); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatal(err)
	}
}
