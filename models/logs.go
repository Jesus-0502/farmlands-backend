package models

// Representación de eventos de logs
type Log struct {
	Event     string `json:"event"`
	Module    string `json:"module"`
	CreatedAt string `json:"created_at"`
	Timestamp string `json:"timestamp"`
	UserID    int64  `json:"user_id"`
}

type LogID struct {
	ID int64 `json:"id"`
}
