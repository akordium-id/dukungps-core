FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates postgresql16-client
WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /app/migrations /migrations
COPY deployments/docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 5023 5027 5020 8080
ENTRYPOINT ["/entrypoint.sh"]
