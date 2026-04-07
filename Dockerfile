# Build
FROM golang:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api

# Final
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y chromium ca-certificates fonts-liberation --no-install-recommends && rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /app/api .
EXPOSE 8080
CMD ["./api"]
