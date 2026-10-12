FROM golang:1.26.5-alpine AS builder

WORKDIR /build

COPY go.mod ./
RUN go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN go build -o main ./cmd/main.go

FROM alpine:3.19

RUN apk add --no-cache curl

WORKDIR /app

COPY --from=builder /build/main .

EXPOSE 8080

CMD ["/app/main"]