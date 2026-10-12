# GameStats - Plataforma Multilenguaje para Estadísticas de Videojuegos

## Descripción

GameStats es una plataforma universitaria para recibir, validar, organizar y analizar eventos producidos durante partidas de videojuegos. El sistema demuestra la integración de dos paradigmas de programación:

- **Go:** ingestión, validación estructural, concurrencia y administración del flujo
- **Scala:** análisis estadístico y operaciones funcionales sobre colecciones

El sistema recibe eventos en formato JSON, los valida contra un esquema común y las estadísticas se calculan separando los análisis por juego (carreras y combate).

## Requisitos Previos

- Go 1.26.5 o superior
- Scala 3.x y sbt (para el servicio Scala)
- Java 17 LTS
- Docker y Docker Compose (para ejecución integrada)
- Acceso a red para descargar dependencias

## Arquitectura General

```
Fuente Simulada → Go (Ingestión y Coordinación) → HTTP/REST → Scala (Análisis) → Resultados JSON
```

### Módulo Go

Servicio independiente responsable de:
- Recibir eventos desde archivo JSONL o API HTTP
- Validar la estructura común de cada evento
- Control de identificadores para evitar duplicados
- Mantener estado temporal de partidas
- Agrupar eventos por juego y partida
- Enviar partidas a Scala para análisis
- Exponer endpoints de consulta HTTP

### Módulo Scala

Servicio independiente con operaciones funcionales que:
- Recibe partidas preparadas por Go
- Calcula estadísticas por partida, por jugador y agregadas
- Detecta patrones secuenciales (remontadas, rachas de eliminación)
- Devuelve resultados estructurados en JSON

### Simulador

Generador de eventos deterministas y reproducibles que produce:
- Múltiples partidas de carreras (`racing`)
- Múltiples partidas de combate (`combat`)
- Varios jugadores por partida
- Diferentes resultados y escenarios

## Instalación

1. Clonar el repositorio
2. Instalar dependencias de Go:
   ```
   cd gamestats/go-service
   go mod download
   ```
3. Instalar dependencias de Scala (requiere sbt):
   ```
   cd gamestats/scala-service
   sbt compile
   ```
4. Construir el simulador:
   ```
   cd gamestats/simulator
   go build -o simulator ./src/simulator.go
   ```

## Configuración

Copiar el archivo de variables de entorno:
```
cp gamestats/.env.example .env
```

Las variables disponibles:
- `GO_SERVICE_PORT`: Puerto del servicio Go (default: 8080)
- `SCALA_SERVICE_PORT`: Puerto del servicio Scala (default: 8081)
- `SCALA_SERVICE_URL`: URL del servicio Scala
- `SIMULATOR_SEED`: Semilla para el generador de eventos deterministas
- `WORKER_COUNT`: Número de workers concurrentes en Go

## Ejecución Local

### Iniciar el servicio Go
```
cd gamestats/go-service
go run cmd/main.go
```

### Iniciar el servicio Scala
El servicio Scala necesita sbt:
```
cd gamestats/scala-service
sbt run
```

### Ejecutar el simulador
```
cd gamestats/simulator
./simulator 42
```

Esto generará eventos de ejemplo en la carpeta `data/`.

### Cargar eventos al servicio Go
```
curl -X POST http://localhost:8080/ingest/file -F "file=@data/carreras_small.jsonl"
```

### Probar los endpoints
```
# Health check
curl http://localhost:8080/health

# Consultar partida
curl http://localhost:8080/matches/match_racing_001

# Consultar estadísticas globales
curl http://localhost:8080/stats
```

## Ejecución con Docker

Levantar los servicios integrados:
```
docker compose up --build
```

Los servicios estarán disponibles en:
- Go: http://localhost:8080
- Scala: http://localhost:8081

Para ver los logs:
```
docker compose logs -f
```

Para detener:
```
docker compose down
```

## Generación de Eventos

El simulador en `gamestats/simulator/src/simulator.go` genera datos deterministas usando una semilla configurable. Con la semilla `42`, los eventos generados son reproducibles.

El simulador produce:

### Carreras (`racing`)
- `MATCH_STARTED` con modo `lap` y track `city_circuit`
- `PLAYER_JOINED` para 2-6 jugadores
- `LAP_COMPLETED` con tiempos variables (800-5800 ms)
- `POSITION_CHANGED` con evolución de posiciones
- `PENALTY_RECEIVED` con tipos `collision`, `speeding`, `cut_corner`
- `MATCH_FINISHED` con ganador y duración

