package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// State 节点状态信息
type State struct {
	RegisterId string `json:"register_id"`
	NodeID     int    `json:"node_id"`
	Hostname   string `json:"hostname"`
}

const (
	DefaultDataDir = "/tmp/trojan-agent-node"
	StateFileName  = "state.json"
)

// GetStateFilePath 获取状态文件的完整路径
func GetStateFilePath(dataDir string) string {
	if dataDir == "" {
		dataDir = DefaultDataDir
	}
	return filepath.Join(dataDir, StateFileName)
}

// LoadState 加载状态文件
func LoadState(dataDir string) (*State, error) {
	path := GetStateFilePath(dataDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("state file not exist: %s", path)
			return nil, nil // 文件不存在返回 nil，不报错
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	log.Debugf("loaded state from %s: %+v", path, state)
	return &state, nil
}

// SaveState 保存状态到文件
func SaveState(dataDir string, state *State) error {
	// 确保目录存在
	if dataDir == "" {
		dataDir = DefaultDataDir
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	path := GetStateFilePath(dataDir)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// 先写入临时文件，再原子性替换
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	log.Infof("saved state to %s", path)
	return nil
}

// ClearState 清空状态文件
func ClearState(dataDir string) error {
	path := GetStateFilePath(dataDir)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}
	if err == nil {
		log.Infof("cleared state file: %s", path)
	}
	return nil
}
