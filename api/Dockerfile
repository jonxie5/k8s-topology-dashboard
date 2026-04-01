# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/dashboard-api ./cmd/dashboard-api

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/dashboard-api /dashboard-api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/dashboard-api"]
