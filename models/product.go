package models

type Product struct {
	ID        int    `json:"id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Title     string `json:"title"`
	Price     int    `json:"price"`
	MessageID string `json:"message_id,omitempty"`
}
