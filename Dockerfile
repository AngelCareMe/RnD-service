FROM golang:1.24.1-alpine AS builder

RUN apk add --no-cache git

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api/main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates

RUN adduser -D -s /bin/sh appuser

WORKDIR /app

COPY config/ ./config/

COPY --from=builder /app/main .

RUN chown -R appuser:appuser ./

USER appuser

EXPOSE 8080

CMD ["./main"]