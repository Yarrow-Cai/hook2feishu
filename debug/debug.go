package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var enabled bool
var logPath string

func init() {
	enabled = os.Getenv("HOOK2FEISHU_DEBUG") == "1" || os.Getenv("HOOK2FEISHU_DEBUG") == "true"
	home, err := os.UserHomeDir()
	if err == nil {
		dir := filepath.Join(home, ".config", "hook2feishu")
		os.MkdirAll(dir, 0700)
		logPath = filepath.Join(dir, "debug.log")
	}
}

func Log(format string, args ...interface{}) {
	if !enabled || logPath == "" {
		return
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] %s\n", ts, fmt.Sprintf(format, args...))
}
