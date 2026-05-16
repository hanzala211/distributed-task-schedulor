package models

import (
	"encoding/json"
	"time"
)

type AddTaskAPIDTO struct {
	TargetURL    string          `json:"target_url" validate:"required,url,startswith=https://"`
	Payload      json.RawMessage `json:"payload" validate:"required"`
	RunAt        time.Time       `json:"run_at" validate:"required"`
	Priority     int             `json:"priority" validate:"gte=0,lte=1"`
	Dependencies []string        `json:"dependencies"`
}
