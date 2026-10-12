package ingestion

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gamestats/go-service/internal/domain"
	"gamestats/go-service/internal/validation"
)

// JSONLReader lee eventos desde un archivo JSONL
type JSONLReader struct {
	validator *validation.Validator
}

// NewJSONLReader crea un nuevo lector JSONL
func NewJSONLReader() *JSONLReader {
	return &JSONLReader{
		validator: validation.NewValidator(),
	}
}

// ValidateJSONRaw valida un JSON crudo (un solo evento)
func ValidateJSONRaw(raw []byte) (*domain.Event, []*validation.ValidationError) {
	return validation.ValidateJSON(raw)
}

// ValidateJSONLLine valida una línea individual JSONL
func (r *JSONLReader) ValidateJSONLLine(line string) (*domain.Event, []*validation.ValidationError) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}
	return validation.ValidateJSON([]byte(line))
}

// ReadEvents lee eventos de un archivo JSONL
func (r *JSONLReader) ReadEvents(ctx context.Context, filePath string) ([]*domain.Event, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("abriendo archivo: %w", err)
	}
	defer file.Close()
	return r.ReadFromReader(ctx, file)
}

// ReadFromReader lee eventos desde cualquier io.Reader (útil para tests y HTTP)
func (r *JSONLReader) ReadFromReader(ctx context.Context, reader io.Reader) ([]*domain.Event, int, error) {
	events := []*domain.Event{}
	rejected := 0
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return events, rejected, ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		event, errs := r.ValidateJSONLLine(line)
		if len(errs) > 0 || event == nil {
			rejected++
			continue
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return events, rejected, fmt.Errorf("leyendo contenido: %w", err)
	}
	return events, rejected, nil
}

// ParseBodyFlexible acepta: arreglo JSON [...] o JSONL (una o varias líneas)
func ParseBodyFlexible(body []byte) ([]*domain.Event, int) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, 0
	}
	// Intento 1: arreglo JSON
	if trimmed[0] == '[' {
		var events []*domain.Event
		if err := json.Unmarshal(trimmed, &events); err == nil {
			v := validation.NewValidator()
			valid := []*domain.Event{}
			rejected := 0
			for _, e := range events {
				if e == nil || len(v.ValidateEvent(e)) > 0 {
					rejected++
					continue
				}
				valid = append(valid, e)
			}
			return valid, rejected
		}
	}
	// Intento 2: un solo objeto JSON
	if trimmed[0] == '{' && json.Valid(trimmed) && !bytes.Contains(trimmed, []byte("\n")) {
		ev, errs := validation.ValidateJSON(trimmed)
		if len(errs) > 0 {
			return nil, 1
		}
		return []*domain.Event{ev}, 0
	}
	// Intento 3: JSONL línea por línea
	r := NewJSONLReader()
	events, rejected, _ := r.ReadFromReader(context.Background(), bytes.NewReader(trimmed))
	return events, rejected
}

// Ingestor procesa eventos entrantes con workers concurrentes.
// Los eventos se rutean por matchId (hash) al mismo worker para preservar
// el orden dentro de cada partida; partidas distintas van en paralelo.
type Ingestor struct {
	validator       *validation.Validator
	matchStore      map[string]*domain.MatchState
	mutex           sync.RWMutex
	analyticsURL    string
	client          *AnalyticsClient
	workers         []chan *domain.Event
	wg              sync.WaitGroup
	processedEvents map[string]bool
	procLock        sync.Mutex
	totalProcessed  atomic.Int64
	totalRejected   atomic.Int64
}

// NewIngestor crea un nuevo ingestor. analyticsPort se conserva por compatibilidad.
func NewIngestor(analyticsPort, analyticsURL string) *Ingestor {
	// Si solo dan puerto, construir URL local
	if analyticsURL == "" && analyticsPort != "" {
		analyticsURL = "http://localhost:" + analyticsPort
	}
	i := &Ingestor{
		validator:       validation.NewValidator(),
		matchStore:      make(map[string]*domain.MatchState),
		analyticsURL:    strings.TrimSuffix(analyticsURL, "/"),
		processedEvents: make(map[string]bool),
	}
	i.client = NewAnalyticsClient(i.analyticsURL)
	return i
}

