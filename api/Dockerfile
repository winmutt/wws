FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go.mod and go.sum first for better caching
COPY api/go.mod api/go.sum ./
RUN go mod download

# Copy rest of source code
COPY api/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -v -a -installsuffix cgo -o main ./cmd

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/main .

EXPOSE 8080

CMD ["./main"]
