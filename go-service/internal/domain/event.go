package domain

import (
	"errors"
	"fmt"
	"time"
)

// EventType tipo de evento
type EventType string

const (
	EventTypeMatchStarted  EventType = "MATCH_STARTED"
	EventTypePlayerJoined  EventType = "PLAYER_JOINED"
	EventTypeLapCompleted  EventType = "LAP_COMPLETED"
	EventTypePositionChanged EventType = "POSITION_CHANGED"
	EventTypePenaltyReceived EventType = "PENALTY_RECEIVED"
	EventTypeMatchFinished EventType = "MATCH_FINISHED"
	EventTypePlayerAttacked  EventType = "PLAYER_ATTACKED"
	EventTypePlayerEliminated EventType = "PLAYER_ELIMINATED"
	EventTypePlayerRespawned EventType = "PLAYER_RESPAWNED"
)

// Evento representa un evento de juego con su estructura común
type Event struct {
	EventID   string      `json:"eventId"`
	Timestamp time.Time   `json:"timestamp"`
	GameID    string      `json:"gameId"`
	GameVersion string   `json:"gameVersion"`
	MatchID   string      `json:"matchId"`
	PlayerID  string      `json:"playerId"`
	Type      EventType   `json:"type"`
	Action    string      `json:"action"`
	Data      interface{} `json:"data"`
}

// Validate verifica la estructura común del evento
func (e *Event) Validate() error {
	if e.EventID == "" {
		return errors.New("eventId es obligatorio y no puede estar vacío")
	}

	if e.Timestamp.IsZero() {
		return errors.New("timestamp es obligatorio")
	}

	if e.GameID == "" {
		return errors.New("gameId es obligatorio")
	}

	if e.GameID != "racing" && e.GameID != "combat" {
		return fmt.Errorf("gameId inválido: %s, debe ser 'racing' o 'combat'", e.GameID)
	}

	if e.GameVersion == "" {
		return errors.New("gameVersion es obligatorio")
	}

	if e.MatchID == "" {
		return errors.New("matchId es obligatorio y no puede estar vacío")
	}

	if string(e.Type) == "" {
		return errors.New("type es obligatorio")
	}

	return nil
}

// ValidateSpecific valida campos específicos por tipo de juego y evento
func (e *Event) ValidateSpecific() error {
	switch e.GameID {
	case "racing":
		return e.validateRacingSpecific()
	case "combat":
		return e.validateCombatSpecific()
	}
	return nil
}

func (e *Event) validateRacingSpecific() error {
	switch e.Type {
	case EventTypeLapCompleted:
		if e.Data == nil {
			return errors.New("LAP_COMPLETED requiere campo data")
		}
		data, ok := e.Data.(map[string]interface{})
		if !ok {
			return errors.New("LAP_COMPLETED: data debe ser un objeto")
		}
		if _, ok := data["lapNumber"]; !ok {
			return errors.New("LAP_COMPLETED: data.lapNumber es obligatorio")
		}
		if _, ok := data["lapTimeMs"]; !ok {
			return errors.New("LAP_COMPLETED: data.lapTimeMs es obligatorio")
		}
		if _, ok := data["averageSpeed"]; !ok {
			return errors.New("LAP_COMPLETED: data.averageSpeed es obligatorio")
		}
		if _, ok := data["position"]; !ok {
			return errors.New("LAP_COMPLETED: data.position es obligatorio")
		}
		// Validar tipos y rangos
		if lapNum, ok := data["lapNumber"].(float64); ok {
			if lapNum < 1 {
				return errors.New("LAP_COMPLETED: lapNumber debe ser >= 1")
			}
		}
		if lapTime, ok := data["lapTimeMs"].(float64); ok {
			if lapTime <= 0 {
				return errors.New("LAP_COMPLETED: lapTimeMs debe ser > 0")
			}
		}
		if pos, ok := data["position"].(float64); ok {
			if pos < 1 {
				return errors.New("LAP_COMPLETED: position debe ser >= 1")
			}
		}
	case EventTypePenaltyReceived:
		if e.Data == nil {
			return errors.New("PENALTY_RECEIVED requiere campo data")
		}
	case EventTypePositionChanged:
		if e.Data == nil {
			return errors.New("POSITION_CHANGED requiere campo data")
		}
	case EventTypeMatchStarted:
		// No requiere campos específicos obligatorios en data
	case EventTypeMatchFinished:
		// No requiere campos específicos obligatorios en data
	case EventTypePlayerJoined:
		// No requiere campos específicos obligatorios en data
	}
	return nil
}

