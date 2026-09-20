# Multi-stage build para a API Go e Frontend HTMX
FROM golang:alpine AS builder

WORKDIR /build

# Instala certificados SSL e ferramentas de compilação
RUN apk add --no-cache ca-certificates tzdata

COPY apps/api/go.mod apps/api/go.sum ./apps/api/
COPY apps/web/go.mod ./apps/web/

COPY apps/web/ ./apps/web/
COPY apps/api/ ./apps/api/

WORKDIR /build/apps/api
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /build/collector ./cmd/collector

# Imagem de execução mínima
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /build/api /app/api
COPY --from=builder /build/migrate /app/migrate
COPY --from=builder /build/collector /app/collector
COPY --from=builder /build/apps/api/migrations /app/migrations

USER appuser

EXPOSE 8080

CMD ["/app/api"]
