# GameStats - Resumen del Proyecto

## Estado Actual

Proyecto GameStats (curso Lenguajes de Programación, TEC). Núcleo completo
y verificado: Go compila y sirve, Scala compila con 10/10 tests, E2E
Go→Scala probado en local y en Docker. Historial Git con 3 commits.

### Fase 1: Análisis y Diseño ✅
- Esquema JSON de eventos en `docs/event-schema.md`
- Arquitectura Go/Scala en `docs/architecture.md`
- Contratos HTTP en `docs/api.md`
- Justificación de juegos/eventos/división en `docs/decisions.md`
- Deployment documentado en `docs/deployment.md`
- Juegos: carreras (`racing`) y combate (`combat`)

### Fase 2: Núcleo Go ✅ (verificado: `go build`, `go vet`, `go test`)
- Modelos: `Event`, `MatchState`, `PlayerMatchState`, `PlayerStats`, `MatchAnalysis`, `PatternResult`
- Validador común + específico por juego; JSON malformado y campos
  obligatorios rechazados (422)
- Ingestión JSON / objeto / JSONL y multipart (`file`)
- Estado temporal por partida con detección de duplicados (global + por partida)
- Concurrencia: workers con goroutines+channels (`WORKER_COUNT`, default 4),
  mutexes para mapas compartidos
- Endpoints: `/health`, `/events`, `/ingest/file`, `/matches/{id}`,
  `/matches/{id}/analysis`, `/players/{id}/stats`, `/stats`
  (`processed`, `rejectedEvents` incluidos)
- Tests Go: 14 casos (`domain`, `validation`, `ingestion`). `-race` corre en CI
  (requiere gcc, no disponible en Windows local)

### Fase 3: Núcleo Scala ✅ (verificado: `sbt compile`, `sbt test` 10/10)
- Solo stdlib + JDK HttpServer (sin frameworks): `GET /health`, `POST /analyze`
- Parser JSON propio; `Event`/`MatchRequest` con validación
- Funcionales: `filter`, `map`, `flatMap`, `groupBy`, `foldLeft`,
  `reduceOption`, `collect` + composición
- Racing: vueltas, mejor/promedio, penalizaciones, evolución, `byPlayer`
- Combate: ataques, daño total/promedio, eliminaciones, top, por arma,
  daño por eliminación (combina tipos), `byPlayer` con K/D
- Patrones: remontada (5→1 con evidencia) y racha (≥3 sin morir en medio)
- Juego desconocido: acepta y responde nota (extensibilidad)

### Fase 4: Integración Go-Scala ✅ (verificada en local y en Docker)
- `POST /analyze` con `{matchId, gameId, events}`; reintentos con backoff
- Go guarda `statistics` + `patterns` (parseo genérico) y expone `/analysis`
- Sin Scala: la partida queda `pending_analysis`, no se marca analizada

### Fase 5: Simulador y datos ✅
- `simulator/` con `go.mod` propio; salida determinista con semilla
  (claves ordenadas; hash idéntico entre corridas con seed 42)
- Genera 3 carreras + 2 combates + 2 casos guionados que garantizan patrones:
  `match_racing_900` (remontada) y `match_combat_900` (racha de 3)
- `data/carreras_all.jsonl` (4 partidas) y `data/combate_all.jsonl`
  (3 partidas): 7 partidas, 110 eventos. `*_small.jsonl`: una partida c/u.
- Casos negativos: `LAP_COMPLETED` inválidos (rechazados con 422)

### Fase 6: API y consultas ✅
- Verificado E2E: ingesta → `analyzed` con stats y patrones de Scala

### Fase 7: Docker ✅ (verificado: build + up + E2E + down)
- `go-service/Dockerfile` (multietapa, +curl para healthcheck)
- `scala-service/Dockerfile` (temurin JDK + sbt por TGZ, assembly → `app.jar`, +curl)
- `simulator/Dockerfile` (compila desde fuente; el `.exe` Windows no corre en Linux)
- `docker-compose.yml`: Scala (healthy primero), Go (`depends_on`), simulador
  one-shot con volumen `./data:/app/data`

### Fase 8: CI/CD ✅ (configurado, pendiente push para ejecución en GitHub)
- `ci.yml`: Go (download/vet/build/test/race) → Scala (compile/test/assembly) →
  Docker (`config` + `build`) → verificación de docs
- Sin job de push a registry (deployment definido: Compose local)

### Fase 9: Documentación ✅
- `README.md`, `docs/event-schema.md`, `docs/architecture.md`, `docs/api.md`,
  `docs/decisions.md`, `docs/deployment.md`, `.env.example`

## Resultados E2E de referencia (dataset actual, seed 42)

- `POST /ingest/file` carreras_all → `processed:72, rejected:2`
- `POST /ingest/file` combate_all → `processed:38, rejected:0`
- `GET /stats` → 7 partidas, 110 eventos, 10 jugadores
- `match_racing_900/analysis` → remontada `player_hero` 5→1
- `match_combat_900/analysis` → racha `player_hero` ×3, K/D 3

## Comandos

```bash
cd gamestats/go-service && go test ./...
cd gamestats/scala-service && sbt test
cd gamestats/simulator && go run ./src/simulator.go 42
docker compose up --build -d   # desde gamestats/
```

## Limitaciones conocidas

1. Estado en memoria: se pierde al reiniciar Go.
2. Sin persistencia externa, réplicas ni registry push.
3. `go test -race` requiere gcc (solo CI).
4. Pendiente: `git remote + push` para historial compartido y CI en GitHub.
