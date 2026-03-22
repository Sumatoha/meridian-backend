## Go version mismatch in Dockerfile
При `go get` зависимости могут поднять go version в go.mod (pgx v5.9 → go 1.25). Всегда проверяй что версия Go в Dockerfile совпадает с go.mod, иначе `go mod download` упадёт с `GOTOOLCHAIN=local`.
