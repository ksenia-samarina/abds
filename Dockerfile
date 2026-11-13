FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server ./cmd/app/main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/bin/server /app/server

EXPOSE 8080

CMD ["/app/server"]
