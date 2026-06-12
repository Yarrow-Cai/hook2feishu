package gitcmd

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// Info holds git repository information.
type Info struct {
	Branch     string
	LastCommit string
	Dirty      bool
}

// GetInfo queries git for branch, last commit, and dirty status.
// Returns empty Info (not error) when cwd is empty or not a git repo.
func GetInfo(cwd string) *Info {
	info := &Info{}
	if cwd == "" {
		return info
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// Branch
	if out, err := run(ctx, cwd, "branch", "--show-current"); err == nil {
		info.Branch = strings.TrimSpace(out)
	}

	// Last commit (short hash + subject, max 60 chars)
	if out, err := run(ctx, cwd, "log", "--oneline", "-1", "--format=%h %s"); err == nil {
		s := strings.TrimSpace(out)
		if len(s) > 60 {
			s = s[:60]
		}
		info.LastCommit = s
	}

	// Dirty working tree
	if out, err := run(ctx, cwd, "status", "--porcelain"); err == nil {
		info.Dirty = strings.TrimSpace(out) != ""
	}

	return info
}

func run(ctx context.Context, cwd string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", cwd}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
