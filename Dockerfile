# Build the application
FROM golang:1.22-alpine AS builder
LABEL authors="BitxBit"
LABEL org.opencontainers.image.source=https://github.com/JeanAEckelberg/BitBattleOrchestrationService
LABEL org.opencontainers.image.description="The Orchestration Service manages the docker contains for the BitBattle System."
LABEL org.opencontainers.image.licenses=MIT


WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main .


# Create the minimal image
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]