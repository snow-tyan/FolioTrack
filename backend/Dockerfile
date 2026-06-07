# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git/certs
RUN apk add --no-cache git ca-certificates

# Copy dependency configs
COPY go.mod ./
# If go.sum exists copy it. We'll download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o foliotrack-backend .

# Run stage
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/foliotrack-backend .

# Expose port
EXPOSE 8080

# Run
CMD ["./foliotrack-backend"]
