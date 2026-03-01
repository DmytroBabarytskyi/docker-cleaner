package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
)

// --- FILTERING HELPERS ---
func (m model) getFilteredContainers() []container.Summary {
	if !m.textInput.Focused() && m.textInput.Value() == "" {
		return m.containers
	}
	var res []container.Summary
	q := strings.ToLower(m.textInput.Value())
	for _, c := range m.containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
		}
		project := c.Labels["com.docker.compose.project"]
		if strings.Contains(strings.ToLower(name), q) || strings.Contains(strings.ToLower(project), q) {
			res = append(res, c)
		}
	}
	return res
}

func (m model) getFilteredImages() []image.Summary {
	if !m.textInput.Focused() && m.textInput.Value() == "" {
		return m.images
	}
	var res []image.Summary
	q := strings.ToLower(m.textInput.Value())
	for _, img := range m.images {
		name := "<none>"
		if len(img.RepoTags) > 0 {
			name = img.RepoTags[0]
		}
		if strings.Contains(strings.ToLower(name), q) {
			res = append(res, img)
		}
	}
	return res
}

func (m model) getFilteredVolumes() []*volume.Volume {
	if !m.textInput.Focused() && m.textInput.Value() == "" {
		return m.volumes
	}
	var res []*volume.Volume
	q := strings.ToLower(m.textInput.Value())
	for _, v := range m.volumes {
		if strings.Contains(strings.ToLower(v.Name), q) {
			res = append(res, v)
		}
	}
	return res
}

