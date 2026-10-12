# GameStats - Arquitectura del Sistema

## Arquitectura General

```
Fuente Simulada → Go (Ingestión y Coordinación) → HTTP/REST → Scala (Análisis Estadístico) → Resultados JSON
                                                   ↑                                      |
                                                   └──────────────────────────────────────┘
```

## Módulo Go

### Responsabilidades

1. **Ingestión**: Recibir eventos desde archivo JSONL o API HTTP (`POST /events`)
2. **Validación**: Validar estructura común y específica por tipo de evento
3. **Control de duplicados**: Mantener identificadores de eventos para evitar procesamiento duplicado
4. **Estado temporal de partidas**: Mantener estado de partidas en memoria
5. **Agrupación por partida**: Agrupar eventos por `matchId` y `gameId`
6. **Flujos de procesamiento**: Decidir cuándo una partida está lista para analizarse
7. **Comunicación HTTP**: Enviar eventos a Scala mediante `POST /analyze`
8. **Consulta de resultados**: Exponer endpoints `GET /matches/{matchId}/analysis`, `GET /players/{playerId}/stats`, `GET /stats`
9. **Manejo de errores**: Comunicación fallida, tiempos de espera, respuestas HTTP inesperadas

### Tecnologías Go

- **Goroutines**: Procesamiento concurrente de eventos de múltiples partidas
- **Channels**: Comunicación entre goroutines, pasar eventos validados al procesador
- **Contextos**: Cancelación y tiempos de espera en operaciones HTTP y goroutines
- **Mutexes**: Sincronización de acceso concurrente al mapa de estados de partidas
- **sync.WaitGroup**: Coordinación del cierre ordenado del servicio

### Restricciones de Concurrencia

- Los mapas mutables NO se comparten entre goroutines sin sincronización
- Cada goroutine tiene su propio estado de partida o usa mutex para acceso compartido
- Límite razonable de procesamiento concurrente (configurable con `WORKER_COUNT`, default: 4)
- Cierre ordenado: pending eventos deben procesarse antes de apagar

## Módulo Scala

### Responsabilidades

1. **Recepción HTTP**: Endpoint `POST /analyze` para recibir partidas preparadas por Go
2. **Deserialización y validación**: Verificar condiciones necesarias para el análisis
3. **Conversión a estructuras**: Transformar JSON en tipos de datos funcionales
4. **Análisis separado por juego**: Separar `racing` de `combat`
5. **Transformaciones funcionales**: Usar `filter`, `map`, `flatMap`, `groupBy`, `foldLeft`, `reduceOption`
6. **Cálculo de estadísticas**: Métricas por partida, por jugador y agregadas
7. **Detección de patrones**: Remontadas en carreras, rachas de eliminación en combate
8. **Devolución de resultados JSON**: Estructura con `statistics` y `patterns`

### Tecnologías Scala 3

- `filter`: Filtrar eventos por tipo o condición
- `map`: Transformar cada evento a estructura de estadística
- `flatMap`: Aplanar colecciones de resultados por jugador
- `groupBy`: Agrupar eventos por jugador, por tipo, por partida
- `foldLeft`: Calcular acumuladores (sumas, promedios, conteos)
- `reduceOption`: Reducción segura con opción nula

## Comunicación entre Go y Scala

### Dirección

- **Go inicia**: Go agrupa eventos por partida y envía a Scala cuando la partida está lista
- **Scala recibe**: Endpoint `POST /analyze`
- **Respuesta**: Scala devuelve JSON con estadísticas y patrones

### URLs y Puertos (por variable de entorno)

| Servicio | Puerto por defecto | Variable de entorno |
|----------|-------------------|---------------------|
| Go | 8080 | `GO_SERVICE_PORT` |
| Scala | 8081 | `SCALA_SERVICE_PORT` |

### Formato de Solicitud Go→Scala

```
POST http://localhost:8081/analyze
Content-Type: application/json

{
  "matchId": "match_001",
  "gameId": "racing",
  "events": [/* eventos validados y agrupados */]
}
```

### Formato de Respuesta Scala→Go

```
200 OK:
{
  "status": "analyzed",
  "matchId": "match_001",
  "gameId": "racing",
  "statistics": { ... },
  "patterns": { ... }
}

400 Bad Request:
{
  "status": "error",
  "matchId": "match_001",
  "errors": ["motivo 1", "motivo 2"]
}

503 Service Unavailable:
{
  "status": "unavailable",
  "matchId": "match_001",
  "error": "Scala service not ready"
}
```

### Manejo de Errores

- Si Scala no está disponible: Go reintenta con backoff exponencial
- Si el análisis falla: La partida NO se marca como analizada exitosamente
- Los reintentos no deben producir resultados duplicados (usar `matchId` + `eventId` como idempotencia)