### Combate (`combat`)
- `MATCH_STARTED`
- `PLAYER_JOINED` para 2-5 jugadores
- `PLAYER_ATTACKED` con daño, weapon y distancia
- `PLAYER_ELIMINATED` con killer y weapon
- `MATCH_FINISHED` con ganador y duración

Usando diferentes semillas se pueden generar variaciones de los mismos escenarios, permitiendo probar casos borde sin cambiar la lógica.

## Ejemplos de Solicitudes HTTP

### Health Check
```
GET /health
```
Respuesta: `{"status": "ok", "service": "gamestats-go"}`

### Recibir Eventos (POST /events)
```json
POST /events
Content-Type: application/json

{
  "eventId": "c2f3a4b5-8d6e-4f1a-bcde-5f6a7b8c9d01",
  "timestamp": "2026-03-15T14:30:00Z",
  "gameId": "racing",
  "gameVersion": "1.0.0",
  "matchId": "match_001",
  "playerId": "player_123",
  "type": "LAP_COMPLETED",
  "action": "lap_done",
  "data": {
    "lapNumber": 5,
    "lapTimeMs": 985,
    "averageSpeed": 142,
    "position": 3
  }
}
```
Respuesta: `{"status": "accepted", "eventId": "c2f3a4b5-8d6e-4f1a-bcde-5f6a7b8c9d01"}`

### Cargar Archivo JSONL (POST /ingest/file)
```bash
curl -X POST http://localhost:8080/ingest/file -F "file=@data/carreras_small.jsonl"
```
Respuesta: `{"status": "accepted", "processed": 26, "rejected": 0}`

### Consultar Partida (GET /matches/{matchId})
```bash
curl http://localhost:8080/matches/match_racing_001
```
Respuesta estructurada con estadísticas de la partida.

### Consultar Estadísticas de Jugador (GET /players/{playerId}/stats)
```bash
curl http://localhost:8080/players/player_base/stats
```
Resumen de carreras, vueltas, penalizaciones, etc.

### Consultar Estadísticas Globales (GET /stats)
```bash
curl http://localhost:8080/stats
```
Total de partidas, jugadores, eventos procesados y rechazados.

## Ejemplos de Resultados JSON

### Análisis de Partida (desde Scala)
```json
{
  "status": "analyzed",
  "matchId": "match_racing_001",
  "gameId": "racing",
  "statistics": {
    "lapsCompleted": 10,
    "bestLapTime": 1539,
    "averageLapTime": 2154.5,
    "penalties": 2,
    "positionEvolution": [4, 3, 3, 2, 1]
  },
  "patterns": [
    {
      "type": "remontada",
      "playerId": "player_3",
      "detected": true,
      "detail": "Jugador comenzó en posición 4 y terminó en posición 1"
    }
  ]
}
```

### Estadísticas de Combate
```json
{
  "status": "analyzed",
  "matchId": "match_combat_001",
  "gameId": "combat",
  "statistics": {
    "totalAttacks": 15,
    "totalDamage": 723,
    "averageDamage": 48.2,
    "totalEliminations": 8,
    "timesEliminated": 3,
    "topEliminator": "player_base"
  }
}
```

### Estadísticas Generales
```json
{
  "status": "ok",
  "totalMatches": 5,
  "totalPlayers": 12,
  "totalEvents": 89,
  "rejectedEvents": 3,
  "durationMs": 3600000
}
```

## Estadísticas y Fórmulas

### Estadísticas de Carreras (por partida)

- `lapsCompleted`: Cantidad de vueltas completadas (count de eventos LAP_COMPLETED)
- `bestLapTime`: Mejor tiempo de vuelta en milisegundos (mínimo de todos los lapTimeMs)
- `averageLapTime`: Tiempo promedio de vuelta = sum(lapTimeMs) / count
- `penalties`: Cantidad total de penalizaciones recibidas por todos los jugadores
- `positionEvolution`: Secuencia de posiciones a lo largo de la carrera

### Estadísticas de Carreras (por jugador)

- `racesCount`: Número de carreras en las que participó el jugador
- `totalLaps`: Total de vueltas completadas por el jugador
- `bestPersonalLap`: Mejor vuelta personal del jugador (mínimo de sus lapTimes)
- `averageLapTime`: Promedio de tiempo por vuelta del jugador
- `penaltiesCount`: Cantidad de penalizaciones del jugador
- `racesFinishedPercentage`: Porcentaje de carreras terminadas = (carreras finalizadas / total de carreras registradas) × 100

### Estadísticas de Combate (por partida)

