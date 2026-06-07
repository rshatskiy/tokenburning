# tokenburning

Единый локальный дашборд по всем твоим ИИ-инструментам (Claude Code, Codex, Cursor) — стоимость, токены, активность и аналитика сессий. Один бинарь, ставится за секунды, ничего не уходит в сеть.

## Установка

**macOS / Linux:**
```sh
curl -fsSL https://raw.githubusercontent.com/rshatskiy/tokenburning/main/install.sh | sh
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/rshatskiy/tokenburning/main/install.ps1 | iex
```

Или скачай бинарь под свою ОС со страницы [Releases](https://github.com/rshatskiy/tokenburning/releases).

## Использование

```sh
tokenburning scan        # разобрать локальные логи и показать стоимость
tokenburning dashboard   # открыть локальный web-дашборд
tokenburning version
```

## Бинарь не подписан (пока)

Релизные бинари ещё не подписаны (notarization/Authenticode — позже). Если ОС предупреждает:

- **macOS (Gatekeeper):** при скачивании через браузер сними карантин —
  `xattr -d com.apple.quarantine /path/to/tokenburning`. Установка через `install.sh` (curl) карантин не ставит.
- **Windows (SmartScreen):** «Подробнее» → «Выполнить в любом случае».

## Приватность

Все данные обрабатываются локально. Дашборд работает на `127.0.0.1` с токеном; сетевых вызовов в дефолте нет.
