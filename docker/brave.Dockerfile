FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /bin/server ./cmd/brave

FROM alpine:3.21
COPY --from=builder /bin/server /server
EXPOSE 8000
ENTRYPOINT ["/server"]
