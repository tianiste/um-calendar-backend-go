FROM golang:1.25.5-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/um-calendar-api ./api/cmd/main.go

FROM alpine:3.22

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/um-calendar-api /app/um-calendar-api
COPY --from=builder /src/db/migrations /app/db/migrations

ENV MIGRATIONS_PATH=/app/db/migrations
ENV GIN_MODE=release
EXPOSE 8080

USER app

CMD ["/app/um-calendar-api"]