package platform

import (
	"os"
	"path/filepath"
	"runtime"
)

// Paths — корневые расположения логов инструментов на текущей ОС.
type Paths struct {
	ClaudeCodeProjects string   // ~/.claude/projects
	CodexSessions      string   // ~/.codex/sessions (override $CODEX_HOME)
	CursorStorage      string   // корень Cursor (содержит User/globalStorage, User/workspaceStorage)
	GeminiTmp          string   // ~/.gemini/tmp/<project>/chats/*.json (override $GEMINI_HOME → <dir>/tmp)
	CopilotSessions    string   // ~/.copilot/session-state (CLI-сессии Copilot)
	VSCodeStorageDirs  []string // User-каталоги VS Code/VSCodium (workspaceStorage/globalStorage внутри)
	OpenCodeData       string   // $XDG_DATA_HOME|~/.local/share + /opencode
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

	geminiHome := filepath.Join(home, ".gemini")
	if v := os.Getenv("GEMINI_HOME"); v != "" {
		geminiHome = v
	}

	return Paths{
		ClaudeCodeProjects: filepath.Join(claudeRoot, "projects"),
		CodexSessions:      filepath.Join(codexHome, "sessions"),
		CursorStorage:      cursorStorage(home),
		GeminiTmp:          filepath.Join(geminiHome, "tmp"),
		CopilotSessions:    filepath.Join(home, ".copilot", "session-state"),
		VSCodeStorageDirs:  vscodeUserDirs(home),
		OpenCodeData:       openCodeData(home),
	}, nil
}

// vscodeUserDirs — каталоги User редакторов VS Code-семейства на текущей ОС
// (внутри лежат workspaceStorage и globalStorage с данными расширений).
func vscodeUserDirs(home string) []string {
	var bases []string
	switch runtime.GOOS {
	case "darwin":
		bases = []string{filepath.Join(home, "Library", "Application Support")}
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			bases = []string{appData}
		} else {
			bases = []string{filepath.Join(home, "AppData", "Roaming")}
		}
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			bases = []string{xdg}
		} else {
			bases = []string{filepath.Join(home, ".config")}
		}
	}
	var out []string
	for _, b := range bases {
		for _, ed := range []string{"Code", "Code - Insiders", "VSCodium"} {
			out = append(out, filepath.Join(b, ed, "User"))
		}
	}
	return out
}

// openCodeData — каталог данных OpenCode ($XDG_DATA_HOME или ~/.local/share).
func openCodeData(home string) string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "opencode")
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
