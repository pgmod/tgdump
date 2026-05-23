#  Используем официальный образ Golang
FROM golang:1.24.2-alpine as builder
ARG BUILDKIT_INLINE_CACHE=0

# Устанавливаем зависимости
WORKDIR /app
COPY go.mod  go.sum  ./
RUN go mod download

# Копируем исходники
COPY cmd cmd
COPY internal internal

# Сборка бинарника
RUN --mount=type=cache,target=/gocache \
GOCACHE=/gocache \
GOOS=linux GOARCH=amd64 \
go build -ldflags="-w -s" -o /telegrampgbackup ./cmd/main/

# Финальный образ
FROM debian:bookworm-slim

WORKDIR /app

# Устанавливаем официальный репозиторий PostgreSQL и клиент версии 16
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    ca-certificates \
    gnupg && \
    # Добавляем ключ и репозиторий
    curl -fsSL https://postgresql.org | gpg --dearmor -o /usr/share/keyrings/postgresql-archive-keyring.gpg && \
    echo "deb [signed-by=/usr/share/keyrings/postgresql-archive-keyring.gpg] http://postgresql.org bookworm-pgdg main" > /etc/apt/sources.list.d/pgdg.list && \
    # Обновляем списки и устанавливаем нужный клиент
    apt-get update && \
    apt-get install -y --no-install-recommends postgresql-client-16 && \
    # Удаляем утилиты сборки и очищаем кэш для уменьшения размера образа
    apt-get purge -y --auto-remove curl ca-certificates gnupg && \
    rm -rf /var/lib/apt/lists/*

# Копируем бинарник из стадии сборки
COPY --from=builder /telegrampgbackup ./telegrampgbackup

# Запуск
ENTRYPOINT ["./telegrampgbackup"]
