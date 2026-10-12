package matches

import (
	"sync"

	"gamestats/go-service/internal/domain"
)

// MatchRepository interfaz para almacenar y recuperar estados de partida
type MatchRepository interface {
	GetMatch(matchID string) (*domain.MatchState, bool)
	GetAllMatches() map[string]*domain.MatchState
	SaveMatch(*domain.MatchState)
}

// InMemoryMatchRepository implementación en memoria del repositorio
type InMemoryMatchRepository struct {
	store map[string]*domain.MatchState
	mu    sync.RWMutex
}

// NewInMemoryMatchRepository crea un nuevo repositorio en memoria
func NewInMemoryMatchRepository() *InMemoryMatchRepository {
	return &InMemoryMatchRepository{
		store: make(map[string]*domain.MatchState),
	}
}

// GetMatch retrieve un estado de partida por ID
func (r *InMemoryMatchRepository) GetMatch(matchID string) (*domain.MatchState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	match, exists := r.store[matchID]
	return match, exists
}

// GetAllMatches retrieve todos los estados de partida
func (r *InMemoryMatchRepository) GetAllMatches() map[string]*domain.MatchState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]*domain.MatchState)
	for k, v := range r.store {
		result[k] = v
	}
	return result
}

// SaveMatch guarda un estado de partida
func (r *InMemoryMatchRepository) SaveMatch(ms *domain.MatchState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[ms.MatchID] = ms
}