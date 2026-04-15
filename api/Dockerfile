FROM golang:1.24-alpine AS builder

WORKDIR /build

RUN apk add --no-cache gcc musl-dev

# Copy go.mod and go.sum first for better caching
COPY api/go.mod api/go.sum ./
RUN go mod download

# Copy rest of source code
COPY api/ ./

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -v -o main ./cmd

FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app

COPY --from=builder /build/main .

EXPOSE 8080

CMD ["./main"]