- `totalAttacks`: Cantidad total de ataques registrados
- `totalDamage`: Suma de todos los daños infligidos
- `averageDamage`: Daño promedio por ataque = totalDamage / totalAttacks
- `totalEliminations`: Eliminaciones totales en la partida
- `timesEliminated`: Veces que los jugadores fueron eliminados
- `topEliminator`: Jugador con más eliminaciones

### Estadísticas de Combate (por jugador)

- `eliminations`: Eliminaciones conseguidas por el jugador
- `timesEliminated`: Veces que el jugador fue eliminado
- `damageInflictado`: Daño total infligido por el jugador
- `averageDamage`: Daño promedio por ataque del jugador
- `killDeathRatio`: Relación eliminaciones/veces eliminado
  - Si `timesEliminated = 0`: ratio = eliminations (nunca fue eliminado)
  - Si `timesEliminated > 0`: ratio = eliminations / timesEliminated

### Estadísticas Generales

- `totalMatches`: Cantidad de partidas procesadas
- `totalPlayers`: Cantidad de jugadores participantes (únicos en todos los matches)
- `totalEvents`: Cantidad total de eventos procesados
- `rejectedEvents`: Cantidad de eventos rechazados por validación fallida
- `durationMs`: Duración de la partida cuando existen timestamps suficientes

## Reglas de Detección de Patrones

### Patrón 1: Remontada en Carreras

Se detecta una remontada cuando un jugador:
1. Comienza o aparece en una posición inferior (ej. posición 5)
2. Presenta una evolución de posición verificable mediante eventos `POSITION_CHANGED` o `LAP_COMPLETED` con cambios de posición
3. Termina en una posición mejor que la inicial (ej. posición 1)
4. Cumple las condiciones mínimas de evidencia (existán eventos suficientes para demostrar el cambio de posición)

**Documentación exacta:** Se requiere al menos un evento `POSITION_CHANGED` que muestre un cambio de posición de N a M donde M < N, seguido de eventos posteriores que confirmen la posición final mejorada.

### Patrón 2: Secuencia de Combate (Racha de Eliminaciones)

Se detecta una racha de eliminaciones cuando un jugador consigue al menos tres eliminaciones sin ser eliminado entre ellas:

1. Usar el orden temporal de los eventos
2. Criterio de desempate determinista para timestamps iguales
3. Distinguir quién realiza la eliminación y quién es eliminado
4. Las eliminaciones sin atacante o con eventos inconsistentes no cuentan para la racha

**Cálculo:** Recorrer los eventos `PLAYER_ELIMINATED` en orden cronológico, contar eliminaciones consecutivas del mismo jugador, reiniciar el contador si el jugador es él mismo eliminado.

## División de Responsabilidades entre Go y Scala

| Responsabilidad | Go | Scala |
|----------------|-----|--------|
| Validación de estructura | ✅ | ❌ (confía en validación previa) |
| Control de duplicados | ✅ | ❌ |
| Gestión de estado de partida | ✅ | ❌ |
| Ingestión de eventos | ✅ | ❌ |
| Cálculo de estadísticas | ❌ | ✅ (usando operaciones funcionales) |
| Detección de patrones | ❌ | ✅ (remontadas, rachas) |
| Comunicación HTTP | ✅ (cliente) | ✅ (servidor) |
| Deserialización JSON | ✅ | ✅ |

### Razones para utilizar concurrencia en Go

- Procesar eventos de varias partidas en paralelo sin mezclar estados
- Mantener el flujo de procesamiento fluido bajo carga
- Goroutines con channels para comunicación segura
- Contextos para cancelación y tiempos de espera
- Mutexes solo cuando es necesario para acceso compartido a mapas

### Razones para utilizar operaciones funcionales en Scala

- `filter`: Seleccionar eventos relevantes por tipo o condición
- `map`: Transformar cada evento a métrica intermedia
- `groupBy`: Agrupar eventos por jugador, tipo de evento o partida
- `foldLeft`: Acumular conteos, sumas y promedios de manera determinista
- `reduceOption`: Reducciones seguras con manejo de nulos
- `collect`: Filtrar y transformar colecciones en una sola operación
- Composición de funciones para pipelines de procesamiento

Estas operaciones son adecuadas para análisis de eventos porque los datos vienen en forma de colecciones (listas de eventos) y las operaciones funcionales permiten pipelines expresivos y deterministas sin efectos secundarios.

## Limitaciones de la Solución

1. **Sin sbt instalado:** El servicio Scala no puede compilarse en entornos sin sbt configurado
2. **Memoria en memoria:** El estado de partidas se mantiene en memoria RAM, no persiste entre reinicios
3. **Sin base de datos:** No hay persistencia a largo plazo de eventos o resultados
4. **Scala sin instalar:** Requiere configuración adicional en entornos que no tengan Scala 3 + sbt
5. **Concurrencia limitada:** El límite de workers concurrentes es configurable pero compartido en un solo proceso
6. **Patrones solo para racing/combat:** Los patrones de remontada y rachas solo están implementados para los dos juegos soportados

