# Docker Build

Jeder MCP-Server bekommt ein eigenes Docker Image.

## Standard-Pattern (eigener Code)

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /bin/server ./cmd/<name>

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /bin/server /server
EXPOSE 8000
ENTRYPOINT ["/server"]
```

## Externes Binary (memory)

```dockerfile
FROM golang:1.26-alpine AS builder
RUN go install github.com/okooo5km/memory-mcp-server-go@latest

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/memory-mcp-server-go /server
EXPOSE 8080
ENTRYPOINT ["/server", "-t", "http"]
```

## Regeln

- IMMER `CGO_ENABLED=0` fuer statisches Binary
- IMMER `-ldflags "-s -w"` fuer kleinere Binary (strip debug info)
- IMMER `scratch` als finale Base (kein Alpine, kein Distroless)
- CA-Zertifikate kopieren fuer HTTPS-Zugriff
- EXPOSE Port dokumentiert den Standard-Port
- Kein HEALTHCHECK im Dockerfile — wird in docker-compose.yml definiert
- Erwartete Image-Groesse: 10-15MB

## Build und Tag

```bash
make docker-brave    # Baut traffino/mcp-brave:latest
make docker-all      # Baut alle Images
```
