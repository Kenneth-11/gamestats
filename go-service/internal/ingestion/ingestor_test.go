package ingestion

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"gamestats/go-service/internal/domain"
)

func mkLap(id, match, player string, lap, ms int) *domain.Event {
	return &domain.Event{
		EventID:     id,
		Timestamp:   time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC),
		GameID:      "racing",
		GameVersion: "1.0.0",
		MatchID:     match,
		PlayerID:    player,
		Type:        domain.EventTypeLapCompleted,
		Action:      "lap_done",
		Data: map[string]interface{}{
			"lapNumber": float64(lap), "lapTimeMs": float64(ms),
			"averageSpeed": float64(150), "position": float64(1),
		},
	}
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout: %s", msg)
}

func TestProcessEventAgrupaPorPartida(t *testing.T) {
	ing := NewIngestor("", "") // sin Scala: el analisis queda pendiente
	ing.Start(2)
	defer ing.Stop()

	if err := ing.ProcessEvent(mkLap("e1", "m1", "p1", 1, 1000)); err != nil {
		t.Fatal(err)
	}
	if err := ing.ProcessEvent(mkLap("e2", "m1", "p1", 2, 900)); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		m, ok := ing.GetMatch("m1")
		return ok && len(m.Events) == 2
	}, "eventos no agrupados en m1")
}

func TestProcessEventDuplicadoGlobal(t *testing.T) {
	ing := NewIngestor("", "")
	ing.Start(2)
	defer ing.Stop()

	if err := ing.ProcessEvent(mkLap("dup", "m1", "p1", 1, 1000)); err != nil {
		t.Fatal(err)
	}
	if err := ing.ProcessEvent(mkLap("dup", "m1", "p1", 1, 1000)); err == nil {
		t.Fatal("se esperaba error por duplicado global")
	}
}

func TestProcessEventInvalidoRechazado(t *testing.T) {
	ing := NewIngestor("", "")
	ing.Start(2)
	defer ing.Stop()

	bad := mkLap("bad", "", "p1", 1, 1000) // sin matchId
	if err := ing.ProcessEvent(bad); err == nil {
		t.Fatal("se esperaba error por evento invalido")
	}
}

func TestPartidasAisladas(t *testing.T) {
	ing := NewIngestor("", "")
	ing.Start(2)
	defer ing.Stop()

	_ = ing.ProcessEvent(mkLap("a1", "ma", "p1", 1, 1000))
	_ = ing.ProcessEvent(mkLap("b1", "mb", "p1", 1, 1100))
	waitFor(t, func() bool {
		ma, oka := ing.GetMatch("ma")
		mb, okb := ing.GetMatch("mb")
		return oka && okb && len(ma.Events) == 1 && len(mb.Events) == 1
	}, "partidas no aisladas")
}

func TestProcesamientoConcurrente(t *testing.T) {
	ing := NewIngestor("", "")
	ing.Start(4)
	defer ing.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = ing.ProcessEvent(mkLap(fmt.Sprintf("c-%d", i), "mc", "p1", i+1, 1000+i))
		}(i)
	}
	wg.Wait()
	waitFor(t, func() bool {
		m, ok := ing.GetMatch("mc")
		return ok && len(m.Events) == 20
	}, "no se procesaron los 20 eventos concurrentes")
}

func TestPartidaFinalizadaQuedaPendienteSinScala(t *testing.T) {
	ing := NewIngestor("", "") // sin Scala: no se puede analizar
	ing.Start(2)
	defer ing.Stop()

	start := &domain.Event{
		EventID: "st1", Timestamp: time.Now().UTC(), GameID: "racing",
		GameVersion: "1.0.0", MatchID: "mf", Type: domain.EventTypeMatchStarted,
	}
	fin := &domain.Event{
		EventID: "fn1", Timestamp: time.Now().UTC(), GameID: "racing",
		GameVersion: "1.0.0", MatchID: "mf", Type: domain.EventTypeMatchFinished,
	}
	if err := ing.ProcessEvent(start); err != nil {
		t.Fatal(err)
	}
	if err := ing.ProcessEvent(fin); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool {
		m, ok := ing.GetMatch("mf")
		return ok && m.Finished && !m.IsAnalyzed()
	}, "partida finalizada deberia quedar pendiente de analisis")
}

func TestAnalyticsClientReintentosFallan(t *testing.T) {
	c := NewAnalyticsClient("http://localhost:1") // puerto cerrado
	c.retryCount = 2
	start := time.Now()
	_, err := c.AnalyzeWithRetry(map[string]interface{}{"matchId": "x"})
	if err == nil {
		t.Fatal("se esperaba error contra Scala inexistente")
	}
	if time.Since(start) < time.Second {
		t.Fatal("se esperaba backoff entre reintentos")
	}
}

func TestParseBodyFlexible(t *testing.T) {	arr := `[{"eventId":"e1","timestamp":"2026-03-15T14:00:00Z","gameId":"racing","gameVersion":"1.0.0","matchId":"m1","playerId":"p1","type":"LAP_COMPLETED","action":"","data":{"lapNumber":1,"lapTimeMs":1000,"averageSpeed":150,"position":1}}]`
	evs, rej := ParseBodyFlexible([]byte(arr))
	if len(evs) != 1 || rej != 0 {
		t.Fatalf("arreglo JSON: evs=%d rej=%d", len(evs), rej)
	}
	single := `{"eventId":"e2","timestamp":"2026-03-15T14:00:00Z","gameId":"racing","gameVersion":"1.0.0","matchId":"m1","playerId":"p1","type":"PLAYER_JOINED","action":"","data":{}}`
	evs, rej = ParseBodyFlexible([]byte(single))
	if len(evs) != 1 || rej != 0 {
		t.Fatalf("objeto JSON: evs=%d rej=%d", len(evs), rej)
	}
}

func TestOrdenPorPartidaPreservado(t *testing.T) {
	ing := NewIngestor("", "")
	ing.Start(4) // varios workers: la misma partida debe mantener el orden
	defer ing.Stop()

	for i := 1; i <= 10; i++ {
		if err := ing.ProcessEvent(mkLap(fmt.Sprintf("o-%d", i), "mo", "p1", i, 1000+i)); err != nil {
			t.Fatal(err)
		}
	}
	waitFor(t, func() bool {
		m, ok := ing.GetMatch("mo")
		return ok && len(m.Events) == 10
	}, "eventos no llegaron")
	m, _ := ing.GetMatch("mo")
	for i, ev := range m.Events {
		want := fmt.Sprintf("o-%d", i+1)
		if ev.EventID != want {
			t.Fatalf("orden roto en pos %d: %s, queria %s", i, ev.EventID, want)
		}
	}
}
