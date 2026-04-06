FROM golang:1.24-alpine AS builder
RUN go install github.com/okooo5km/memory-mcp-server-go@latest

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/memory-mcp-server-go /server
EXPOSE 8080
ENTRYPOINT ["/server", "-t", "http"]
