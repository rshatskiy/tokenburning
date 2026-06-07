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

## Фоновый сбор (опционально)

По умолчанию `tokenburning` ничего не делает в фоне. Включить периодический сбор (демон с автозапуском при логине):

```sh
tokenburning enable                 # локальный фоновый сбор, интервал 15 мин
tokenburning enable --interval-min 30
tokenburning disable                # выключить
```

Отправка агрегатов наверх — строго по согласию и только производных данных (без сырого, без проектов):

```sh
tokenburning push --breadth --dry-run          # посмотреть, что именно уйдёт
tokenburning enable --to https://server --breadth   # включить фоновую отправку breadth
```

- **macOS:** LaunchAgent (`~/Library/LaunchAgents/com.tokenburning.daemon.plist`)
- **Linux:** systemd user-unit (`~/.config/systemd/user/tokenburning.service`)
- **Windows:** Scheduled Task при логине

Всё без root. Лог демона: `~/.tokenburning/daemon.log` (macOS).

## Бинарь не подписан (пока)

Релизные бинари ещё не подписаны (notarization/Authenticode — позже). Если ОС предупреждает:

- **macOS (Gatekeeper):** при скачивании через браузер сними карантин —
  `xattr -d com.apple.quarantine /path/to/tokenburning`. Установка через `install.sh` (curl) карантин не ставит.
- **Windows (SmartScreen):** «Подробнее» → «Выполнить в любом случае».

## Приватность

Все данные обрабатываются локально. Дашборд работает на `127.0.0.1` с токеном; сетевых вызовов в дефолте нет.