func (e *Event) validateCombatSpecific() error {
	switch e.Type {
	case EventTypePlayerAttacked:
		if e.Data == nil {
			return errors.New("PLAYER_ATTACKED requiere campo data")
		}
		data, ok := e.Data.(map[string]interface{})
		if !ok {
			return errors.New("PLAYER_ATTACKED: data debe ser un objeto")
		}
		if _, ok := data["attacker"]; !ok {
			return errors.New("PLAYER_ATTACKED: data.attacker es obligatorio")
		}
		if _, ok := data["target"]; !ok {
			return errors.New("PLAYER_ATTACKED: data.target es obligatorio")
		}
		if _, ok := data["damage"]; !ok {
			return errors.New("PLAYER_ATTACKED: data.damage es obligatorio")
		}
		if _, ok := data["weapon"]; !ok {
			return errors.New("PLAYER_ATTACKED: data.weapon es obligatorio")
		}
		if _, ok := data["distance"]; !ok {
			return errors.New("PLAYER_ATTACKED: data.distance es obligatorio")
		}
		if damage, ok := data["damage"].(float64); ok {
			if damage < 0 {
				return errors.New("PLAYER_ATTACKED: damage no puede ser negativo")
			}
		}
	case EventTypePlayerEliminated:
		if e.Data == nil {
			return errors.New("PLAYER_ELIMINATED requiere campo data")
		}
	case EventTypeMatchStarted:
		// No requiere campos específicos obligatorios en data
	case EventTypeMatchFinished:
		// No requiere campos específicos obligatorios en data
	case EventTypePlayerJoined:
		// No requiere campos específicos obligatorios en data
	}
	return nil
}

// MatchState representa el estado temporal de una partida
type MatchState struct {
	MatchID   string
	GameID    string
	PlayerMap map[string]*PlayerMatchState
	Events    []*Event
	Started   bool
	Finished  bool
	Analyzed  bool
	Analysis  *MatchAnalysis
	PlayerCount int
}

type PlayerMatchState struct {
	PlayerID    string
	Events      []*Event
	Positions   []int
	LapTimes    []int64
	Penalties   int
	Eliminations int
	TimesEliminated int
	TotalDamage float64
	Attacks     int
	BestLapTime int64
}

// NewMatchState crea un nuevo estado de partida
func NewMatchState(matchID, gameID string) *MatchState {
	return &MatchState{
		MatchID:   matchID,
		GameID:    gameID,
		PlayerMap: make(map[string]*PlayerMatchState),
		Events:    []*Event{},
	}
}

// AddEvent agrega un evento al estado de la partida
func (ms *MatchState) AddEvent(event *Event) error {
	// Verificar duplicado por eventID dentro de la partida
	for _, e := range ms.Events {
		if e.EventID == event.EventID {
			return errors.New("evento duplicado en la partida: " + event.EventID)
		}
	}
	ms.Events = append(ms.Events, event)
	
	// Actualizar estado del jugador
	if event.PlayerID != "" {
		if _, ok := ms.PlayerMap[event.PlayerID]; !ok {
			ms.PlayerMap[event.PlayerID] = &PlayerMatchState{
				PlayerID: event.PlayerID,
			}
		}
		ms.PlayerMap[event.PlayerID].Events = append(ms.PlayerMap[event.PlayerID].Events, event)
	}
	
	// Manejar eventos de inicio y fin
	switch event.Type {
	case EventTypeMatchStarted:
		ms.Started = true
	case EventTypeMatchFinished:
		ms.Finished = true
	}
	
	// Actualizar información específica por tipo
	switch event.Type {
	case EventTypeLapCompleted:
		if ps, ok := ms.PlayerMap[event.PlayerID]; ok {
			if data, ok := event.Data.(map[string]interface{}); ok {
				if lt, ok := data["lapTimeMs"].(float64); ok {
					ps.LapTimes = append(ps.LapTimes, int64(lt))
				}
				if pos, ok := data["position"].(float64); ok {
					ps.Positions = append(ps.Positions, int(pos))
				}
			}
		}
	case EventTypePenaltyReceived:
		if ps, ok := ms.PlayerMap[event.PlayerID]; ok {
			ps.Penalties++
		}
	case EventTypePlayerEliminated:
		if ps, ok := ms.PlayerMap[event.PlayerID]; ok {
			ps.TimesEliminated++
		}
	case EventTypePlayerAttacked:
		if ps, ok := ms.PlayerMap[event.PlayerID]; ok {
			ps.Attacks++
			if data, ok := event.Data.(map[string]interface{}); ok {
				if d, ok := data["damage"].(float64); ok {
					ps.TotalDamage += d
				}
			}
		}
	}
	
	return nil
}

// IsAnalyzed verifica si la partida ya fue analizada
func (ms *MatchState) IsAnalyzed() bool {
	return ms.Analyzed
}

