package renderer

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/smford/tf-blast/pkg/analyzer"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF"))

	selectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#EE6FF8")).
				Background(lipgloss.Color("#353535"))

	detailPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1).
			MarginLeft(1)

	criticalStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF4D4D"))
	highStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFA500"))
	mediumStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EE6FF8"))
	lowStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF7F"))
	grayStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
)

type tuiModel struct {
	report   *analyzer.AnalysisReport
	cursor   int
	filter   string // "ALL", "CRITICAL", "HIGH", "MEDIUM", "LOW"
	filtered []analyzer.ResourceAnalysis
	expanded bool
	width    int
	height   int
	quitting bool
}

func newTUIModel(report *analyzer.AnalysisReport) tuiModel {
	m := tuiModel{
		report: report,
		filter: "ALL",
	}
	m.updateFiltered()
	return m
}

func (m *tuiModel) updateFiltered() {
	if m.filter == "ALL" {
		m.filtered = m.report.Resources
	} else {
		var filtered []analyzer.ResourceAnalysis
		for _, r := range m.report.Resources {
			if string(r.Severity) == m.filter {
				filtered = append(filtered, r)
			}
		}
		m.filtered = filtered
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
}

func (m tuiModel) Init() tea.Cmd {
	return nil
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}

		case "tab":
			filters := []string{"ALL", "CRITICAL", "HIGH", "MEDIUM", "LOW"}
			for i, f := range filters {
				if f == m.filter {
					m.filter = filters[(i+1)%len(filters)]
					break
				}
			}
			m.updateFiltered()

		case "enter", " ":
			m.expanded = !m.expanded
		}
	}

	return m, nil
}

func (m tuiModel) View() string {
	if m.quitting {
		return "Exiting tf-blast interactive explorer.\n"
	}

	var sb strings.Builder

	// Top Title Bar
	sb.WriteString(titleStyle.Render("💥 tf-blast Interactive Explorer"))
	sb.WriteString("  ")
	sb.WriteString(grayStyle.Render("Filter: [TAB] ") + m.filter + "  " + grayStyle.Render("Navigate: [↑/↓]  Details: [Enter]  Quit: [Q]"))
	sb.WriteString("\n\n")

	// Plan Health Banner
	sb.WriteString(fmt.Sprintf("%s  |  + %d to add  ~ %d to update  - %d to destroy  ± %d to replace\n",
		headerStyle.Render("Health: "+m.report.Summary.PlanHealth),
		m.report.Summary.ToAdd,
		m.report.Summary.ToUpdate,
		m.report.Summary.ToDestroy,
		m.report.Summary.ToReplace,
	))
	sb.WriteString(grayStyle.Render(fmt.Sprintf("Total Blast Radius: %d affected resource(s)", m.report.Summary.TotalBlastRadius)))
	if m.report.Summary.DriftCount > 0 {
		sb.WriteString(highStyle.Render(fmt.Sprintf("  |  ⚠️  %d drifted resource(s)", m.report.Summary.DriftCount)))
	}
	sb.WriteString("\n\n")

	if len(m.filtered) == 0 {
		sb.WriteString(grayStyle.Render("No resources match current filter."))
		return sb.String()
	}

	// Layout: Left List, Right Detail Pane
	var listBuilder strings.Builder
	for i, res := range m.filtered {
		cursor := "  "
		isCurrent := i == m.cursor
		if isCurrent {
			cursor = "> "
		}

		sevBadge := formatTUISeverity(res.Severity)
		addr := res.Address
		if len(addr) > 36 {
			addr = addr[:33] + "..."
		}

		row := fmt.Sprintf("%s%-10s %-36s %-12s", cursor, sevBadge, addr, string(res.Action))
		if isCurrent {
			listBuilder.WriteString(selectedItemStyle.Render(row))
		} else {
			listBuilder.WriteString(row)
		}
		listBuilder.WriteString("\n")
	}

	// Detail Pane for Current Selection
	var detailBuilder strings.Builder
	if m.cursor < len(m.filtered) {
		cur := m.filtered[m.cursor]
		detailBuilder.WriteString(headerStyle.Render(cur.Address) + "\n\n")
		detailBuilder.WriteString(fmt.Sprintf("Type:        %s\n", cur.Type))
		detailBuilder.WriteString(fmt.Sprintf("Action:      %s\n", cur.Action))
		detailBuilder.WriteString(fmt.Sprintf("Severity:    %s\n", cur.Severity))
		detailBuilder.WriteString(fmt.Sprintf("Root Cause:  %s\n", cur.RootCause))
		detailBuilder.WriteString(fmt.Sprintf("Downstream:  %d dependent(s)\n", cur.DownstreamCount))

		if cur.HasDrift {
			detailBuilder.WriteString(highStyle.Render("⚠️  State Drift: Out-of-band console changes detected\n"))
		}

		if cur.DownstreamTree != nil && len(cur.DownstreamTree.Children) > 0 {
			detailBuilder.WriteString("\n" + headerStyle.Render("Cascading Blast Tree:") + "\n")
			treeStr := cur.DownstreamTree.RenderTree()
			detailBuilder.WriteString(grayStyle.Render(treeStr))
		}
	}

	leftPane := listBuilder.String()
	rightPane := detailPaneStyle.Render(detailBuilder.String())

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	sb.WriteString(content)

	return sb.String()
}

func formatTUISeverity(s analyzer.Severity) string {
	switch s {
	case analyzer.SeverityCritical:
		return criticalStyle.Render("CRITICAL")
	case analyzer.SeverityHigh:
		return highStyle.Render("HIGH")
	case analyzer.SeverityMedium:
		return mediumStyle.Render("MEDIUM")
	case analyzer.SeverityLow:
		return lowStyle.Render("LOW")
	default:
		return grayStyle.Render("CLEAN")
	}
}

// RenderInteractive launches the terminal interactive TUI.
func RenderInteractive(report *analyzer.AnalysisReport) error {
	p := tea.NewProgram(newTUIModel(report), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