// route elige el worker por hash del matchId (orden por partida, paralelo entre partidas)
func (i *Ingestor) route(matchID string) chan *domain.Event {
	h := fnv.New32a()
	_, _ = h.Write([]byte(matchID))
	return i.workers[int(h.Sum32()%uint32(len(i.workers)))]
}

// Start inicia los workers
func (i *Ingestor) Start(workerCount int) {
	if workerCount < 1 {
		workerCount = 1
	}
	i.workers = make([]chan *domain.Event, workerCount)
	for w := 0; w < workerCount; w++ {
		ch := make(chan *domain.Event, 1000)
		i.workers[w] = ch
		i.wg.Add(1)
		go i.worker(ch)
	}
}

// Stop detiene el ingestor
func (i *Ingestor) Stop() {
	for _, ch := range i.workers {
		close(ch)
	}
	i.wg.Wait()
}

// Stats contadores globales
func (i *Ingestor) Stats() (processed, rejected int64) {
	return i.totalProcessed.Load(), i.totalRejected.Load()
}

// ProcessEvent valida duplicado global y encola para workers.
// Es síncrono en validación básica para poder responder 422 rápido,
// el resto lo hace el worker.
func (i *Ingestor) ProcessEvent(event *domain.Event) error {
	if event == nil {
		i.totalRejected.Add(1)
		return fmt.Errorf("evento nulo")
	}
	if errs := i.validator.ValidateEvent(event); len(errs) > 0 {
		i.totalRejected.Add(1)
		return fmt.Errorf("validación: %s", errs[0].Error())
	}

	i.procLock.Lock()
	if i.processedEvents[event.EventID] {
		i.procLock.Unlock()
		i.totalRejected.Add(1)
		return fmt.Errorf("evento duplicado: %s", event.EventID)
	}
	i.processedEvents[event.EventID] = true
	i.procLock.Unlock()

	select {
	case i.route(event.MatchID) <- event:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout encolando evento")
	}
}

// ProcessEventSync procesa inmediatamente (usado en ingesta masiva/tests)
func (i *Ingestor) processSingleEvent(event *domain.Event) error {
	if errs := i.validator.ValidateEvent(event); len(errs) > 0 {
		i.totalRejected.Add(1)
		return fmt.Errorf("evento inválido")
	}

	i.mutex.RLock()
	match, exists := i.matchStore[event.MatchID]
	i.mutex.RUnlock()

	if !exists {
		// Si el juego difiere del registrado, se respeta el primero;
		// eventos de otro gameId con mismo matchId se rechazan.
		match = domain.NewMatchState(event.MatchID, event.GameID)
		i.mutex.Lock()
		// doble check
		if existing, ok := i.matchStore[event.MatchID]; ok {
			match = existing
		} else {
			i.matchStore[event.MatchID] = match
		}
		i.mutex.Unlock()
	}

	if match.GameID != event.GameID {
		i.totalRejected.Add(1)
		return fmt.Errorf("gameId %s no coincide con partida %s (%s)", event.GameID, event.MatchID, match.GameID)
	}

	// Proteger el estado de la partida (varios workers pueden tocar la misma)
	i.mutex.Lock()
	err := match.AddEvent(event)
	finished := match.Finished && !match.IsAnalyzed()
	i.mutex.Unlock()

	if err != nil {
		i.totalRejected.Add(1)
		return err
	}
	i.totalProcessed.Add(1)

	if finished {
		go i.sendForAnalysis(match.MatchID)
	}
	return nil
}

func (i *Ingestor) worker(ch chan *domain.Event) {
	defer i.wg.Done()
	for event := range ch {
		_ = i.processSingleEvent(event)
	}
}

