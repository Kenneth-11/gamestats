# GameStats - Decisiones de diseño (justificación)

## Por qué carreras (`racing`) y combate (`combat`)

Se eligieron dos naturalezas opuestas para demostrar que el núcleo
genérico (esquema común + `matchId`/`playerId`) sirve para juegos distintos:

- **Carreras**: desempeño continuo e individual (tiempos, posiciones).
  Los eventos son periódicos (`LAP_COMPLETED`) y la historia de una
  partida es la evolución de marcas. Habilita promedios, mejor vuelta,
  consistencia y remontadas.
- **Combate**: desempeño discreto e interactivo (daño, eliminaciones).
  Los eventos relacionan a dos jugadores (atacante/víctima) y exigen
  combinar tipos (`PLAYER_ATTACKED` + `PLAYER_ELIMINATED`) para métricas
  como daño por eliminación o K/D. Habilita rachas y liderazgos.

## Por qué estos eventos

| Evento | Justificación |
|--------|---------------|
| `MATCH_STARTED` / `MATCH_FINISHED` | Delimitan la partida; Go decide cuándo enviar a Scala. Sin ellos no hay agrupación confiable. |
| `PLAYER_JOINED` | Censo de participantes por partida y por jugador global. |
| `LAP_COMPLETED` (lapNumber, lapTimeMs, averageSpeed, position) | Cuatro campos bastan para vueltas, mejor/promedio, velocidad y evolución de posición sin telemetría completa. |
| `POSITION_CHANGED` (from/to) | Evidencia directa de remontadas; con `LAP_COMPLETED.position` se corrobora. |
| `PENALTY_RECEIVED` | Factor explicativo del rendimiento (consistencia vs sanciones). |
| `PLAYER_ATTACKED` (attacker, target, damage, weapon, distance) | Base del daño total/promedio y del análisis por arma y distancia. |
| `PLAYER_ELIMINATED` (victim, killer, weapon) | Cierra el ciclo ataque→eliminación; permite K/D, líderes y rachas. `killer="world"` cubre muertes ambientales. |

## División Go (imperativo) / Scala (funcional)

- **Go**: I/O y estado mutable (HTTP, archivos, mapa de partidas),
  validación con salida temprana, concurrencia con goroutines+channels
  y reintentos con backoff. Todo lo que el paradigma imperativo hace
  directo y eficiente. Go **no** calcula estadísticas finales.
- **Scala**: transformaciones puras sobre colecciones inmutables
  (`filter`/`map`/`flatMap`/`groupBy`/`foldLeft`/`reduceOption`/`collect`
  + composición). Las stats son pipelines deterministas sin efectos
  secundarios, fáciles de testear (`sbt test`, 10 casos).

## Extensibilidad (nuevo juego sin reescribir el núcleo)

1. Nuevo `gameId` con el esquema común (Go lo valida, agrupa y reenvía).
2. Validación específica en `domain/event.go` + `ValidateSpecific`.
3. Rama de análisis en `Analytics.analyze` con `filter/groupBy/foldLeft`.
4. Patrones como funciones puras `List[Event] => List[JObj]`.
   Juegos desconocidos pasan por Go y Scala responde nota sin romper.
