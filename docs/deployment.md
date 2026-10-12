# GameStats - Deployment

Entorno de deployment definido por el equipo: **Docker Compose local**
(`docker-compose.yml`). Los tres componentes corren como servicios
integrados en una red privada con healthchecks y arranque ordenado.

## Servicios

| Servicio | Imagen | Puerto | Salud |
|----------|--------|--------|-------|
| `gamestats-scala` | build `./scala-service` (sbt assembly → `app.jar`) | 8081 | `GET /health` |
| `gamestats-go` | build `./go-service` (binario estático) | 8080 | `GET /health`, espera a Scala (`service_healthy`) |
| `simulador` | build `./simulator` (one-shot, semilla `${SIMULATOR_SEED:-42}`) | — | genera `data/*.jsonl` vía volumen `./data:/app/data` |

## Despliegue local

```bash
cp .env.example .env   # opcional: ajustar puertos y semilla
docker compose up --build -d
curl http://localhost:8080/health
curl http://localhost:8081/health
```

## Carga de datos y verificación

```bash
# 7 partidas, 110 eventos (incluye casos guionados de patrones)
curl -X POST http://localhost:8080/ingest/file \
  -H "Content-Type: text/plain" --data-binary @data/carreras_all.jsonl
curl -X POST http://localhost:8080/ingest/file \
  -H "Content-Type: text/plain" --data-binary @data/combate_all.jsonl

# Patrones demostrables:
curl http://localhost:8080/matches/match_racing_900/analysis  # remontada player_hero 5->1
curl http://localhost:8080/matches/match_combat_900/analysis  # racha de 3 player_hero
curl http://localhost:8080/stats
```

## Publicación de imágenes (registry)

```bash
docker compose build
docker tag gamestats-gamestats-go <registry>/gamestats-go:1.0.0
docker tag gamestats-gamestats-scala <registry>/gamestats-scala:1.0.0
docker push <registry>/gamestats-go:1.0.0
docker push <registry>/gamestats-scala:1.0.0
```

## Pipeline CI/CD (`.github/workflows/ci.yml`)

1. `build-test-go`: `go mod download`, `go vet`, `go build`, `go test`, `go test -race`.
2. `build-test-scala`: `sbt compile`, `sbt test`, `sbt assembly`.
3. `docker-validate`: `docker compose config` + `build` de las 3 imágenes.
4. `documentation`: verifica que existan docs, compose, env example y README.

## Limitaciones conocidas del deployment

1. El estado de partidas vive en memoria de Go: se pierde al reiniciar el contenedor.
2. Sin persistencia externa ni réplicas: un solo contenedor por servicio.
3. El simulador es one-shot: genera datos y termina (exit 0, comportamiento esperado).
