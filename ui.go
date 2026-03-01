package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// --- STYLES ---
var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1).MarginBottom(1)
	activeTabStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#01FAC6")).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(lipgloss.Color("#01FAC6")).Padding(0, 2)
	tabStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Padding(0, 2)
	cursorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#01FAC6")).Bold(true)
	checkedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true)
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).MarginTop(1)
	successStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	footerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B")).Bold(true).MarginTop(1)
	warningStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Border(lipgloss.RoundedBorder()).Padding(1, 2).MarginTop(1)
	spinnerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#01FAC6")).Bold(true)
	projectStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A371F7")).Italic(true)
)

func formatBytes(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "kMGTPE"[exp])
}

// View renders the application UI.
func (m model) View() string {
	if m.quitting {
		if m.finalMsg != "" {
			return successStyle.Render(m.finalMsg) + "\n"
		}
		return helpStyle.Render("Exited without changes.") + "\n"
	}

	// Sub-views for Logs or Inspect
	if m.state == stateLogs || m.state == stateInspect {
		if m.isProcessing {
			return fmt.Sprintf("\n %s Loading data. Please wait...\n", spinnerStyle.Render(m.spinner.View()))
		}
		header := titleStyle.Render(m.detailTitle)
		footer := "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("esc/q - back • ↑/↓ - scroll")
		return header + "\n" + m.viewport.View() + footer
	}

	if m.isProcessing {
		return fmt.Sprintf("\n %s Processing selected items. Please wait...\n", spinnerStyle.Render(m.spinner.View()))
	}

	var globalSelectedSize int64
	totalItemsSelected := len(m.selectedC) + len(m.selectedI) + len(m.selectedV) + len(m.selectedN)
	for _, c := range m.selectedC {
		globalSelectedSize += c.SizeRw
	}
	for _, img := range m.selectedI {
		globalSelectedSize += img.Size
	}

	if m.confirming {
		warningText := fmt.Sprintf("⚠️  WARNING: You are about to delete %d items.\n\nTotal space to free: %s\n\nAre you sure? (y/N)", totalItemsSelected, formatBytes(globalSelectedSize))
		return warningStyle.Render(warningText) + "\n"
	}

	s := titleStyle.Render("🧹 Docker Cleaner") + "\n\n"

	// Restored Tab labels with counts
	tabLabels := []string{
		fmt.Sprintf("Containers (%d)", len(m.containers)),
		fmt.Sprintf("Images (%d)", len(m.images)),
		fmt.Sprintf("Volumes (%d)", len(m.volumes)),
		fmt.Sprintf("Networks (%d)", len(m.networks)),
	}

	renderedTabs := make([]string, 4)
	for i, t := range tabLabels {
		if m.activeTab == i {
			renderedTabs[i] = activeTabStyle.Render(t)
		} else {
			renderedTabs[i] = tabStyle.Render(t)
		}
	}
	s += lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs[0], renderedTabs[1], renderedTabs[2], renderedTabs[3]) + "\n\n"

	maxVisible := m.height - 13
	if maxVisible < 5 {
		maxVisible = 5
	}

	t := m.activeTab
	cursor := m.cursors[t]
	offset := m.offsets[t]
	end := offset + maxVisible

	// Render Containers
	if t == 0 {
		fc := m.getFilteredContainers()
		if len(fc) == 0 {
			s += "No containers match.\n"
		}
		if end > len(fc) {
			end = len(fc)
		}
		for i := offset; i < end; i++ {
			c := fc[i]
			cIcon, checked, rowStyle := " ", " ", lipgloss.NewStyle()
			if cursor == i {
				cIcon, rowStyle = ">", cursorStyle
			}
			if _, ok := m.selectedC[c.ID]; ok {
				checked, rowStyle = "x", checkedStyle
			}

			stateIcon := "🟢"
			if c.State == "exited" || c.State == "dead" {
				stateIcon = "🔴"
			}

			project := c.Labels["com.docker.compose.project"]
			projStr := ""
			if project != "" {
				projStr = projectStyle.Render("[" + project + "] ")
			}

			name := "Unknown"
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			row := fmt.Sprintf("%s [%s] %s %s%-20s (%s) [%s]", cIcon, checked, stateIcon, projStr, name, c.ID[:8], formatBytes(c.SizeRw))
			s += rowStyle.Render(row) + "\n"
		}
	} else if t == 1 {
		// Render Images
		fi := m.getFilteredImages()
		if len(fi) == 0 {
			s += "No images match.\n"
		}
		if end > len(fi) {
			end = len(fi)
		}
		for i := offset; i < end; i++ {
			img := fi[i]
			cIcon, checked, rowStyle := " ", " ", lipgloss.NewStyle()
			if cursor == i {
				cIcon, rowStyle = ">", cursorStyle
			}
			if _, ok := m.selectedI[img.ID]; ok {
				checked, rowStyle = "x", checkedStyle
			}

			name := "<none>"
			if len(img.RepoTags) > 0 {
				name = img.RepoTags[0]
			}
			shortID := img.ID
			if len(shortID) > 17 {
				shortID = shortID[7:17]
			}
			row := fmt.Sprintf("%s [%s] %-40s (%s) [%s]", cIcon, checked, name, shortID, formatBytes(img.Size))
			s += rowStyle.Render(row) + "\n"
		}
	} else if t == 2 {
		// Render Volumes
		fv := m.getFilteredVolumes()
		if len(fv) == 0 {
			s += "No volumes match.\n"
		}
		if end > len(fv) {
			end = len(fv)
		}
		for i := offset; i < end; i++ {
			v := fv[i]
			cIcon, checked, rowStyle := " ", " ", lipgloss.NewStyle()
			if cursor == i {
				cIcon, rowStyle = ">", cursorStyle
			}
			if _, ok := m.selectedV[v.Name]; ok {
				checked, rowStyle = "x", checkedStyle
			}
			name := v.Name
			if len(name) > 30 {
				name = name[:27] + "..."
			}
			row := fmt.Sprintf("%s [%s] %-35s (Driver: %s)", cIcon, checked, name, v.Driver)
			s += rowStyle.Render(row) + "\n"
		}
	} else if t == 3 {
		// Render Networks
		fn := m.getFilteredNetworks()
		if len(fn) == 0 {
			s += "No networks match.\n"
		}
		if end > len(fn) {
			end = len(fn)
		}
		for i := offset; i < end; i++ {
			n := fn[i]
			cIcon, checked, rowStyle := " ", " ", lipgloss.NewStyle()
			if cursor == i {
				cIcon, rowStyle = ">", cursorStyle
			}
			if _, ok := m.selectedN[n.ID]; ok {
				checked, rowStyle = "x", checkedStyle
			}
			row := fmt.Sprintf("%s [%s] %-20s (Driver: %s, Scope: %s)", cIcon, checked, n.Name, n.Driver, n.Scope)
			s += rowStyle.Render(row) + "\n"
		}
	}

	// Search bar
	if m.textInput.Focused() || m.textInput.Value() != "" {
		s += "\n🔍 Search: " + m.textInput.View() + "\n"
	} else {
		s += "\n"
	}

	// Footer
	s += footerStyle.Render(fmt.Sprintf("\n💾 Global Space to free: %s (%d items selected)", formatBytes(globalSelectedSize), totalItemsSelected)) + "\n"

	// Restored explicitly typed Help Menu
	if t == 0 {
		s += helpStyle.Render("Search: / • Tabs: Tab/Left/Right • Stop: s • Restart: r • Logs: L • Inspect: i • Prune: p • Select: Space/a • Delete: Enter • Quit: q") + "\n"
	} else {
		s += helpStyle.Render("Search: / • Tabs: Tab/Left/Right • Prune: p (Images) • Select: Space/a • Delete: Enter • Quit: q") + "\n"
	}

	return s
}
