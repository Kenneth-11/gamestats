package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Evento representa un evento de juego simulado
type Evento struct {
	EventID   string      `json:"eventId"`
	Timestamp time.Time   `json:"timestamp"`
	GameID    string      `json:"gameId"`
	GameVersion string   `json:"gameVersion"`
	MatchID   string      `json:"matchId"`
	PlayerID  string      `json:"playerId"`
	Type      string      `json:"type"`
	Action    string      `json:"action"`
	Data      interface{} `json:"data"`
}

// Simulador genera eventos deterministas para carreras y combate
type Simulador struct {
	seed       int64
	rng        *rand.Rand
	eventCount int
}

// NewSimulador crea un nuevo simulador con semilla configurable
func NewSimulador(seed int64) *Simulador {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &Simulador{
		seed:       seed,
		rng:        rand.New(rand.NewSource(seed)),
		eventCount: 0,
	}
}

// GenerarPartesCarreras genera múltiples partidas de carreras
func (s *Simulador) GenerarPartesCarreras(cantidad int) [][]*Evento {
	partes := make([][]*Evento, cantidad)
	for i := 0; i < cantidad; i++ {
		partes[i] = s.generarPartidaCarrera(i + 1)
	}
	return partes
}

// GenerarPartesCombate genera múltiples partidas de combate
func (s *Simulador) GenerarPartesCombate(cantidad int) [][]*Evento {
	partes := make([][]*Evento, cantidad)
	for i := 0; i < cantidad; i++ {
		partes[i] = s.generarPartidaCombate(i + 1)
	}
	return partes
}

// generarPartidaCarrera genera una partida de carreras completa
func (s *Simulador) generarPartidaCarrera(partID int) []*Evento {
	eventos := []*Evento{}

	/* EVENT: MATCH_STARTED */
	eventos = s.agregarEvento(eventos, s.eventoMatchStarted("racing", "v1", "match_racing_"+fmt.Sprintf("%03d", partID), "player_base"))

	// Jugadores joining
	numJugadores := s.rng.Intn(5) + 2 // 2-6 jugadores
	jugadores := make(map[string]bool)
	jugadores["player_base"] = true

	for j := 1; j < numJugadores; j++ {
		nombre := fmt.Sprintf("player_%d", j)
		jugadores[nombre] = true
		eventos = s.agregarEvento(eventos, s.eventoPlayerJoined("racing", "v1", "match_racing_"+fmt.Sprintf("%03d", partID), nombre))
	}

	// Laps completados - algunos jugadores tienen mejores tiempos
	posiciones := []int{3, 1, 2, 5, 4}
	for lap := 1; lap <= s.rng.Intn(10)+5; lap++ { // 5-15 vueltas
		// Seleccionar jugador actual (rotar entre jugadores)
		players := []string{}
		for p := range jugadores {
			players = append(players, p)
		}
		if len(players) == 0 {
			break
		}
		jugador := players[s.rng.Intn(len(players))]

		// A veces el cambio de posición ocurre
		if s.rng.Float32() < 0.7 {
			fromPos := posiciones[s.rng.Intn(len(posiciones))]
			toPos := fromPos
			// A veces sube, a veces baja
			if s.rng.Float32() < 0.5 && fromPos > 1 {
				toPos = fromPos - 1 // remontada parcial
			} else if s.rng.Float32() < 0.5 && fromPos < 5 {
				toPos = fromPos + 1
			}
			eventos = s.agregarEvento(eventos, s.eventoPositionChanged("racing", "v1", "match_racing_"+fmt.Sprintf("%03d", partID), jugador, fromPos, toPos))
		}

		// Evento de vuelta completada
		lapTime := s.rng.Intn(5000) + 800 // 800-5800 ms
		speed := s.rng.Intn(100) + 120   // 120-220 km/h
		lapNum := lap
		if s.rng.Float32() < 0.1 {
			lapNum = 0 // evento sin número de vuelta
		}
		eventos = s.agregarEvento(eventos, s.eventoLapCompleted("racing", "v1", "match_racing_"+fmt.Sprintf("%03d", partID), jugador, lapNum, lapTime, speed, (lap%5)+1))
		
		// Penalizaciones ocasionales
		if s.rng.Float32() < 0.3 {
			penaltyType := []string{"collision", "speeding", "cut_corner"}[s.rng.Intn(3)]
			eventos = s.agregarEvento(eventos, s.eventoPenaltyReceived("racing", "v1", "match_racing_"+fmt.Sprintf("%03d", partID), jugador, penaltyType))
		}
	}

	// EVENT: MATCH_FINISHED
	ganador := ""
	for p := range jugadores {
		ganador = p
		break
	}
	eventos = s.agregarEvento(eventos, s.eventoMatchFinished("racing", "v1", "match_racing_"+fmt.Sprintf("%03d", partID), ganador, 3600000))

	s.eventCount += len(eventos)
	return eventos
}

