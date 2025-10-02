package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/qrave1/go-service-bootstrap/internal/tui"
)

func main() {
	p := tea.NewProgram(tui.InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Houston, we have an error: %v", err)
		os.Exit(1)
	}
}
