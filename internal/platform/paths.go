package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

// Paths — корневые расположения логов инструментов на текущей ОС.
type Paths struct {
	ClaudeCodeProjects string // ~/.claude/projects
	CodexSessions      string // ~/.codex/sessions (override $CODEX_HOME)
	CursorStorage      string // корень Cursor (содержит User/globalStorage, User/workspaceStorage)
}

// Detect вычисляет пути для текущего пользователя.
func Detect() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}

	claudeRoot := filepath.Join(home, ".claude")
	if v := os.Getenv("CLAUDE_CONFIG_DIR"); v != "" {
		claudeRoot = v
	}

	codexHome := filepath.Join(home, ".codex")
	if v := os.Getenv("CODEX_HOME"); v != "" {
		codexHome = v
	}

	return Paths{
		ClaudeCodeProjects: filepath.Join(claudeRoot, "projects"),
		CodexSessions:      filepath.Join(codexHome, "sessions"),
		CursorStorage:      cursorStorage(home),
	}, nil
}

// cursorStorage возвращает корень данных Cursor для текущей ОС.
func cursorStorage(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Cursor")
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Cursor")
		}
		return filepath.Join(home, "AppData", "Roaming", "Cursor")
	default: // linux и прочие unix
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "Cursor")
		}
		return filepath.Join(home, ".config", "Cursor")
	}
}
