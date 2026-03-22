## Go version mismatch in Dockerfile
При `go get` зависимости могут поднять go version в go.mod (pgx v5.9 → go 1.25). Всегда проверяй что версия Go в Dockerfile совпадает с go.mod, иначе `go mod download` упадёт с `GOTOOLCHAIN=local`.

## .gitignore `server` матчит директории
Паттерн `server` без слеша в .gitignore матчит любой файл/папку с этим именем на любом уровне, включая `cmd/server/`. Для бинарника в корне использовать `/server` (с ведущим слешем). Всегда после первого коммита проверять `gh api repos/.../git/trees/main` что все файлы попали в репо.

## JWT secret whitespace causes 401
Railway env vars can have trailing newlines or spaces from copy-paste. Always `strings.TrimSpace()` the JWT secret before using it. Log `secret_len` on startup to catch this early.
