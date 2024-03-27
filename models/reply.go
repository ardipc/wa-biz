package models

type Reply struct {
	ID        int    `json:"id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Name      string `json:"title"`
	Phone     string `json:"price"`
	MessageID string `json:"message_id,omitempty"`
	ProductID string `json:"product_id,omitempty"`
	Status    string `json:"status"`
}
