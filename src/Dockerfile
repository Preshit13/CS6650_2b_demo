FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o product-service main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/product-service .

EXPOSE 8080

CMD ["./product-service"]