FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
RUN git clone --depth 1 https://github.com/okooo5km/memory-mcp-server-go.git .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /bin/server .

FROM alpine:3.21
COPY --from=builder /bin/server /server
EXPOSE 8080
ENTRYPOINT ["/server", "-t", "http"]
