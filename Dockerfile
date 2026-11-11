# Build Stage
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/main.go

# Final Production Stage
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/bin/server /app/server

EXPOSE 8080

CMD ["/app/server"]