func (m model) getFilteredNetworks() []network.Summary {
	if !m.textInput.Focused() && m.textInput.Value() == "" {
		return m.networks
	}
	var res []network.Summary
	q := strings.ToLower(m.textInput.Value())
	for _, n := range m.networks {
		if strings.Contains(strings.ToLower(n.Name), q) {
			res = append(res, n)
		}
	}
	return res
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 6

	case stateChangeMsg:
		for i, c := range m.containers {
			if c.ID == msg.id {
				m.containers[i].State = msg.state
				break
			}
		}
		return m, nil

	case logsMsg:
		m.viewport.SetContent(string(msg))
		m.viewport.GotoTop()
		m.isProcessing = false
		return m, nil

	case inspectMsg:
		m.viewport.SetContent(string(msg))
		m.viewport.GotoTop()
		m.isProcessing = false
		return m, nil

	case deleteCompleteMsg:
		m.isProcessing = false
		m.quitting = true
		m.finalMsg = msg.msg
		return m, tea.Quit

	case spinner.TickMsg:
		if m.isProcessing {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	// Logs & Inspect State Navigation
	if m.state == stateLogs || m.state == stateInspect {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "q", "esc":
				m.state = stateList // Back to main list
				return m, nil
			}
		}
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	if m.isProcessing {
		return m, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.textInput.Focused() {
			switch msg.String() {
			case "enter", "esc":
				m.textInput.Blur()
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			m.cursors[m.activeTab] = 0
			m.offsets[m.activeTab] = 0
			return m, cmd
		}

		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.isProcessing = true
				return m, tea.Batch(m.spinner.Tick, m.runAsyncDeletion())
			case "n", "N", "esc":
				m.confirming = false
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		var listLen int
		switch m.activeTab {
		case 0:
			listLen = len(m.getFilteredContainers())
		case 1:
			listLen = len(m.getFilteredImages())
		case 2:
			listLen = len(m.getFilteredVolumes())
		case 3:
			listLen = len(m.getFilteredNetworks())
		}

		maxVisible := m.height - 13
		if maxVisible < 5 {
			maxVisible = 5
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "/":
			m.textInput.Focus()
			return m, nil
		case "tab", "right":
			m.activeTab = (m.activeTab + 1) % 4
		case "left":
			m.activeTab = (m.activeTab - 1 + 4) % 4

		// Up/Down movement
		case "up", "k":
			if m.cursors[m.activeTab] > 0 {
				m.cursors[m.activeTab]--
				if m.cursors[m.activeTab] < m.offsets[m.activeTab] {
					m.offsets[m.activeTab] = m.cursors[m.activeTab]
				}
			}
		case "down", "j":
			if m.cursors[m.activeTab] < listLen-1 {
				m.cursors[m.activeTab]++
				if m.cursors[m.activeTab] >= m.offsets[m.activeTab]+maxVisible {
					m.offsets[m.activeTab] = m.cursors[m.activeTab] - maxVisible + 1
				}
			}

		// Actions
		case "s":
			if m.activeTab == 0 && listLen > 0 {
				return m, stopContainerCmd(m.cli, m.getFilteredContainers()[m.cursors[0]].ID)
			}
		case "r":
			if m.activeTab == 0 && listLen > 0 {
				return m, restartContainerCmd(m.cli, m.getFilteredContainers()[m.cursors[0]].ID)
			}
		case "L":
			if m.activeTab == 0 && listLen > 0 {
				m.state = stateLogs
				m.isProcessing = true
				m.detailTitle = "Logs: " + m.getFilteredContainers()[m.cursors[0]].Names[0]
				return m, tea.Batch(m.spinner.Tick, fetchLogsCmd(m.cli, m.getFilteredContainers()[m.cursors[0]].ID))
			}
		case "i":
			if m.activeTab == 0 && listLen > 0 {
				m.state = stateInspect
				m.isProcessing = true
				m.detailTitle = "Inspect: " + m.getFilteredContainers()[m.cursors[0]].Names[0]
				return m, tea.Batch(m.spinner.Tick, fetchInspectCmd(m.cli, m.getFilteredContainers()[m.cursors[0]].ID))
			}

		// Select One
		case " ":
			t, c := m.activeTab, m.cursors[m.activeTab]
			if listLen > 0 {
				if t == 0 {
					id := m.getFilteredContainers()[c].ID
					if _, ok := m.selectedC[id]; ok {
						delete(m.selectedC, id)
					} else {
						m.selectedC[id] = m.getFilteredContainers()[c]
					}
				} else if t == 1 {
					id := m.getFilteredImages()[c].ID
					if _, ok := m.selectedI[id]; ok {
						delete(m.selectedI, id)
					} else {
						m.selectedI[id] = m.getFilteredImages()[c]
					}
				} else if t == 2 {
					id := m.getFilteredVolumes()[c].Name
					if _, ok := m.selectedV[id]; ok {
						delete(m.selectedV, id)
					} else {
						m.selectedV[id] = m.getFilteredVolumes()[c]
					}
				} else if t == 3 {
					id := m.getFilteredNetworks()[c].ID
					if _, ok := m.selectedN[id]; ok {
						delete(m.selectedN, id)
					} else {
						m.selectedN[id] = m.getFilteredNetworks()[c]
					}
				}
			}

		// Select All
		case "a":
			t := m.activeTab
			if t == 0 {
				fc := m.getFilteredContainers()
				if len(m.selectedC) == len(fc) {
					m.selectedC = make(map[string]container.Summary)
				} else {
					for _, item := range fc {
						m.selectedC[item.ID] = item
					}
				}
			} else if t == 1 {
				fi := m.getFilteredImages()
				if len(m.selectedI) == len(fi) {
					m.selectedI = make(map[string]image.Summary)
				} else {
					for _, item := range fi {
						m.selectedI[item.ID] = item
					}
				}
			} else if t == 2 {
				fv := m.getFilteredVolumes()
				if len(m.selectedV) == len(fv) {
					m.selectedV = make(map[string]*volume.Volume)
				} else {
					for _, item := range fv {
						m.selectedV[item.Name] = item
					}
				}
			} else if t == 3 {
				fn := m.getFilteredNetworks()
				if len(m.selectedN) == len(fn) {
					m.selectedN = make(map[string]network.Summary)
				} else {
					for _, item := range fn {
						m.selectedN[item.ID] = item
					}
				}
			}

		// Prune
		case "p":
			if m.activeTab == 0 {
				for _, c := range m.getFilteredContainers() {
					if c.State == "exited" || c.State == "dead" {
						m.selectedC[c.ID] = c
					}
				}
			} else if m.activeTab == 1 {
				for _, img := range m.getFilteredImages() {
					if len(img.RepoTags) == 0 || img.RepoTags[0] == "<none>:<none>" {
						m.selectedI[img.ID] = img
					}
				}
			}

		case "enter":
			if len(m.selectedC) > 0 || len(m.selectedI) > 0 || len(m.selectedV) > 0 || len(m.selectedN) > 0 {
				m.confirming = true
			}
		}
	}
	return m, tea.Batch(cmds...)
}
