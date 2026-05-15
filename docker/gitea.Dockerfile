FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git
WORKDIR /src
RUN git clone --depth 1 https://gitea.com/gitea/gitea-mcp.git .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /bin/server .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/server /server
EXPOSE 8000
ENTRYPOINT ["/server", "-t", "http", "--host", "0.0.0.0", "--port", "8000"]
