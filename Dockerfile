FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/tokeping ./cmd/tokeping

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/tokeping /app/

ENTRYPOINT ["/app/tokeping"]
