# Деплой Advisor (Docker + GitHub Actions + SSH)

Один Docker-образ (Go-бинарь отдаёт API и SPA) за **Caddy** с авто-TLS. CI на
GitHub Actions: тесты → сборка образа → публикация в **GHCR** → деплой на VPS по SSH.
Постоянную работу обеспечивает Docker `restart: always` + systemd самого Docker.

```
push в main ──▶ [test: go vet+test] ──▶ [build-push: образ → ghcr.io] ──▶ [deploy: scp+ssh → docker compose up]
                                                                              VPS: Caddy(443) → advisor(8080)
```

## Что на VPS
- Ubuntu + **Docker Engine** и **docker compose plugin** (`docker compose version`).
- Папка `/app`, доступная на запись SSH-пользователю (`sudo mkdir -p /app && sudo chown $USER /app`).
- Открыты порты **80** и **443** (Caddy получает и продлевает сертификат Let's Encrypt).
- **DNS**: A-запись `твой-домен` → IP VPS (обязательно до первого деплоя, иначе TLS не выпустится).

## Настройка GitHub
1. Создать репозиторий и запушить проект (сейчас это ещё не git-репозиторий):
   ```
   git init && git add . && git commit -m "advisor"
   git branch -M main
   git remote add origin git@github.com:USER/advisor.git
   git push -u origin main
   ```
2. **Settings → Secrets and variables → Actions**:
   - Secrets:
     - `HOST` — IP/домен VPS
     - `USERNAME` — SSH-пользователь
     - `SSH_KEY` — приватный SSH-ключ (весь файл, с BEGIN/END)
     - `ADVISOR_TOKEN` — токен доступа к приложению (придумай длинный)
     - `GHCR_PAT` — *опционально*, только если образ приватный (PAT с `read:packages`)
   - Variables:
     - `ADVISOR_DOMAIN` — домен приложения (например `advisor.example.com`)
3. **GHCR-образ**: после первой успешной сборки образ появится в
   *Packages* репозитория. Проще всего сделать его **публичным**
   (Package → Package settings → Change visibility → Public) — тогда VPS тянет без логина.
   Иначе задай секрет `GHCR_PAT`, и деплой залогинится сам.

## Первый деплой
Пуш в `main` (или вручную *Actions → Build and Deploy → Run workflow*). Пайплайн
прогонит тесты, соберёт образ и развернёт стек. Через ~минуту:

- Открой **https://твой-домен** — введи `ADVISOR_TOKEN`.
- Android-форвардер: в приложении укажи `https://твой-домен` и тот же токен.

Образ пинуется по коммиту (`ADVISOR_IMAGE=...:<sha>` в `/app/.env`) — деплой детерминированный.

## Бэкапы
`deploy/backup.sh` делает консистентный дамп SQLite (через `.backup`, учитывает WAL),
жмёт и хранит последние 14. Добавь в cron на VPS:
```
crontab -e
30 3 * * * /app/backup.sh >> /app/backups/backup.log 2>&1
```
Восстановление: распакуй нужный `.db.gz`, останови стек, положи файл в том
`advisor-data` (`docker run --rm -v advisor-data:/data -v $PWD:/b alpine cp /b/advisor-XXXX.db /data/server.db`), подними стек.

## Откат
В `/app/.env` поменяй `ADVISOR_IMAGE` на прежний `...:<sha>` и `docker compose up -d`.
Список тегов — в GHCR-пакете.

## Локальная проверка образа (без TLS/Caddy)
```
docker build -t advisor .
docker run --rm -p 8080:8080 -e ADVISOR_TOKEN=dev advisor
# → http://localhost:8080, токен dev
```

## Почему один контейнер (а не как в aicity)
Advisor — один Go-бинарь, обслуживающий и API, и статику SPA (`ADVISOR_WEB`).
Поэтому вместо множества сервисов — один образ + Caddy. Меньше движущихся частей,
проще деплой и откат. БД — SQLite на именованном томе (миграции применяются
автоматически при старте).
