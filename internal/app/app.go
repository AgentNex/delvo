package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/AgentNex/delvo/internal/config"
	"github.com/AgentNex/delvo/internal/index"
)

// Run starts Delvo CLI's interactive full-screen TUI.
func Run(cfg *config.Config) error {
	idx, err := index.New(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open index at %s: %w", cfg.DBPath, err)
	}
	defer idx.Close()

	model, err := NewModel(cfg, idx)
	if err != nil {
		return fmt.Errorf("initialize model: %w", err)
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run program: %w", err)
	}

	return nil
}
