# GameStats - Schema de Eventos

## Estructura Común de Eventos

Todo evento debe contener como mínimo los siguientes campos:

| Campo | Tipo | Obligatorio | Descripción |
|-------|------|-----------|-------------|
| `eventId` | string | Sí | Identificador único del evento (UUID v4 recomendado) |
| `timestamp` | string (ISO 8601 UTC) | Sí | Instante en que ocurrió el evento |
| `gameId` | string | Sí | Identificador del juego (`racing` o `combat`) |
| `gameVersion` | string | Sí | Versión del formato o del juego |
| `matchId` | string | Sí | Identificador de la partida a la que pertenece |
| `playerId` | string | No | Jugador relacionado (puede ser vacío para eventos globales) |
| `type` | string | Sí | Tipo de evento (ej. `MATCH_STARTED`, `LAP_COMPLETED`) |
| `action` | string | No | Acción específica dentro del tipo de evento |
| `data` | object | No | Información adicional específica del evento |

## Eventos Globales (sin playerId)

Los siguientes eventos son de naturaleza global y no llevan `playerId`:

- `MATCH_STARTED`: inicio de una partida
- `MATCH_FINISHED`: fin de una partida

Estos eventos usan `playerId` vacío o `null` y identifican al partida mediante `matchId`.

## Tipos de Juego y Eventos

### Juego A: Carreras (`racing`)

#### Eventos sugeridos:

1. **MATCH_STARTED**
   - `data`: vacío o con `mode`, `track`
   - Ejemplo: `{"mode": "lap", "track": "city_circuit"}`

2. **PLAYER_JOINED**
   - `data`: `playerName`, `team`

3. **LAP_COMPLETED** (campo `data` obligatorio)
   - `data`: `lapNumber` (number), `lapTimeMs` (number), `averageSpeed` (number), `position` (number)
   - Ejemplo: `{"lapNumber": 3, "lapTimeMs": 950, "averageSpeed": 140, "position": 2}`

4. **POSITION_CHANGED**
   - `data`: `fromPosition` (number), `toPosition` (number)
   - Ejemplo: `{"fromPosition": 5, "toPosition": 3}`

5. **PENALTY_RECEIVED**
   - `data`: `penaltyType` (string), `durationMs` (number)
   - Ejemplo: `{"penaltyType": "collision", "durationMs": 5000}`

6. **MATCH_FINISHED**
   - `data`: `winnerPlayerId` (string), `durationMs` (number)
   - Ejemplo: `{"winnerPlayerId": "player_123", "durationMs": 3600000}`

#### Validación racing:
- `LAP_COMPLETED` requiere `lapNumber`, `lapTimeMs`, `averageSpeed`, `position` en `data`
- `lapNumber` debe ser > 0
- `lapTimeMs` debe ser > 0
- `position` debe ser >= 1

### Juego B: Combate (`combat`)

#### Eventos sugeridos:

1. **MATCH_STARTED**
   - `data`: vacío o con `mode`, `map`

2. **PLAYER_JOINED**
   - `data`: `playerName`, `team`, `startingHealth`

3. **PLAYER_ATTACKED** (campo `data` obligatorio)
   - `data`: `attacker` (string), `target` (string), `damage` (number), `weapon` (string), `distance` (number)
   - Ejemplo: `{"attacker": "player_456", "target": "player_789", "damage": 45, "weapon": "rifle", "distance": 50}`

4. **PLAYER_ELIMINATED** (campo `data` obligatorio)
   - `data`: `victim` (string), `killer` (string), `weapon` (string), `distance` (number)
   - Ejemplo: `{"victim": "player_789", "killer": "player_456", "weapon": "rifle", "distance": 50}`

5. **MATCH_FINISHED**
   - `data`: `winnerPlayerId` (string), `durationMs` (number), `killsLeader` (string)
   - Ejemplo: `{"winnerPlayerId": "player_456", "durationMs": 1800000, "killsLeader": "player_456"}`

#### Validación combat:
- `PLAYER_ATTACKED` requiere `attacker`, `target`, `damage`, `weapon`, `distance` en `data`
- `damage` debe ser >= 0
- `PLAYER_ELIMINATED` requiere `victim` y `killer` en `data`
- El `killer` puede ser `"world"` para eliminaciones por caída o medio ambiente

## Validación de Campos Obligatorios por Tipo

