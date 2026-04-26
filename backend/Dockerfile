# ---- Build stage ----
FROM golang:1.24-alpine AS builder

ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/harem-api ./cmd/api

# ---- Runtime stage ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/harem-api /app/harem-api
COPY migrations/ /app/migrations/

ENV PORT=40080
ENV DATABASE_URL=postgres://harem:harem@postgres:5432/harem?sslmode=disable
ENV REDIS_URL=redis://redis:6379/0

EXPOSE 40080

ENTRYPOINT ["/app/harem-api"]
CMD ["serve"]
