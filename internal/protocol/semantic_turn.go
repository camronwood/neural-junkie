package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/camronwood/neural-junkie/internal/intent"
)

const MetadataTurnDecision = "semantic_turn_decision"

// StampTurnDecision stores the canonical, versioned server decision on a turn.
func StampTurnDecision(msg *Message, decision intent.TurnDecision) error {
	if msg == nil {
		return fmt.Errorf("stamp turn decision: nil message")
	}
	intent.EnsureContextPlan(&decision, intent.TurnFeatures{})
	if err := decision.Validate(); err != nil {
		return err
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	encoded, err := json.Marshal(decision)
	if err != nil {
		return fmt.Errorf("marshal turn decision: %w", err)
	}
	var value map[string]interface{}
	if err := json.Unmarshal(encoded, &value); err != nil {
		return fmt.Errorf("normalize turn decision: %w", err)
	}
	msg.Metadata[MetadataTurnDecision] = value
	return nil
}

// ExtractTurnDecision validates a canonical decision from message metadata.
func ExtractTurnDecision(msg *Message) (intent.TurnDecision, bool) {
	if msg == nil || msg.Metadata == nil {
		return intent.TurnDecision{}, false
	}
	raw, ok := msg.Metadata[MetadataTurnDecision]
	if !ok {
		return intent.TurnDecision{}, false
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return intent.TurnDecision{}, false
	}
	var decision intent.TurnDecision
	if err := json.Unmarshal(encoded, &decision); err != nil {
		return intent.TurnDecision{}, false
	}
	if err := decision.Validate(); err != nil {
		return intent.TurnDecision{}, false
	}
	return decision, true
}
