package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Messages for Bubble Tea
type stateChangeMsg struct{ id, state string }
type deleteCompleteMsg struct{ msg string }
type logsMsg string
type inspectMsg string

func stopContainerCmd(cli *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		_ = cli.ContainerStop(context.Background(), id, container.StopOptions{})
		return stateChangeMsg{id: id, state: "exited"}
	}
}

func restartContainerCmd(cli *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		_ = cli.ContainerRestart(context.Background(), id, container.StopOptions{})
		return stateChangeMsg{id: id, state: "running"}
	}
}

func fetchLogsCmd(cli *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		options := container.LogsOptions{ShowStdout: true, ShowStderr: true, Tail: "100"}
		out, err := cli.ContainerLogs(ctx, id, options)
		if err != nil {
			return logsMsg(fmt.Sprintf("Error fetching logs: %v", err))
		}
		defer out.Close()

		var stdout, stderr bytes.Buffer
		_, err = stdcopy.StdCopy(&stdout, &stderr, out)
		if err != nil {
			return logsMsg(fmt.Sprintf("Error reading logs: %v", err))
		}

		result := stdout.String() + "\n" + stderr.String()
		if result == "\n" || result == "" {
			return logsMsg("Logs are empty.")
		}
		return logsMsg(result)
	}
}

func fetchInspectCmd(cli *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		data, err := cli.ContainerInspect(ctx, id)
		if err != nil {
			return inspectMsg(fmt.Sprintf("Error inspecting: %v", err))
		}

		prettyJSON, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return inspectMsg("Error formatting JSON")
		}
		return inspectMsg(string(prettyJSON))
	}
}

func (m model) runAsyncDeletion() tea.Cmd {
	return func() tea.Msg {
		deletedCount := 0
		var freedSpace int64
		ctx := context.Background()

		for id, c := range m.selectedC {
			if err := m.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true}); err == nil {
				deletedCount++
				freedSpace += c.SizeRw
			}
		}
		for id, img := range m.selectedI {
			if _, err := m.cli.ImageRemove(ctx, id, image.RemoveOptions{Force: true, PruneChildren: true}); err == nil {
				deletedCount++
				freedSpace += img.Size
			}
		}
		for name := range m.selectedV {
			if err := m.cli.VolumeRemove(ctx, name, true); err == nil {
				deletedCount++
			}
		}
		for id := range m.selectedN {
			if err := m.cli.NetworkRemove(ctx, id); err == nil {
				deletedCount++
			}
		}

		result := fmt.Sprintf("✅ Successfully deleted %d items! Freed: %s", deletedCount, formatBytes(freedSpace))
		if deletedCount == 0 {
			result = "🚫 Error occurred or items were in use."
		}
		return deleteCompleteMsg{msg: result}
	}
}