// MarkAnalyzed marca la partida como analizada
func (ms *MatchState) MarkAnalyzed(a *MatchAnalysis) {
	ms.Analyzed = true
	ms.Analysis = a
}

// GetStatistics estadísticas de la partida
func (ms *MatchState) GetStatistics() map[string]interface{} {
	result := map[string]interface{}{
		"matchId":     ms.MatchID,
		"gameId":      ms.GameID,
		"eventsCount": len(ms.Events),
		"playerCount": len(ms.PlayerMap),
	}
	
	if ms.GameID == "racing" {
		result["racing"] = ms.racingStats()
	} else if ms.GameID == "combat" {
		result["combat"] = ms.combatStats()
	}
	
	return result
}

func (ms *MatchState) racingStats() map[string]interface{} {
	lapTimes := []int64{}
	positionEvolution := []int{}
	
	for _, ps := range ms.PlayerMap {
		lt := ms.extractLapTimes(ps)
		lapTimes = append(lapTimes, lt...)
		positionEvolution = append(positionEvolution, ps.Positions...)
	}
	
	bestLap := int64(0)
	if len(lapTimes) > 0 {
		bestLap = lapTimes[0]
		for _, lt := range lapTimes[1:] {
			if lt < bestLap {
				bestLap = lt
			}
		}
	}
	
	avgLap := float64(0)
	if len(lapTimes) > 0 {
		sum := int64(0)
		for _, lt := range lapTimes {
			sum += lt
		}
		avgLap = float64(sum) / float64(len(lapTimes))
	}
	
	totalPenalties := 0
	for _, ps := range ms.PlayerMap {
		totalPenalties += ps.Penalties
	}
	
	return map[string]interface{}{
		"lapsCompleted": len(lapTimes),
		"bestLapTime":   bestLap,
		"averageLapTime": avgLap,
		"penalties":     totalPenalties,
		"positionEvolution": positionEvolution,
	}
}

func (ms *MatchState) extractLapTimes(ps *PlayerMatchState) []int64 {
	result := []int64{}
	for _, lt := range ps.LapTimes {
		result = append(result, lt)
	}
	return result
}

func (ms *MatchState) combatStats() map[string]interface{} {
	totalAttacks := 0
	totalDamage := float64(0)
	totalEliminations := 0
	timesEliminated := 0
	
	for _, ps := range ms.PlayerMap {
		totalAttacks += ps.Attacks
		totalDamage += ps.TotalDamage
		totalEliminations += ps.Eliminations
		timesEliminated += ps.TimesEliminated
	}
	
	avgDamage := float64(0)
	if totalAttacks > 0 {
		avgDamage = totalDamage / float64(totalAttacks)
	}
	
	return map[string]interface{}{
		"totalAttacks":     totalAttacks,
		"totalDamage":      totalDamage,
		"averageDamage":   avgDamage,
		"totalEliminations": totalEliminations,
		"timesEliminated": timesEliminated,
	}
}

// PlayerStats estadísticas de un jugador
type PlayerStats struct {
	PlayerID string `json:"playerId"`
	
	// Racing stats
	RacesCount     int     `json:"racesCount"`
	TotalLaps      int     `json:"totalLaps"`
	BestPersonalLap float64 `json:"bestPersonalLap"`
	AverageLapTime float64 `json:"averageLapTime"`
	PenaltiesCount int     `json:"penaltiesCount"`
	RacesFinished  int     `json:"racesFinished"`
	
	// Combat stats
	Eliminations     int     `json:"eliminations"`
	TimesEliminated  int     `json:"timesEliminated"`
	DamageInflicted  float64 `json:"damageInflicted"`
	AverageDamage    float64 `json:"averageDamage"`
	KillDeathRatio   float64 `json:"killDeathRatio"`
}

// GetKillDeathRatio calcula la relación elimaciones/veces eliminado
func (ps *PlayerStats) GetKillDeathRatio() float64 {
	if ps.TimesEliminated == 0 {
		return float64(ps.Eliminations) // Return eliminaciones si nunca fue eliminado
	}
	return float64(ps.Eliminations) / float64(ps.TimesEliminated)
}

// MatchAnalysis resultado del análisis de una partida
type MatchAnalysis struct {
	MatchID    string                 `json:"matchId"`
	GameID     string                 `json:"gameId"`
	Statistics map[string]interface{} `json:"statistics"`
	Patterns   []*PatternResult       `json:"patterns"`
}

// PatternResult resultado de detección de patrones
type PatternResult struct {
	Type      string `json:"type"`
	PlayerID  string `json:"playerId,omitempty"`
	Detected  bool   `json:"detected"`
	Detail    string `json:"detail,omitempty"`
}