| Campo | MATCH_STARTED | PLAYER_JOINED | LAP_COMPLETED | POSITION_CHANGED | PENALTY_RECEIVED | MATCH_FINISHED | PLAYER_ATTACKED | PLAYER_ELIMINATED |
|-------|--------------|---------------|---------------|------------------|-------------------|----------------|-----------------|-------------------|
| `eventId` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `timestamp` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `gameId` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `gameVersion` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `matchId` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `type` | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| `action` | opcional | opcional | opcional | opcional | opcional | opcional | ✓ | ✓ |
| `data` | opcional | opcional | ✓ | opcional | opcional | opcional | ✓ | ✓ |

## Ejemplos JSON

### Evento racing - LAP_COMPLETED

```json
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

### Evento racing - MATCH_STARTED

```json
{
  "eventId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "timestamp": "2026-03-15T14:00:00Z",
  "gameId": "racing",
  "gameVersion": "1.0.0",
  "matchId": "match_001",
  "playerId": "",
  "type": "MATCH_STARTED",
  "action": "start",
  "data": {
    "mode": "lap",
    "track": "city_circuit"
  }
}
```

### Evento combat - PLAYER_ATTACKED

```json
{
  "eventId": "z9y8x7w6-v5u4-3t2s-rqpo-0mnlkjihgfedcba",
  "timestamp": "2026-03-15T15:00:00Z",
  "gameId": "combat",
  "gameVersion": "1.0.0",
  "matchId": "match_002",
  "playerId": "player_456",
  "type": "PLAYER_ATTACKED",
  "action": "shoot",
  "data": {
    "attacker": "player_456",
    "target": "player_789",
    "damage": 45,
    "weapon": "rifle",
    "distance": 50
  }
}
```

### Evento combat - MATCH_FINISHED

```json
{
  "eventId": "m0n1l2k3-j4o5-6p7q-8r9s-0t1u2v3w4x5y",
  "timestamp": "2026-03-15T16:00:00Z",
  "gameId": "combat",
  "gameVersion": "1.0.0",
  "matchId": "match_002",
  "playerId": "",
  "type": "MATCH_FINISHED",
  "action": "end",
  "data": {
    "winnerPlayerId": "player_456",
    "durationMs": 1800000,
    "killsLeader": "player_456"
  }
}
```

## Reglas de Validación

### Campos Comunes Obligatorios
- `eventId` debe ser un identificador no vacío
- `timestamp` debe ser un timestamp ISO 8601 válido en UTC
- `gameId` debe ser uno de los valores permitidos (`racing`, `combat`)
- `gameVersion` debe ser un string no vacío
- `matchId` debe ser un identificador no vacío
- `type` debe ser un string no vacío

### Identificadores Vacíos o Inválidos
- `eventId` vacío -> ERROR de validación estructural
- `matchId` vacío -> ERROR de validación estructural
- `gameId` inválido (no `racing` ni `combat`) -> ERROR de validación estructural

### Timestamps Incorrectos
- Formato no ISO 8601 -> ERROR de validación estructural
- Timestamp en el futuro -> EVENTO RECHAZADO (política de negocio)
- Falta `timestamp` -> ERROR de validación estructural

### Tipos de Eventos Ausentes
- `type` vacío o nulo -> ERROR de validación estructural
- `type` no reconocido para el `gameId` especificado -> EVENTO DESCONOCIDO (no necesariamente rechazado)

### JSON Malformado
- Cualquier evento que no pueda ser deserializado -> ERROR de validación, retorno de error HTTP 400

### Eventos Duplicados
- Detectado por `eventId` repetido dentro de la misma partida -> EVENTO RECHAZADO

### Eventos en Partidas Inexistentes o Finalizadas
- Evento con `matchId` que no existe en el estado actual -> EVENTO RECHAZADO
- Evento en partida ya finalizada (`MATCH_FINISHED` ya procesado) -> EVENTO RECHAZADO

### Campos Específicos Incorrectos
- `LAP_COMPLETED` sin `lapTimeMs` negativo -> ERROR de validación
- `damage` negativo en combate -> ERROR de validación
- `position` fuera de rango en racing -> ERROR de validación

## Diferenciación de Errores

| Tipo | Descripción | Acción |
|------|-------------|--------|
| `structural_error` | Falta campo obligatorio, formato inválido | Rechazar con HTTP 400, no almacenar |
| `unknown_event` | Evento con `type` no reconocido pero estructura válida | Aceptar, almacenar, no analizar hasta tener datos suficientes |
| `business_inconsistent` | Evento válido pero lógicamente incorrecto (ej. timestamp futuro, partida finalizada) | Rechazar, marcar para reporte |

## Extensibilidad

La estructura común permite transportar eventos de tipos desconocidos para el núcleo de Go siempre que cumplan el esquema. Scala solo produce análisis especializado para los juegos que implementa (`racing`, `combat`). Eventos de otros juegos pasan a través de Go sin ser analizados estadísticamente hasta que se implementen.