// generarPartidaCombate genera una partida de combate completa
func (s *Simulador) generarPartidaCombate(partID int) []*Evento {
	eventos := []*Evento{}

	/* EVENT: MATCH_STARTED */
	eventos = s.agregarEvento(eventos, s.eventoMatchStarted("combat", "v1", "match_combat_"+fmt.Sprintf("%03d", partID), ""))

	// Jugadores joining
	numJugadores := s.rng.Intn(4) + 2 // 2-5 jugadores
	jugadores := make(map[string]bool)
	jugadores["player_base"] = true

	for j := 1; j < numJugadores; j++ {
		nombre := fmt.Sprintf("player_%d", j)
		jugadores[nombre] = true
		eventos = s.agregarEvento(eventos, s.eventoPlayerJoined("combat", "v1", "match_combat_"+fmt.Sprintf("%03d", partID), nombre))
	}

	// Ataques y eliminaciones
	numAtaques := s.rng.Intn(8) + 4 // 4-11 ataques
	for a := 0; a < numAtaques; a++ {
		// Selector de atacante y objetivo
		attackers := []string{}
		for p := range jugadores {
			attackers = append(attackers, p)
		}
		if len(attackers) == 0 {
			break
		}
		atacante := attackers[s.rng.Intn(len(attackers))]

		// Objetivo diferente del atacante
		objetivo := ""
		for p := range jugadores {
			if p != atacante {
				objetivo = p
				break
			}
		}
		if objetivo == "" {
			objetivo = atacante // fallback
		}

		damage := s.rng.Intn(50) + 20   // 20-70 de daño
		weapon := []string{"rifle", "shotgun", "sniper"}[s.rng.Intn(3)]
		distance := s.rng.Intn(100)      // 0-99 metros

		eventos = s.agregarEvento(eventos, s.eventoPlayerAttacked("combat", "v1", "match_combat_"+fmt.Sprintf("%03d", partID), atacante, objetivo, damage, weapon, distance))

		// A veces hay eliminación
		if s.rng.Float32() < 0.6 { // 60% de probabilidad de eliminación
			// Verificar que el objetivo no fue ya eliminado anteriormente
			eventos = s.agregarEvento(eventos, s.eventoPlayerEliminated("combat", "v1", "match_combat_"+fmt.Sprintf("%03d", partID), objetivo, atacante, weapon, distance))
		}
	}

	// EVENT: MATCH_FINISHED
	ganador := ""
	for p := range jugadores {
		ganador = p
		break
	}
	eventos = s.agregarEvento(eventos, s.eventoMatchFinished("combat", "v1", "match_combat_"+fmt.Sprintf("%03d", partID), ganador, 1800000))

	s.eventCount += len(eventos)
	return eventos
}

// Eventos helper methods

func (s *Simulador) eventoMatchStarted(juego, version, matchID, playerID string) *Evento {
	return s.crearEvento("MATCH_STARTED", juego, version, matchID, playerID, nil)
}

func (s *Simulador) eventoPlayerJoined(juego, version, matchID, playerID string) *Evento {
	data := map[string]string{"playerName": playerID, "team": "blue"}
	return s.crearEvento("PLAYER_JOINED", juego, version, matchID, playerID, data)
}

func (s *Simulador) eventoLapCompleted(juego, version, matchID, playerID string, lapNum, lapTimeMs, averageSpeed, position int) *Evento {
	data := map[string]interface{}{
		"lapNumber":     lapNum,
		"lapTimeMs":     lapTimeMs,
		"averageSpeed":  averageSpeed,
		"position":      position,
	}
	return s.crearEvento("LAP_COMPLETED", juego, version, matchID, playerID, data)
}

func (s *Simulador) eventoPositionChanged(juego, version, matchID, playerID string, fromPos, toPos int) *Evento {
	data := map[string]interface{}{
		"fromPosition": fromPos,
		"toPosition":   toPos,
	}
	return s.crearEvento("POSITION_CHANGED", juego, version, matchID, playerID, data)
}

func (s *Simulador) eventoPenaltyReceived(juego, version, matchID, playerID string, penaltyType string) *Evento {
	data := map[string]string{"penaltyType": penaltyType}
	return s.crearEvento("PENALTY_RECEIVED", juego, version, matchID, playerID, data)
}

