package pricing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Курсы ЦБ РФ (официальный XML, cp1251 — поэтому парсим регуляркой по байтам:
// нужные поля ASCII). Хранятся стоимости всегда в USD; конвертация — только
// при отображении.
const cbrURL = "https://www.cbr.ru/scripts/XML_daily.asp"

type fxCache struct {
	Fetched time.Time `json:"fetched"`
	USDRUB  float64   `json:"usdrub"`
	EURRUB  float64   `json:"eurrub"`
}

func fxCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tokenburning", "fx.json"), nil
}

var valuteRe = regexp.MustCompile(`(?s)<CharCode>(USD|EUR)</CharCode>.*?<VunitRate>([\d,\.]+)</VunitRate>`)

// FXRate возвращает множитель USD→code (USD=1). Курс ЦБ кэшируется сутки;
// офлайн — берём любой имеющийся кэш.
func FXRate(code string) (float64, error) {
	code = strings.ToUpper(code)
	if code == "" || code == "USD" {
		return 1, nil
	}
	if code != "RUB" && code != "EUR" {
		return 0, fmt.Errorf("валюта %q не поддерживается (USD, RUB, EUR)", code)
	}
	p, err := fxCachePath()
	if err != nil {
		return 0, err
	}
	var c fxCache
	if b, rerr := os.ReadFile(p); rerr == nil {
		_ = json.Unmarshal(b, &c)
	}
	if time.Since(c.Fetched) > 24*time.Hour && os.Getenv("TOKENBURNING_OFFLINE") == "" {
		if fresh, ferr := fetchCBR(); ferr == nil {
			c = fresh
			if b, merr := json.Marshal(c); merr == nil {
				_ = os.MkdirAll(filepath.Dir(p), 0o755)
				tmp := p + ".tmp"
				if os.WriteFile(tmp, b, 0o644) == nil {
					_ = os.Rename(tmp, p)
				}
			}
		}
	}
	if c.USDRUB <= 0 {
		return 0, fmt.Errorf("курс ЦБ недоступен (офлайн и нет кэша) — попробуйте позже")
	}
	switch code {
	case "RUB":
		return c.USDRUB, nil
	default: // EUR: USD→RUB→EUR кросс-курсом
		if c.EURRUB <= 0 {
			return 0, fmt.Errorf("курс EUR недоступен")
		}
		return c.USDRUB / c.EURRUB, nil
	}
}

func fetchCBR() (fxCache, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest("GET", cbrURL, nil)
	if err != nil {
		return fxCache{}, err
	}
	// ЦБ молча вешает соединения с дефолтным Go/curl User-Agent
	req.Header.Set("User-Agent", "tokenburning/1.0 (+https://tokenburning.ru)")
	resp, err := client.Do(req)
	if err != nil {
		return fxCache{}, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fxCache{}, err
	}
	c := fxCache{Fetched: time.Now()}
	for _, m := range valuteRe.FindAllSubmatch(b, -1) {
		v, perr := strconv.ParseFloat(strings.ReplaceAll(string(m[2]), ",", "."), 64)
		if perr != nil {
			continue
		}
		switch string(m[1]) {
		case "USD":
			c.USDRUB = v
		case "EUR":
			c.EURRUB = v
		}
	}
	if c.USDRUB <= 0 {
		return fxCache{}, fmt.Errorf("cbr: USD не найден в ответе")
	}
	return c, nil
}

// CurrencySymbol — символ для отображения.
func CurrencySymbol(code string) string {
	switch strings.ToUpper(code) {
	case "RUB":
		return "₽"
	case "EUR":
		return "€"
	default:
		return "$"
	}
}
