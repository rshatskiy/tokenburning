package platform

import (
	"os"
	"path/filepath"
)

// Paths — корневые расположения логов инструментов на текущей ОС.
type Paths struct {
	ClaudeCodeProjects string // ~/.claude/projects
}

// Detect вычисляет пути для текущего пользователя.
// Учитывает override $CLAUDE_CONFIG_DIR для Claude Code.
func Detect() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	claudeRoot := filepath.Join(home, ".claude")
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
		claudeRoot = v
	}
	return Paths{
		ClaudeCodeProjects: filepath.Join(claudeRoot, "projects"),
	}, nil
}
