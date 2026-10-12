# GameStats - Contratos API

## Endpoints del Servicio Go

### `GET /health`

- **Descripción**: Verificar disponibilidad del servicio
- **Método**: GET
- **Respuestas**:
  - `200 OK`: `{"status": "ok", "service": "gamestats-go"}`
  - `503 Service Unavailable`: `{"status": "error", "message": "service unavailable"}`

### `POST /events`

- **Descripción**: Recibir un evento individual validado
- **Método**: POST
- **Content-Type**: application/json
- **Cuerpo**: Objeto JSON con evento (formato común o específico por juego)
- **Respuestas**:
  - `202 Accepted`: `{"status": "accepted", "eventId": "..."}`
  - `400 Bad Request`: `{"status": "error", "errors": ["campo obligatorio faltante"]}`
  - `422 Unprocessable Entity`: `{"status": "invalid", "errors": ["razón de negocio"]}`

### `POST /ingest/file`

- **Descripción**: Cargar un archivo JSONL con múltiples eventos
- **Método**: POST
- **Content-Type**: multipart/form-data o application/json (arreglo de eventos)
- **Cuerpo**: `{"events": [...]} o archivo JSONL
- **Respuestas**:
  - `202 Accepted`: `{"status": "ingesting", "processed": N, "rejected": M}`
  - `400 Bad Request`: Eventos malformados

### `GET /matches/{matchId}`

- **Descripción**: Consultar el estado de una partida
- **Método**: GET
- **Parametros**: `matchId` en la URL
- **Respuestas**:
  - `200 OK`: Estado actual de la partida con eventos acumulados
  - `404 Not Found`: `{"status": "not_found", "message": "match not found"}`

### `GET /matches/{matchId}/analysis`

- **Descripción**: Consultar el análisis de una partida (si ya fue enviado a Scala)
- **Método**: GET
- **Parametros**: `matchId` en la URL
- **Respuestas**:
  - `200 OK`: Resultado del análisis desde Scala
  - `202 Accepted`: `{"status": "pending_analysis", "message": "analysis in progress"}`
  - `404 Not Found`: partida no existe

### `GET /players/{playerId}/stats`

- **Descripción**: Consultar estadísticas agregadas de un jugador
- **Método**: GET
- **Parametros**: `playerId` en la URL
- **Respuestas**:
  - `200 OK`: Estadísticas del jugador (por juego y global)
  - `404 Not Found`: `{"status": "not_found", "message": "player not found"}`

### `GET /stats`

- **Descripción**: Consultar estadísticas globales
- **Método**: GET
- **Respuestas**:
  - `200 OK`: `{"totalMatches": N, "totalPlayers": M, "totalEvents": P, "rejectedEvents": R}`
  - `503 Service Unavailable`: Si el servicio no está completamente inicializado

## Endpoints del Servicio Scala

### `GET /health`

- **Descripción**: Verificar disponibilidad del servicio de análisis
- **Método**: GET
- **Respuestas**:
  - `200 OK`: `{"status": "ok", "service": "gamestats-scala"}`

### `POST /analyze`

- **Descripción**: Analizar una partida enviada por Go
- **Método**: POST
- **Content-Type**: application/json
- **Cuerpo**:
  ```json
  {
    "matchId": "match_001",
    "gameId": "racing",
    "events": [...]
  }
  ```
- **Respuestas**:
  - `200 OK`: Análisis completado
    ```json
    {
      "status": "analyzed",
      "matchId": "match_001",
      "gameId": "racing",
      "statistics": { ... },
      "patterns": { ... }
    }
    ```
  - `400 Bad Request`: Datos inválidos
    ```json
    {
      "status": "error",
      "matchId": "match_001",
      "errors": ["motivo 1"]
    }
    ```
  - `503 Service Unavailable`: Scala no listo

## Códigos HTTP

| Código | Significado | Cuándo Usar |
|--------|-------------|-------------|
| 200 | OK | Operación exitosa, datos disponibles |
| 202 | Accepted | Petición aceptada, procesamiento en curso |
| 400 | Bad Request | JSON malformato, campos obligatorios faltantes |
| 422 | Unprocessable Entity | Evento válido estructura pero inconsistente negocio |
| 404 | Not Found | partida o jugador no existe |
| 503 | Service Unavailable | Servicio dependiente no disponible (Scala caído) |

## Estructura JSON de Respuesta

### Éxito General

```json
{
  "status": "ok",
  "matchId": "match_001",
  "gameId": "racing",
  "statistics": { ... },
  "patterns": { ... }
}
```

### Error Estructural

```json
{
  "status": "error",
  "matchId": "match_001",
  "errors": [
    "campo obligatorio 'eventId' faltante",
    "timestamp inválido: no es ISO 8601"
  ]
}
```

### Error de Negocio

```json
{
  "status": "business_error",
  "matchId": "match_001",
  "errors": [
    "partida ya finalizada",
    "evento duplicado detectado"
  ]
}
```