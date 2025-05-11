FROM golang:1.21-alpine AS builder

# Install required dependencies for ZeroMQ
RUN apk add --no-cache gcc g++ make pkgconfig zeromq-dev

WORKDIR /app
COPY . .

# Build with CGO enabled
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -o tokeping ./cmd/tokeping

FROM alpine:latest

# Install runtime dependencies for ZeroMQ
RUN apk add --no-cache libzmq

WORKDIR /app
COPY --from=builder /app/tokeping /app/
COPY web /app/web

ENTRYPOINT ["/app/tokeping"]