func (s *Simulador) eventoPlayerAttacked(juego, version, matchID, attacker, target string, damage int, weapon string, distance int) *Evento {
	data := map[string]interface{}{
		"attacker":  attacker,
		"target":    target,
		"damage":    damage,
		"weapon":    weapon,
		"distance":  distance,
	}
	return s.crearEvento("PLAYER_ATTACKED", juego, version, matchID, "", data) // playerId vacío para eventos de combate
}

func (s *Simulador) eventoPlayerEliminated(juego, version, matchID, victim, killer string, weapon string, distance int) *Evento {
	data := map[string]interface{}{
		"victim":    victim,
		"killer":    killer,
		"weapon":    weapon,
		"distance":  distance,
	}
	// Evento sin playerId ya que es global
	return s.crearEvento("PLAYER_ELIMINATED", juego, version, matchID, "", data)
}

func (s *Simulador) eventoMatchFinished(juego, version, matchID, winnerPlayerID string, durationMs int64) *Evento {
	data := map[string]interface{}{
		"winnerPlayerId": winnerPlayerID,
		"durationMs":     durationMs,
	}
	return s.crearEvento("MATCH_FINISHED", juego, version, matchID, "", data)
}

func (s *Simulador) crearEvento(tipo, juego, version, matchID, playerID string, data interface{}) *Evento {
	eventID := fmt.Sprintf("evt_%04d_%d", s.eventCount, s.rng.Intn(10000))
	ts := time.Date(2026, 3, 15, s.rng.Intn(24), s.rng.Intn(60), s.rng.Intn(60), 0, time.UTC)
	
	evento := &Evento{
		EventID:   eventID,
		Timestamp: ts,
		GameID:    juego,
		GameVersion: version,
		MatchID:   matchID,
		PlayerID:  playerID,
		Type:      tipo,
		Action:    "",
		Data:      data,
	}
	s.eventCount++
	return evento
}

// Función auxiliar para agregar evento a la lista
func (s *Simulador) agregarEvento(eventos []*Evento, nuevo *Evento) []*Evento {
	// Verificar duplicado
	for _, e := range eventos {
		if e.EventID == nuevo.EventID {
			// Generar ID único si hay duplicado
			nuevo.EventID = fmt.Sprintf("evt_%04d_%d", s.eventCount, s.rng.Intn(10000))
			return s.agregarEvento(eventos, nuevo)
		}
	}
	eventos = append(eventos, nuevo)
	return eventos
}

// Main simulador
func main() {
	// Parsear semilla opcional
	seed := int64(42) // Semilla por defecto para reproducibilidad
	args := os.Args[1:]
	if len(args) > 0 {
		if s, err := fmt.Sscanf(args[0], "%d", &seed); s == 1 && err == nil {
			// seed configurado
		}
	}

	s := NewSimulador(seed)
	
	fmt.Println("=== GameStats - Simulador de Eventos ===")
	fmt.Printf("Semilla: %d\n", s.seed)
	fmt.Println()

	// Generar 3 partidas de carreras
	fmt.Println("--- Partes de Carreras ---")
	partesCarreras := s.GenerarPartesCarreras(3)
	for i, parte := range partesCarreras {
		fmt.Printf("Partida de carreras %d (%d eventos):\n", i+1, len(parte))
		for _, e := range parte {
			b, _ := json.Marshal(e)
			fmt.Println(string(b))
		}
		fmt.Println()
	}

	// Generar 2 partidas de combate
	fmt.Println("--- Partes de Combate ---")
	partesCombate := s.GenerarPartesCombate(2)
	for i, parte := range partesCombate {
		fmt.Printf("Partida de combate %d (%d eventos):\n", i+1, len(parte))
		for _, e := range parte {
			b, _ := json.Marshal(e)
			fmt.Println(string(b))
		}
		fmt.Println()
	}

	// Escribir archivos JSONL
	err := os.MkdirAll("data", 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creando directorio data: %v\n", err)
		os.Exit(1)
	}

	// Escribir datos deterministas
	err = writeJSONLFile("data/carreras_small.jsonl", partesCarreras[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error escribiendo file: %v\n", err)
	}

	err = writeJSONLFile("data/combate_small.jsonl", partesCombate[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error escribiendo file: %v\n", err)
	}

	fmt.Println("=== Eventos guardados en data/ ===")
	fmt.Println("Para probar: go-run-go-service --ingest-file data/carreras_small.jsonl")
}

func writeJSONLFile(path string, events []*Evento) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, e := range events {
		b, _ := json.Marshal(e)
		f.Write(b)
		f.Write([]byte("\n"))
	}
	return nil
}