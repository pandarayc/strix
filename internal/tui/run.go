package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/raydraw/ergate/internal/config"
	"github.com/raydraw/ergate/internal/engine"
	"github.com/raydraw/ergate/internal/session"
)

// Run starts the bubbletea TUI.
func Run(cfg *config.Config, eng *engine.Engine) error {
	store, _ := session.NewStore(cfg.SessionDir)
	m := NewModel(cfg, eng, store)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
