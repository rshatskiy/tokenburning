package model

import "time"

type Tool string

const (
	ToolClaudeCode Tool = "claude_code"
	ToolCursor     Tool = "cursor"
	ToolCodex      Tool = "codex"
	ToolGemini     Tool = "gemini_cli"
	ToolCopilot    Tool = "copilot"
	ToolOpenCode   Tool = "opencode"
	ToolCline      Tool = "cline" // семейство: Cline / Roo Code / KiloCode (один формат)
)

type BillingMode string

const (
	BillingAPIUsage       BillingMode = "api_usage"       // реальные деньги
	BillingFlatEquivalent BillingMode = "flat_equivalent" // эквивалент по API-ставкам
)

type CostBasis string

const (
	BasisActual     CostBasis = "actual"
	BasisEquivalent CostBasis = "equivalent"
	BasisEstimated  CostBasis = "estimated"
)

// Tokens — нормализованная разбивка токенов. Поля nullable по смыслу:
// 0 означает "не отдано инструментом". Cache1h/Cache5m — кэш по горизонтам
// (Claude Code), Reasoning — рассуждающие токены (Codex).
type Tokens struct {
	Input     int64 `json:"input"`
	Output    int64 `json:"output"`
	CacheRead int64 `json:"cache_read"`
	Cache1h   int64 `json:"cache_1h"`
	Cache5m   int64 `json:"cache_5m"`
	Reasoning int64 `json:"reasoning"`
	Total     int64 `json:"total"`
}

// TotalOrSum возвращает Total, если источник его сообщил (ненулевой),
// иначе сумму известных компонентов. Сообщённый источником Total == 0
// трактуется как «не задан»; у реальных событий Total > 0.
func (t Tokens) TotalOrSum() int64 {
	if t.Total > 0 {
		return t.Total
	}
	return t.Input + t.Output + t.CacheRead + t.Cache1h + t.Cache5m + t.Reasoning
}

type Cost struct {
	Amount         float64   `json:"amount"`
	Currency       string    `json:"currency"`
	Basis          CostBasis `json:"basis"`
	PricingVersion string    `json:"pricing_version"`
}

// Event — нормализованное биллинговое событие из любого инструмента.
type Event struct {
	EventID     string      `json:"event_id"` // естественный ключ источника
	Tool        Tool        `json:"tool"`
	TS          time.Time   `json:"ts"`
	Model       string      `json:"model"` // "unknown" если неизвестна
	BillingMode BillingMode `json:"billing_mode"`
	Cost        Cost        `json:"cost"`
	Tokens      Tokens      `json:"tokens"`
	SessionID   string      `json:"session_id,omitempty"`
	ProjectKey  string      `json:"project_key,omitempty"` // хеш пути; локально
	ExtraRaw    []byte      `json:"-"`                     // сырьё для бэкфилла (исчезающие источники)
}

type Fidelity string

const (
	FidelityNone    Fidelity = "none"
	FidelityPartial Fidelity = "partial"
	FidelityFull    Fidelity = "full"
)

// Capabilities декларирует, какие данные адаптер реально отдаёт.
type Capabilities struct {
	HasTokens     Fidelity      `json:"has_tokens"`
	HasCache      bool          `json:"has_cache"`
	HasSessions   bool          `json:"has_sessions"`
	HasAcceptRate bool          `json:"has_accept_rate"`
	BillingModes  []BillingMode `json:"billing_modes"`
}
