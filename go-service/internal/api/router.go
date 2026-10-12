package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"gamestats/go-service/internal/domain"
	"gamestats/go-service/internal/ingestion"
)

// Server representa el servidor HTTP de Go
type Server struct {
	ingestor *ingestion.Ingestor
	mux      *http.ServeMux
}

// NewServer crea un nuevo servidor HTTP
func NewServer(ingestor *ingestion.Ingestor) *Server {
	s := &Server{
		ingestor: ingestor,
		mux:      http.NewServeMux(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/events", s.handleEvents)
	s.mux.HandleFunc("/ingest/file", s.handleIngestFile)
	s.mux.HandleFunc("/matches/", s.handleMatch)
	s.mux.HandleFunc("/players/", s.handlePlayer)
	s.mux.HandleFunc("/stats", s.handleStats)
}

// Handler expone el mux
func (s *Server) Handler() http.Handler {
	return s.mux
}

// HandleHTTP alias por compatibilidad
func (s *Server) HandleHTTP() http.Handler {
	return s.mux
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "gamestats-go",
	})
}

// POST /events : un evento JSON
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "error leyendo cuerpo", http.StatusBadRequest)
		return
	}
	event, errs := ingestion.ValidateJSONRaw(body)
	if len(errs) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"status": "error",
			"errors": errs,
		})
		return
	}
	if err := s.ingestor.ProcessEvent(event); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{
		"status":  "accepted",
		"eventId": event.EventID,
	})
}

// POST /ingest/file : multipart (campo "file") o body con arreglo JSON / JSONL
func (s *Server) handleIngestFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var events []*domain.Event
	rejected := 0

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "campo 'file' requerido", http.StatusBadRequest)
			return
		}
		defer file.Close()
		body, err := io.ReadAll(io.LimitReader(file, 50<<20))
		if err != nil {
			http.Error(w, "error leyendo archivo", http.StatusBadRequest)
			return
		}
		events, rejected = ingestion.ParseBodyFlexible(body)
	} else {
		body, err := io.ReadAll(io.LimitReader(r.Body, 50<<20))
		if err != nil {
			http.Error(w, "error leyendo cuerpo", http.StatusBadRequest)
			return
		}
		events, rejected = ingestion.ParseBodyFlexible(body)
	}

	processed := 0
	for _, ev := range events {
		if err := s.ingestor.ProcessEvent(ev); err != nil {
			rejected++
			continue
		}
		processed++
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":    "accepted",
		"processed": processed,
		"rejected":  rejected,
	})
}

// GET /matches/{id} y GET /matches/{id}/analysis
func (s *Server) handleMatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/matches/")
	rest = strings.Trim(rest, "/")
	if rest == "" {
		http.Error(w, "matchId requerido", http.StatusBadRequest)
		return
	}
	if strings.HasSuffix(rest, "/analysis") {
		matchID := strings.TrimSuffix(rest, "/analysis")
		matchID = strings.Trim(matchID, "/")
		s.handleMatchAnalysis(w, r, matchID)
		return
	}
	// matchId no debe contener "/"
	if strings.Contains(rest, "/") {
		http.Error(w, "ruta inválida", http.StatusBadRequest)
		return
	}
	s.handleMatchGet(w, r, rest)
}

func (s *Server) handleMatchGet(w http.ResponseWriter, r *http.Request, matchID string) {
	match, exists := s.ingestor.GetMatch(matchID)
	if !exists {
		http.Error(w, "partida no encontrada", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"matchId":    matchID,
		"statistics": match.GetStatistics(),
	})
}

func (s *Server) handleMatchAnalysis(w http.ResponseWriter, r *http.Request, matchID string) {
	match, exists := s.ingestor.GetMatch(matchID)
	if !exists {
		http.Error(w, "partida no encontrada", http.StatusNotFound)
		return
	}
	if match == nil || !match.IsAnalyzed() || match.Analysis == nil {
		writeJSON(w, http.StatusAccepted, map[string]string{
			"status":  "pending_analysis",
			"message": "analysis in progress",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "analyzed",
		"matchId":    matchID,
		"gameId":     match.GameID,
		"statistics": match.Analysis.Statistics,
		"patterns":   match.Analysis.Patterns,
	})
}

// GET /players/{id}/stats
func (s *Server) handlePlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/players/")
	rest = strings.Trim(rest, "/")
	// aceptar /players/{id} y /players/{id}/stats
	playerID := strings.TrimSuffix(rest, "/stats")
	playerID = strings.Trim(playerID, "/")
	if playerID == "" || strings.Contains(playerID, "/") {
		http.Error(w, "playerId inválido", http.StatusBadRequest)
		return
	}

	matches := s.ingestor.GetAllMatches()
	stats := &domain.PlayerStats{PlayerID: playerID}
	found := false
	var lapSum int64
	var lapCount int64

	for _, m := range matches {
		ps, ok := m.PlayerMap[playerID]
		if !ok {
			continue
		}
		found = true
		switch m.GameID {
		case "racing":
			stats.RacesCount++
			stats.TotalLaps += len(ps.LapTimes)
			stats.PenaltiesCount += ps.Penalties
			for _, lt := range ps.LapTimes {
				lapSum += lt
				lapCount++
				if stats.BestPersonalLap == 0 || float64(lt) < stats.BestPersonalLap {
					stats.BestPersonalLap = float64(lt)
				}
			}
			if m.Finished {
				stats.RacesFinished++
			}
		case "combat":
			stats.TimesEliminated += ps.TimesEliminated
			stats.DamageInflicted += ps.TotalDamage
			// Eliminaciones como killer: contar PLAYER_ELIMINATED donde data.killer == playerID
			for _, ev := range ps.Events {
				_ = ev
			}
			for _, ev := range m.Events {
				if ev.Type == domain.EventTypePlayerEliminated && ev.Data != nil {
					if d, ok := ev.Data.(map[string]interface{}); ok {
						if k, _ := d["killer"].(string); k == playerID {
							stats.Eliminations++
						}
					}
				}
				if ev.Type == domain.EventTypePlayerAttacked && ev.PlayerID == playerID {
					stats.AverageDamage = 0 // se recalcula abajo
				}
			}
		}
	}

	// Ataques totales del jugador (para promedio de daño en combate)
	var attacks int
	for _, m := range matches {
		if m.GameID != "combat" {
			continue
		}
		if ps, ok := m.PlayerMap[playerID]; ok {
			attacks += ps.Attacks
		}
	}
	if attacks > 0 {
		stats.AverageDamage = stats.DamageInflicted / float64(attacks)
	}
	if lapCount > 0 {
		stats.AverageLapTime = float64(lapSum) / float64(lapCount)
	}
	stats.KillDeathRatio = stats.GetKillDeathRatio()

	if !found {
		http.Error(w, "jugador no encontrado", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "ok",
		"playerId":   playerID,
		"statistics": stats,
	})
}

// GET /stats
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		return
	}
	matches := s.ingestor.GetAllMatches()
	uniquePlayers := map[string]bool{}
	totalEvents := 0
	racing, combat := 0, 0
	for _, m := range matches {
		totalEvents += len(m.Events)
		for pid := range m.PlayerMap {
			uniquePlayers[pid] = true
		}
		switch m.GameID {
		case "racing":
			racing++
		case "combat":
			combat++
		}
	}
	processed, rejected := s.ingestor.Stats()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"totalMatches":   len(matches),
		"totalPlayers":   len(uniquePlayers),
		"totalEvents":    totalEvents,
		"racingMatches":  racing,
		"combatMatches":  combat,
		"processed":      processed,
		"rejectedEvents": rejected,
	})
}
