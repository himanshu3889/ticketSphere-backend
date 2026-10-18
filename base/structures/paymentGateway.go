package structures

// Structure of the payment gateway webhook payload
// TODO: should this live here or in the payment module's structures?
type PaymentGatewayWebhookPayload struct {
	ID        string            `json:"id"`
	PaymentID string            `json:"payment_id"`
	Reference string            `json:"reference"`
	Amount    int64             `json:"amount"`
	Status    string            `json:"status"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp string            `json:"timestamp"`
}
