package tui

import (
	"errors"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/hpcsc/aws-profile/internal/config"
	"log"
)

func getDisplayableLabels(profiles []config.Profile) []string {
	var labels []string

	for _, profile := range profiles {
		labels = append(labels, profile.DisplayProfileName)
	}

	return labels
}

func SelectProfileFromList(profiles config.Profiles, pattern string) ([]byte, error) {
	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	filteredProfiles := profiles.Filter(pattern)
	labels := getDisplayableLabels(filteredProfiles)

	list := widgets.NewList()
	list.Title = "Select a AWS profile"
	list.Rows = labels
	list.SelectedRowStyle = ui.NewStyle(ui.ColorGreen)
	list.WrapText = true

	grid := ui.NewGrid()
	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	grid.Set(
		ui.NewRow(1.0/3,
			ui.NewCol(1.0, list),
		),
	)

	ui.Render(grid)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return nil, errors.New("cancelled by user")
		case "j", "<Down>":
			list.ScrollDown()
		case "k", "<Up>":
			list.ScrollUp()
		case "<Enter>":
			return []byte(filteredProfiles[list.SelectedRow].ProfileName), nil
		}

		ui.Render(grid)
	}
}
