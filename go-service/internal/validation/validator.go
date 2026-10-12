package validation

import (
	"encoding/json"

	"gamestats/go-service/internal/domain"
)

// ValidationError representa un error de validación
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implementa la interface error
func (ve *ValidationError) Error() string {
	return ve.Field + ": " + ve.Message
}

// Validator valida eventos de Go
type Validator struct{}

// NewValidator crea un nuevo validador
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateEvent valida un evento completo
func (v *Validator) ValidateEvent(event *domain.Event) []*ValidationError {
	errors := []*ValidationError{}

	// Validar estructura común
	if err := event.Validate(); err != nil {
		errors = append(errors, &ValidationError{
			Field:   "common",
			Message: err.Error(),
		})
	}

	// Validar campos específicos por tipo de juego
	if err := event.ValidateSpecific(); err != nil {
		errors = append(errors, &ValidationError{
			Field:   "specific",
			Message: err.Error(),
		})
	}

	return errors
}

// ValidateJSON valida JSON crudo y convierte a evento
func ValidateJSON(rawJSON []byte) (*domain.Event, []*ValidationError) {
	event := &domain.Event{}
	errors := []*ValidationError{}

	// Intentar deserializar
	if err := json.Unmarshal(rawJSON, event); err != nil {
		errors = append(errors, &ValidationError{
			Field:   "json",
			Message: "JSON malformado: " + err.Error(),
		})
		return event, errors
	}

	// Validar evento
	v := NewValidator()
	for _, e := range v.ValidateEvent(event) {
		errors = append(errors, e)
	}

	return event, errors
}

// ValidateEventString valida un evento desde string
func ValidateEventString(eventString string) (*domain.Event, []*ValidationError) {
	return ValidateJSON([]byte(eventString))
}
