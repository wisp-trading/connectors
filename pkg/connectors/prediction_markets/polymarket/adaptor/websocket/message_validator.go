package websocket

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/wisp-trading/connectors/pkg/websocket/security"
)

// ValidationConfig configures the Polymarket message validator.
type ValidationConfig struct {
	MaxMessageSize int
	AllowedEvents  map[string]bool
	RequiredFields map[string][]string
}

// DefaultValidationConfig returns the default Polymarket validator configuration.
// Supports all Polymarket WebSocket message types from the Market Channel.
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		MaxMessageSize: 2 * 1024 * 1024, // 2MB (for large new_market/market_resolved messages)
		AllowedEvents: map[string]bool{
			"book":             true,
			"price_change":     true,
			"tick_size_change": true,
			"last_trade_price": true,
			"best_bid_ask":     true,
			"new_market":       true,
			"market_resolved":  true,
		},
		RequiredFields: map[string][]string{
			"book":             {"event_type", "asset_id", "market", "timestamp"},
			"price_change":     {"event_type", "market", "timestamp"},
			"tick_size_change": {"event_type", "asset_id", "market", "timestamp"},
			"last_trade_price": {"event_type", "asset_id", "market", "timestamp"},
			"best_bid_ask":     {"event_type", "asset_id", "market", "timestamp"},
			"new_market":       {"event_type", "market", "timestamp"},
			"market_resolved":  {"event_type", "market", "timestamp"},
		},
	}
}

type messageValidator struct {
	config ValidationConfig
}

// NewMessageValidator creates a new Polymarket-specific message validator.
// Polymarket sends messages as JSON arrays, unlike standard WebSocket APIs.
func NewMessageValidator(config ValidationConfig) security.MessageValidator {
	return &messageValidator{config: config}
}

// ValidateMessage validates a Polymarket WebSocket message.
// Polymarket sends three formats:
// 1. Raw text (PONG keepalive)
// 2. Control messages: {"type": "subscribed"}
// 3. Market data: [{"event_type": "book", ...}]
func (mv *messageValidator) ValidateMessage(message []byte) error {
	// Size validation
	if len(message) > mv.config.MaxMessageSize {
		return fmt.Errorf("message too large: %d bytes (max: %d)",
			len(message), mv.config.MaxMessageSize)
	}

	trimmed := bytes.TrimSpace(message)
	if len(trimmed) == 0 {
		return fmt.Errorf("empty message")
	}

	// Handle raw text messages (PONG keepalive)
	trimmedUpper := bytes.ToUpper(trimmed)
	if bytes.Equal(trimmedUpper, []byte("PONG")) {
		return nil // PONG is always valid
	}

	// Try to parse as JSON first
	if !json.Valid(trimmed) {
		// Not JSON and not PONG - log the actual content for debugging
		return fmt.Errorf("invalid message format: not JSON and not PONG (starts with: %q)",
			string(trimmed[:min(len(trimmed), 50)]))
	}

	// Handle control messages (JSON objects)
	if trimmed[0] == '{' {
		var controlMsg map[string]interface{}
		if err := json.Unmarshal(trimmed, &controlMsg); err != nil {
			return fmt.Errorf("invalid JSON object: %w", err)
		}
		// Control messages are always valid
		return nil
	}

	// Handle market data (JSON arrays)
	if trimmed[0] == '[' {
		var messages []map[string]interface{}
		if err := json.Unmarshal(trimmed, &messages); err != nil {
			return fmt.Errorf("invalid JSON array: %w", err)
		}

		if len(messages) == 0 {
			return fmt.Errorf("empty message array")
		}

		// Validate each message in the array
		for i, msg := range messages {
			if err := mv.validateSingleMessage(msg, i); err != nil {
				return err
			}
		}
		return nil
	}

	return fmt.Errorf("invalid message format: expected JSON object or array")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// validateSingleMessage validates a single Polymarket message from the array.
func (mv *messageValidator) validateSingleMessage(msg map[string]interface{}, index int) error {
	// Extract event_type (required for all messages)
	eventType, ok := msg["event_type"].(string)
	if !ok || eventType == "" {
		return fmt.Errorf("message[%d]: missing or invalid event_type field", index)
	}

	// Check if event type is allowed
	if !mv.config.AllowedEvents[eventType] {
		return fmt.Errorf("message[%d]: invalid event_type: %s", index, eventType)
	}

	// Validate required fields based on event type
	if requiredFields, exists := mv.config.RequiredFields[eventType]; exists {
		for _, field := range requiredFields {
			if err := mv.validateRequiredField(msg, field, index); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateRequiredField checks if a required field exists and is non-empty.
func (mv *messageValidator) validateRequiredField(msg map[string]interface{}, field string, index int) error {
	value, exists := msg[field]
	if !exists {
		return fmt.Errorf("message[%d]: missing required field '%s'", index, field)
	}

	// Check if value is non-empty (for string fields)
	switch v := value.(type) {
	case string:
		if v == "" {
			return fmt.Errorf("message[%d]: empty value for required field '%s'", index, field)
		}
	case nil:
		return fmt.Errorf("message[%d]: null value for required field '%s'", index, field)
	}

	return nil
}
