package daemon

import (
	"context"
	"fmt"
	"time"

	"github.com/rshatskiy/tokenburning/internal/aggregate"
	"github.com/rshatskiy/tokenburning/internal/collect"
	"github.com/rshatskiy/tokenburning/internal/config"
	"github.com/rshatskiy/tokenburning/internal/platform"
	"github.com/rshatskiy/tokenburning/internal/pricing"
	"github.com/rshatskiy/tokenburning/internal/selfupdate"
	"github.com/rshatskiy/tokenburning/internal/store"
)

type Options struct {
	DBPath         string
	Interval       time.Duration
	PushEnabled    bool
	PushCategories []string
	PushEndpoint   string
	PushToken      string
	Period         string // период для push-агрегата (например "30d")
	AutoUpdate     bool   // фоновое самообновление (раз в сутки)
	Version        string // текущая версия (для сравнения с релизом)
	Log            func(string)
}

func (o Options) log(msg string) {
	if o.Log != nil {
		o.Log(msg)
	}
}

// RunOnce выполняет один проход: собрать в БД и (если включено) отправить агрегат.
func RunOnce(o Options) (collect.Result, error) {
	_ = pricing.RefreshLive(5 * time.Second) // best-effort обновление цен (кэш сутки)
	cfg, _ := config.Load()
	cat, err := pricing.LoadEffective(cfg.ModelAliases)
	if err != nil {
		return collect.Result{}, err
	}
	paths, err := platform.Detect()
	if err != nil {
		return collect.Result{}, err
	}
	db, err := store.Open(o.DBPath)
	if err != nil {
		return collect.Result{}, err
	}
	defer db.Close()

	res, err := collect.Run(db, cat, paths, nil)
	if err != nil {
		return res, err
	}
	o.log(fmt.Sprintf("сбор: собрано=%d карантин=%d", res.Collected, res.Quarantined))

	if o.PushEnabled && o.PushEndpoint != "" && len(o.PushCategories) > 0 {
		period := o.Period
		if period == "" {
			period = "30d"
		}
		p, err := aggregate.Build(db, o.PushCategories, period)
		if err != nil {
			o.log("push: ошибка сборки агрегата: " + err.Error())
		} else if err := aggregate.Push(p, o.PushEndpoint, o.PushToken); err != nil {
			o.log("push: " + err.Error())
		} else {
			o.log(fmt.Sprintf("push: отправлено %v", o.PushCategories))
		}
	}
	return res, nil
}

// Run запускает периодический цикл до отмены ctx. Сразу делает первый проход.
func Run(ctx context.Context, o Options) error {
	if o.Interval <= 0 {
		o.Interval = 15 * time.Minute
	}
	if _, err := RunOnce(o); err != nil {
		o.log("сбор: ошибка: " + err.Error())
	}
	t := time.NewTicker(o.Interval)
	defer t.Stop()

	var updC <-chan time.Time
	if o.AutoUpdate {
		if o.maybeUpdate() {
			return nil
		}
		ut := time.NewTicker(24 * time.Hour)
		defer ut.Stop()
		updC = ut.C
	}

	for {
		select {
		case <-ctx.Done():
			o.log("демон остановлен")
			return nil
		case <-t.C:
			if _, err := RunOnce(o); err != nil {
				o.log("сбор: ошибка: " + err.Error())
			}
		case <-updC:
			if o.maybeUpdate() {
				return nil
			}
		}
	}
}

// maybeUpdate проверяет релизы и при наличии новее — качает, ставит и перезапускает
// процесс (на unix — exec в тот же PID; на Windows применится при следующем запуске).
// Возвращает true, если обновление применено (текущий процесс должен завершиться).
func (o Options) maybeUpdate() bool {
	tag, updated, err := selfupdate.CheckAndApply(o.Version)
	if err != nil {
		o.log("авто-апдейт: " + err.Error())
		return false
	}
	if !updated {
		return false
	}
	o.log("авто-апдейт: обновлено до " + tag + " — перезапуск")
	if err := restartSelf(); err != nil {
		o.log("авто-апдейт: перезапуск не удался: " + err.Error())
	}
	return true
}
