package validation

import (
	"testing"
	"time"

	"gamestats/go-service/internal/domain"
)

func validRacingEvent() *domain.Event {
	return &domain.Event{
		EventID:     "evt-test-1",
		Timestamp:   time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC),
		GameID:      "racing",
		GameVersion: "1.0.0",
		MatchID:     "m1",
		PlayerID:    "p1",
		Type:        domain.EventTypeLapCompleted,
		Action:      "lap_done",
		Data: map[string]interface{}{
			"lapNumber": 1.0, "lapTimeMs": 1000.0, "averageSpeed": 150.0, "position": 2.0,
		},
	}
}

func TestValidateEventOK(t *testing.T) {
	v := NewValidator()
	if errs := v.ValidateEvent(validRacingEvent()); len(errs) != 0 {
		t.Fatalf("evento valido rechazado: %v", errs)
	}
}

func TestValidateEventCamposObligatorios(t *testing.T) {
	v := NewValidator()
	ev := validRacingEvent()
	ev.EventID = ""
	if errs := v.ValidateEvent(ev); len(errs) == 0 {
		t.Fatal("se esperaba error por eventId vacio")
	}
	ev = validRacingEvent()
	ev.GameID = "ajedrez"
	if errs := v.ValidateEvent(ev); len(errs) == 0 {
		t.Fatal("se esperaba error por gameId invalido")
	}
}

func TestValidateLapSinData(t *testing.T) {
	v := NewValidator()
	ev := validRacingEvent()
	ev.Data = nil
	if errs := v.ValidateEvent(ev); len(errs) == 0 {
		t.Fatal("se esperaba error por LAP_COMPLETED sin data")
	}
}

func TestValidateJSONMalformado(t *testing.T) {
	_, errs := ValidateJSON([]byte("{no json"))
	if len(errs) == 0 {
		t.Fatal("se esperaba error por JSON malformado")
	}
}

func TestValidateJSONCamposAusentes(t *testing.T) {
	_, errs := ValidateJSON([]byte(`{"eventId":"x"}`))
	if len(errs) == 0 {
		t.Fatal("se esperaba error por campos obligatorios ausentes")
	}
}
