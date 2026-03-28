## Go version mismatch in Dockerfile
При `go get` зависимости могут поднять go version в go.mod (pgx v5.9 → go 1.25). Всегда проверяй что версия Go в Dockerfile совпадает с go.mod, иначе `go mod download` упадёт с `GOTOOLCHAIN=local`.

## .gitignore `server` матчит директории
Паттерн `server` без слеша в .gitignore матчит любой файл/папку с этим именем на любом уровне, включая `cmd/server/`. Для бинарника в корне использовать `/server` (с ведущим слешем). Всегда после первого коммита проверять `gh api repos/.../git/trees/main` что все файлы попали в репо.

## JWT secret whitespace causes 401
Railway env vars can have trailing newlines or spaces from copy-paste. Always `strings.TrimSpace()` the JWT secret before using it. Log `secret_len` on startup to catch this early.

## Goroutine with r.Context() cancels immediately after 202
When spawning a goroutine from an HTTP handler that returns early (202 Accepted), never pass `r.Context()` — it cancels when the response is sent. Use `context.Background()` for background work.

## Silent slot insertion errors leave plans empty
Never ignore DB insert errors in a loop. If CreateSlot fails (constraint violation, bad data, etc.), log the error AND track it. If ALL inserts fail, return an error — otherwise the plan exists with 0 slots and the frontend shows an empty calendar. Always log: AI response length, parsed slot count, inserted count, and the actual error.

## AI JSON parser breaks on Cyrillic mixed with JSON
Claude sometimes returns JSON with markdown fences or adds commentary around the JSON array. The `cleanJSONResponse` function must also handle extracting the JSON array/object from surrounding text, not just stripping code fences. Additionally, always try `extractJSON` as a fallback after initial parse failure.

## Create plan record before AI generation, not after
When generation is fire-and-forget (goroutine returns 202), create the DB plan record with status='generating' BEFORE starting AI calls. This lets the frontend poll plan status and show progress/errors. Update to 'draft' on success or 'failed' with error_message on failure.

## Supabase JWKS endpoint path
Supabase serves JWKS at `/auth/v1/.well-known/jwks.json`, NOT at `/.well-known/jwks.json`. The correct URL is `{SUPABASE_URL}/auth/v1/.well-known/jwks.json`. Discoverable via OpenID config at `/auth/v1/.well-known/openid-configuration`.
