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
FROM postgres:18-alpine

WORKDIR /app

# Копируем бинарник из стадии сборки
COPY --from=builder /telegrampgbackup ./telegrampgbackup

# Запуск
ENTRYPOINT ["./telegrampgbackup"]
