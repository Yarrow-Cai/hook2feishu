package checkpoint

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Yarrow-Cai/hook2feishu/config"
)

// Checkpoint stores per-session stats for incremental tracking.
type Checkpoint struct {
	OutputTokens int     `json:"output_tokens"`
	ToolCalls    int     `json:"tool_calls"`
	Turns        int     `json:"turns"`
	Agents       int     `json:"agents"`
	Time         float64 `json:"time"`
}

func dir() (string, error) {
	d, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "checkpoints"), nil
}

func pathFor(sessionID string) (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	safe := strings.ReplaceAll(sessionID, "/", "_")
	if len(safe) > 64 {
		safe = safe[:64]
	}
	if safe == "" {
		safe = "default"
	}
	return filepath.Join(d, safe+".json"), nil
}

func Load(sessionID string) *Checkpoint {
	path, err := pathFor(sessionID)
	if err != nil {
		return &Checkpoint{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return &Checkpoint{}
	}
	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return &Checkpoint{}
	}
	return &cp
}

func Save(sessionID string, cp *Checkpoint) error {
	path, err := pathFor(sessionID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(cp)
	if err != nil {
		return err
	}
	Cleanup()
	return os.WriteFile(path, data, 0600)
}

func Cleanup() {
	d, err := dir()
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	entries, err := os.ReadDir(d)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(d, entry.Name()))
		}
	}
}