## Decisiones de Diseño para Añadir Nuevos Juegos

La plataforma está diseñada para ser extensible:

1. **Nuevo tipo de juego:** Agregar el identificador `gameId` en el esquema común (`racing`, `combat`, y por ejemplo `rpg`)
2. **Eventos específicos:** Agregar validación en el validador Go para campos específicos del nuevo juego
3. **Análisis en Scala:** Implementar operaciones `filter`, `groupBy` y `foldLeft` específicas para los nuevos tipos de eventos
4. **Patrones:** Agregar reglas de detección de patrones en el servicio Scala siguiendo el mismo patrón de secuencias de eventos
5. **Sin modificar el núcleo:** El núcleo de Go no necesita modificaciones si el nuevo juego usa el esquema común de eventos

Los nombres de campos específicos pueden diferir del esquema común, manteniendo la información particular separada de la estructura global. Los eventos desconocidos que cumplan el esquema común pasan a través de Go sin ser analizados estadísticamente hasta que Scala implemente soporte para ellos.

## Estructura del Repositorio

```
gamestats/
├── go-service/           # Servicio Go principal
│   ├── cmd/             # Punto de entrada
│   ├── internal/        # Módulos internos (domain, validation, ingestion, etc.)
│   ├── Dockerfile
│   └── tests/
├── scala-service/        # Servicio Scala de análisis
│   ├── src/             # Código fuente Scala
│   ├── build.sbt
│   └── Dockerfile
├── simulator/            # Simulador de eventos
│   └── src/
├── docs/                # Documentación
├── .github/             # GitHub Actions
│   └── workflows/
├── docker-compose.yml          # Docker Compose
├── .env.example         # Variables de entorno
├── .dockerignore        # Ignorar archivos Docker
├── .gitignore           # Ignorar archivos Git
└── README.md            # Este archivo
```

## Ejecución de Pruebas

### Pruebas de Go

```
go test ./...
go test -race ./...
```

Las pruebas cubren:
- Validación de eventos correctos
- Rechazo de JSON malformado
- Rechazo de campos obligatorios ausentes
- Tratamiento de eventos duplicados
- Agrupación por partida
- Aislamiento de estados entre partidas
- Manejo de partidas finalizadas
- Comunicación con Scala usando HTTP de prueba
- Reintentos y fallos de comunicación
- Procesamiento concurrente

### Pruebas de Scala

Ejecutar con sbt:
```
sbt test
```

Las pruebas cubren:
- Deserialización y validación
- Estadísticas de carreras
- Estadísticas de combate
- Agregaciones por jugador
- Agregaciones por partida
- Tratamiento de colecciones vacías
- Valores faltantes
- Orden temporal
- Remontadas
- Rachas de eliminaciones
- Eventos desconocidos
- Resultados JSON esperados

### Integración

Comprobar el recorrido completo:
1. El simulador produce eventos
2. Go los recibe y valida
3. Go agrupa los eventos por partida
4. Go envía una partida a Scala
5. Scala calcula las estadísticas y patrones
6. Go recibe el resultado
7. El cliente consulta el análisis mediante la API de Go

## Solución de Errores Frecuentes

1. **Error: connection refused en puerto 8080**
   - Verificar que el servicio Go esté en ejecución
   - Revisar que el puerto no esté ocupado por otro proceso

2. **Error: JSON malformado en POST /events**
   - Verificar que el JSON esté bien formado
   - Validar que todos los campos comunes obligatorios estén presentes

3. **Error: partida no encontrada (404)**
   - El matchId no ha sido procesado aún
   - Verificar que los eventos hayan sido cargados correctamente

4. **Error: servicio Scala no disponible (503)**
   - Verificar que el contenedor Scala esté saludable
   - Revisar la variable SCALA_SERVICE_URL

5. **Eventos rechazados en la consola**
   - Los eventos deben tener todos los campos comunes obligatorios
   - Verificar que gameId sea 'racing' o 'combat'
   - Validar que los timestamps sean ISO 8601 en UTC

6. **Resultados inesperados en estadísticas**
   - Verificar que el simulador se haya ejecutado con la semilla correcta
   - Revisar que los eventos hayan sido agrupados por partida correcta

7. **Compilación Scala falla**
   - Asegurarse de tener sbt y Java 17 instalados
   - Ejecutar `sbt compile` para ver errores específicos
