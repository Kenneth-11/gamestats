package domain

import (
	"testing"
	"time"
)

func mkEvent(id, match, player string, typ EventType, data interface{}) *Event {
	return &Event{
		EventID:     id,
		Timestamp:   time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC),
		GameID:      "racing",
		GameVersion: "1.0.0",
		MatchID:     match,
		PlayerID:    player,
		Type:        typ,
		Data:        data,
	}
}

func lapData(lap, ms, pos int) map[string]interface{} {
	return map[string]interface{}{
		"lapNumber": float64(lap), "lapTimeMs": float64(ms),
		"averageSpeed": float64(150), "position": float64(pos),
	}
}

func TestAddEventDuplicado(t *testing.T) {
	ms := NewMatchState("m1", "racing")
	ev := mkEvent("e1", "m1", "p1", EventTypeLapCompleted, lapData(1, 1000, 2))
	if err := ms.AddEvent(ev); err != nil {
		t.Fatalf("primer evento fallo: %v", err)
	}
	if err := ms.AddEvent(ev); err == nil {
		t.Fatal("se esperaba error por evento duplicado")
	}
}

func TestMatchStartedFinished(t *testing.T) {
	ms := NewMatchState("m1", "racing")
	if err := ms.AddEvent(mkEvent("s", "m1", "", EventTypeMatchStarted, nil)); err != nil {
		t.Fatal(err)
	}
	if !ms.Started {
		t.Fatal("Started deberia ser true")
	}
	if err := ms.AddEvent(mkEvent("f", "m1", "", EventTypeMatchFinished, nil)); err != nil {
		t.Fatal(err)
	}
	if !ms.Finished {
		t.Fatal("Finished deberia ser true")
	}
}

func TestRacingStatsBestAvg(t *testing.T) {
	ms := NewMatchState("m1", "racing")
	_ = ms.AddEvent(mkEvent("e1", "m1", "p1", EventTypeLapCompleted, lapData(1, 1000, 2)))
	_ = ms.AddEvent(mkEvent("e2", "m1", "p1", EventTypeLapCompleted, lapData(2, 900, 1)))
	stats := ms.GetStatistics()
	racing, ok := stats["racing"].(map[string]interface{})
	if !ok {
		t.Fatalf("sin bloque racing: %v", stats)
	}
	if racing["lapsCompleted"] != 2 {
		t.Fatalf("lapsCompleted=%v, queria 2", racing["lapsCompleted"])
	}
	if racing["bestLapTime"] != int64(900) {
		t.Fatalf("bestLapTime=%v, queria 900", racing["bestLapTime"])
	}
	if racing["averageLapTime"] != 950.0 {
		t.Fatalf("averageLapTime=%v, queria 950", racing["averageLapTime"])
	}
}

func TestKillDeathRatio(t *testing.T) {
	ps := &PlayerStats{Eliminations: 3, TimesEliminated: 0}
	if ps.GetKillDeathRatio() != 3 {
		t.Fatalf("K/D sin muertes=%v, queria 3", ps.GetKillDeathRatio())
	}
	ps.TimesEliminated = 2
	if ps.GetKillDeathRatio() != 1.5 {
		t.Fatalf("K/D=%v, queria 1.5", ps.GetKillDeathRatio())
	}
}