// sendForAnalysis toma snapshot de eventos y lo envía a Scala
func (i *Ingestor) sendForAnalysis(matchID string) {
	i.mutex.RLock()
	match, ok := i.matchStore[matchID]
	if !ok {
		i.mutex.RUnlock()
		return
	}
	// snapshot bajo lock de lectura
	eventsData := make([]map[string]interface{}, 0, len(match.Events))
	for _, e := range match.Events {
		eventsData = append(eventsData, map[string]interface{}{
			"eventId":     e.EventID,
			"timestamp":   e.Timestamp.Format(time.RFC3339),
			"gameId":      e.GameID,
			"gameVersion": e.GameVersion,
			"matchId":     e.MatchID,
			"playerId":    e.PlayerID,
			"type":        string(e.Type),
			"action":      e.Action,
			"data":        e.Data,
		})
	}
	gameID := match.GameID
	i.mutex.RUnlock()

	if i.analyticsURL == "" {
		return
	}

	request := map[string]interface{}{
		"matchId": matchID,
		"gameId":  gameID,
		"events":  eventsData,
	}

	result, err := i.client.AnalyzeWithRetry(request)
	if err != nil {
		return // Scala no disponible: queda pendiente, reintentará al consultar
	}

	// Guardar análisis devuelto por Scala (si trae formato esperado)
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if m, ok := i.matchStore[matchID]; ok && !m.IsAnalyzed() {
		analysis := &domain.MatchAnalysis{
			MatchID: matchID,
			GameID:  gameID,
		}
		if stats, ok := result["statistics"].(map[string]interface{}); ok {
			analysis.Statistics = stats
		} else {
			analysis.Statistics = result
		}
		if pats, ok := result["patterns"].([]interface{}); ok {
			for _, p := range pats {
				if pm, ok := p.(map[string]interface{}); ok {
					pr := &domain.PatternResult{}
					if t, ok := pm["type"].(string); ok {
						pr.Type = t
					}
					if pid, ok := pm["playerId"].(string); ok {
						pr.PlayerID = pid
					}
					if d, ok := pm["detected"].(bool); ok {
						pr.Detected = d
					}
					if detail, ok := pm["detail"].(string); ok {
						pr.Detail = detail
					}
					analysis.Patterns = append(analysis.Patterns, pr)
				}
			}
		}
		m.MarkAnalyzed(analysis)
	}
}

// GetMatch recupera el estado de una partida
func (i *Ingestor) GetMatch(matchID string) (*domain.MatchState, bool) {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	match, exists := i.matchStore[matchID]
	return match, exists
}

// GetAllMatches recupera todos los estados
func (i *Ingestor) GetAllMatches() map[string]*domain.MatchState {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	result := make(map[string]*domain.MatchState, len(i.matchStore))
	for k, v := range i.matchStore {
		result[k] = v
	}
	return result
}

// AnalyticsClient cliente HTTP hacia Scala
type AnalyticsClient struct {
	baseURL    string
	client     *http.Client
	retryCount int
}

// NewAnalyticsClient crea un nuevo cliente
func NewAnalyticsClient(baseURL string) *AnalyticsClient {
	return &AnalyticsClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		client:     &http.Client{Timeout: 10 * time.Second},
		retryCount: 3,
	}
}

// Analyze envía una partida para análisis y devuelve el JSON de Scala
func (c *AnalyticsClient) Analyze(request map[string]interface{}) (map[string]interface{}, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("SCALA_SERVICE_URL no configurada")
	}
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/analyze", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error conectando con Scala: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("scala respondió %d: %s", resp.StatusCode, string(respBody))
	}
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("respuesta Scala no es JSON: %w", err)
	}
	return result, nil
}

// AnalyzeWithRetry intenta con backoff exponencial 1s, 2s, 4s
func (c *AnalyticsClient) AnalyzeWithRetry(request map[string]interface{}) (map[string]interface{}, error) {
	var lastErr error
	for attempt := 1; attempt <= c.retryCount; attempt++ {
		result, err := c.Analyze(request)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if attempt < c.retryCount {
			time.Sleep(time.Duration(1<<(attempt-1)) * time.Second)
		}
	}
	return nil, fmt.Errorf("fallo tras %d intentos: %w", c.retryCount, lastErr)
}
