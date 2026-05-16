package models

import (
	"encoding/json"
	"time"
)

type Tasks struct {
	ID        string          `json:"id"`
	TargetURL string          `json:"target_url"`
	Payload   json.RawMessage `json:"payload"`
	RunAt     time.Time       `json:"run_at"`
	Status    string          `json:"status"`
	Priority  int             `json:"priority"`
}
