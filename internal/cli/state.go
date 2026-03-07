package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type ProjectState struct {
	Project    string                      `json:"project"`
	StartedAt time.Time                   `json:"started_at"`
	Containers map[string]ContainerState  `json:"containers"`
	NetworkID  string                      `json:"network_id"`
}

type ContainerState struct {
	ContainerID   string         `json:"container_id"`
	ContainerName string         `json:"container_name"`
	Image         string         `json:"image"`
	Ports         map[string]int `json:"ports"`
	Status        string         `json:"status"`
}

func stateDir(projectName string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dokrypt", "state", projectName)
}

func statePath(projectName string) string {
	return filepath.Join(stateDir(projectName), "state.json")
}

func saveState(state *ProjectState) error {
	dir := stateDir(state.Project)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(state.Project), data, 0644)
}

func loadState(projectName string) (*ProjectState, error) {
	data, err := os.ReadFile(statePath(projectName))
	if err != nil {
		return nil, err
	}
	var state ProjectState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func removeState(projectName string) error {
	return os.RemoveAll(stateDir(projectName))
}
