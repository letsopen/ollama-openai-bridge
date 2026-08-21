FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o ollama-bridge main.go

FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/ollama-bridge .

EXPOSE 11434

ENTRYPOINT ["./ollama-bridge"]
