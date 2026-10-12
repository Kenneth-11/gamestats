# Simulador: se compila desde fuente (el .exe del repo es Windows y no corre en Linux)
FROM golang:1.26.5-alpine AS builder
WORKDIR /build
COPY go.mod ./
COPY src/ ./src/
RUN go build -o simulator ./src/simulator.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /build/simulator .
# Escribe data/*.jsonl relativo a /app; se monta ./data:/app/data
CMD ["./simulator", "42"]
