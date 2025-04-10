# Build stage
FROM golang:1.23 as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o pvz_service ./cmd/api

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/pvz_service .
EXPOSE 8080
EXPOSE 9000
EXPOSE 3000
CMD ["./pvz_service"]