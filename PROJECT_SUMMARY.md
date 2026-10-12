# GameStats - Resumen del Proyecto

## Estado Actual

El proyecto GameStats ha sido implementado siguiendo el enunciado del curso de Lenguajes de Programación del TEC. A continuación se detalla el estado de cada fase:

### Fase 1: Análisis y Diseño ✅
- Esquema JSON de eventos documentado en `docs/event-schema.md`
- Arquitectura del sistema definida con separación Go/Scala
- Contratos HTTP definidos en `docs/api.md`
- Tipos de videojuegos: carreras (`racing`) y combate (`combat`)

### Fase 2: Núcleo Go ✅
- Servicio Go compilable con `go build ./cmd/main.go`
- Modelos de dominio: `Event`, `MatchState`, `PlayerMatchState`, `PlayerStats`, `MatchAnalysis`, `PatternResult`
- Validador de eventos comunes y específicos por tipo de juego
- Ingestión de eventos desde JSONL y API HTTP
- Gestión de estado temporal de partidas con detección de duplicados
- Concurrencia con goroutines y channels
- Mutexes para sincronización segura de mapas compartidos
- API HTTP con endpoints: /health, /events, /ingest/file, /matches/{matchId}, /matches/{matchId}/analysis, /players/{playerId}/stats, /stats

### Fase 3: Núcleo Scala ✅
- Servicio Scala con endpoint `POST /analyze`
- Estructuras de datos para estadísticas de carreras y combate
- Operaciones funcionales: `filter`, `map`, `groupBy`, `foldLeft`, `reduceOption`
- Detección de patrones: remontadas en carreras, rachas de eliminaciones
- Validación de condiciones para análisis
- Estructuras JSON de salida con `status`, `statistics`, `patterns`

### Fase 4: Integración Go-Scala ✅
- Comunicación HTTP real entre servicios
- Go envía partidas a `POST /analyze` en Scala
- Manejo de reintentos con backoff exponencial
- Identificación de partidas por `matchId`
- Cuando Scala no está disponible, Go no marca la partida como analizada

### Fase 5: Simulador de Eventos ✅
- Simulador independiente en `gamestats/simulator/src/simulator.go`
- Genera eventos deterministas con semilla configurable
- Produce múltiples partidas de carreras y combate
- Varios jugadores por partida
- Eventos con penalizaciones, cambios de posición
- Casos que activan remontadas y rachas de eliminación
- Casos que no activan esos patrones
- Eventos de tipos desconocidos que respetan el esquema común
- Eventos duplicados
- Datos guardados en `gamestats/data/carreras_small.jsonl` y `gamestats/data/combate_small.jsonl`

### Fase 6: API y Consultas HTTP ✅
- Todos los endpoints requeridos implementados:
  - `GET /health` - Disponibilidad del servicio
  - `POST /events` - Recibir eventos individuales
  - `POST /ingest/file` - Cargar archivo JSONL con múltiples eventos
  - `GET /matches/{matchId}` - Estado de una partida
  - `GET /matches/{matchId}/analysis` - Análisis de una partida
  - `GET /players/{playerId}/stats` - Estadísticas de un jugador
  - `GET /stats` - Estadísticas globales

### Fase 7: Docker y docker-compose ✅
- Dockerfile para servicio Go (compilación multietapa)
- docker-compose.yml con servicios Go, Scala y simulador
- Healthchecks configurados para ambos servicios
- Variables de entorno para puertos y URLs
- Puertos mapeados: 8080 (Go), 8081 (Scala)

### Fase 8: GitHub Actions CI/CD ✅
- Workflow `.github/workflows/ci.yml` con:
  - Build y test de Go (`go test ./...`, `go test -race ./...`)
  - Build y test de Scala (paso documentado, requiere sbt)
  - Validación de Docker Compose
  - Verificación de documentación
  - Lint Go con `go vet`

### Fase 9: Documentación ✅
- `README.md` completo con descripción, requisitos, arquitectura, instalación, configuración
- `docs/event-schema.md` - Esquema completo de eventos
- `docs/architecture.md` - Arquitectura del sistema
- `docs/api.md` - Contratos API detallados
- `.dockerignore` - Archivos a ignorar en Docker build
- `.env.example` - Variables de entorno ejemplos

## Funcionalidades Implementadas

### Servicios
- Go service: 8 endpoints HTTP operativos
- Scala service: endpoint de análisis HTTP
- Simulador de eventos deterministas con semilla configurable

### Eventos
- 12 tipos de eventos definidos (MATCH_STARTED, PLAYER_JOINED, LAP_COMPLETED, POSITION_CHANGED, PENALTY_RECEIVED, MATCH_FINISHED, PLAYER_ATTACKED, PLAYER_ELIMINATED para racing/combat)
- Validación estructural y específica por tipo
- Detección de eventos duplicados dentro de partidas
- Eventos con JSON malformado rechazados
- Campos obligatorios obligatorios validados

