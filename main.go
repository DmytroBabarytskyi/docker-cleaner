package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

type sessionState int

const (
	stateList sessionState = iota
	stateLogs
	stateInspect
)

// model зберігає стан всього додатка
type model struct {
	cli       *client.Client
	state     sessionState
	activeTab int
	width     int
	height    int

	containers []container.Summary
	images     []image.Summary
	volumes    []*volume.Volume
	networks   []network.Summary

	cursors [4]int
	offsets [4]int

	selectedC map[string]container.Summary
	selectedI map[string]image.Summary
	selectedV map[string]*volume.Volume
	selectedN map[string]network.Summary

	textInput textinput.Model
	spinner   spinner.Model
	viewport  viewport.Model

	isProcessing bool
	confirming   bool
	quitting     bool
	finalMsg     string
	detailTitle  string
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Println("Error connecting to Docker:", err)
		os.Exit(1)
	}
	defer cli.Close()

	ctx := context.Background()
	containers, _ := cli.ContainerList(ctx, container.ListOptions{All: true, Size: true})
	images, _ := cli.ImageList(ctx, image.ListOptions{All: true})
	volumeResp, _ := cli.VolumeList(ctx, volume.ListOptions{})
	networks, _ := cli.NetworkList(ctx, network.ListOptions{})

	sort.Slice(containers, func(i, j int) bool { return containers[i].SizeRw > containers[j].SizeRw })
	sort.Slice(images, func(i, j int) bool { return images[i].Size > images[j].Size })

	ti := textinput.New()
	ti.Placeholder = "Type to filter (name or compose project)..."
	ti.CharLimit = 50
	ti.Width = 40

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().Border(lipgloss.RoundedBorder(), true).BorderForeground(lipgloss.Color("#7D56F4")).Padding(1, 2)

	initialModel := model{
		cli:       cli,
		state:     stateList,
		textInput: ti,
		spinner:   sp,
		viewport:  vp,

		containers: containers, selectedC: make(map[string]container.Summary),
		images: images, selectedI: make(map[string]image.Summary),
		volumes: volumeResp.Volumes, selectedV: make(map[string]*volume.Volume),
		networks: networks, selectedN: make(map[string]network.Summary),
	}

	if _, err := tea.NewProgram(initialModel, tea.WithAltScreen()).Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