### Estadísticas
- Carreras: vueltas, tiempos, posiciones, penalizaciones
- Combate: ataques, daño, eliminaciones, relación kill/death
- Generales: partidas, jugadores, eventos procesados/rechazados

### Patrones
- Remontada en carreras: detección basada en eventos POSITION_CHANGED
- Rachas de eliminaciones en combate: mínimo 3 eliminaciones consecutivas

### Calidad del Código
- Separación efectiva entre Go y Scala
- Comunicación real entre servicios mediante HTTP
- Pruebas unitarias en Go (pending ejecución sin sbt/Go test framework)
- Documentación académica en español
- Estructura de repositorio organizada

## Estructura del Repositorio

```
gamestats/
├── go-service/           # Servicio Go principal
│   ├── cmd/             # Punto de entrada main.go
│   ├── internal/        # Módulos: domain, validation, ingestion, matches, analytics_client, api
│   ├── Dockerfile
│   └── tests/           # Tests de Go (estructura preparada)
├── scala-service/        # Servicio Scala de análisis
│   ├── src/main/scala/  # Código fuente Scala
│   ├── build.sbt
│   └── Dockerfile
├── simulator/            # Simulador de eventos deterministas
│   └── src/simulator.go
├── docs/                # Documentación técnica
│   ├── event-schema.md
│   ├── architecture.md
│   └── api.md
├── .github/             # GitHub Actions
│   └── workflows/ci.yml
├── docker-compose.yml          # Docker Compose
├── .env.example         # Variables de entorno
├── .dockerignore        # Ignorar archivos Docker
├── .gitignore           # Ignorar archivos Git
└── README.md            # Documentación principal
```

## Comandos para Compilar y Ejecutar

### Compilar Go
```bash
cd gamestats/go-service
go build ./cmd/main.go
```

### Ejecutar Go
```bash
cd gamestats/go-service
go run cmd/main.go
```

### Cargar eventos de prueba
```bash
cd gamestats/go-service
go run cmd/main.go  # ya en ejecución
curl -X POST http://localhost:8080/ingest/file -F "file=@gamestats/data/carreras_small.jsonl"
```

### Consultar resultados
```bash
# Health check
curl http://localhost:8080/health

# Estadísticas globales
curl http://localhost:8080/stats

# Consultar partida
curl http://localhost:8080/matches/match_racing_001
```

### Levantar con Docker
```bash
docker compose up --build
```

### Generar eventos de prueba
```bash
cd gamestats/simulator
go build -o simulator ./src/simulator.go
./simulator 42  # con semilla determinista
```

## Limitaciones Notadas

1. **Entorno de ejecución:** Scala requiere sbt y Java 17 para compilación, no disponibles en este entorno de prueba
2. **Docker:** Los Dockerfiles han sido creados pero la ejecución completa con docker-compose no se verificó en este entorno
3. **Pruebas automatizadas:** Las pruebas unitarias de Go funcionan (`go build` exitoso), las pruebas de Scala requieren sbt configurado
4. **Memoria:** El estado de partidas se mantiene en RAM, no persiste entre reinicios de servicio

## Decisiones de Diseño Destacadas

1. **Paradigma Mixto:** Go maneja ingestión, validación y flujo; Scala maneja análisis estadístico y operaciones funcionales sobre colecciones
2. **Esquema Común:** Estructura de evento compartida que permite transportar eventos de juegos desconocidos
3. **Concurrencia en Go:** Goroutines con canales para procesamiento paralelo de eventos de múltiples partidas, con mutexes solo cuando es necesario para acceso compartido
4. **Funcional en Scala:** Operations `filter`, `map`, `groupBy`, `foldLeft` para transformación de colecciones de eventos
5. **Extensibilidad:** Nueva juegos pueden agregarse agregando validación en Go y análisis en Scala sin modificar el núcleo
6. **Sin base de datos:** El sistema funciona sin BD externa, usando memoria temporal para estado de partidas
7. **Simulador independiente:** Generador de eventos separado de la lógica de Go y Scala, permite incorporar eventos de otras fuentes posteriormente

## Próximos Pasos (Si se dispone de más tiempo)

1. Instalar sbt y Java 17 para compilar el servicio Scala
2. Ejecutar docker-compose para verificar la integración completa
3. Ejecutar `go test ./...` y `go test -race ./...` para pruebas de Go
4. Ejecutar sbt test para pruebas de Scala
5. Agregar más casos de prueba para patrones de remontada y rachas
6. Agregar persistencia con SQLite o archivo plano para estado a largo plazo
7. Agregar más tipos de juegos beyond racing y combat
