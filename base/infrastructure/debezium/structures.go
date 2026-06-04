package baseDebezium

import "encoding/json"

// represents the envelope structure Debezium sends
type DebeziumEvent struct {
	Before json.RawMessage `json:"before"`
	After  json.RawMessage `json:"after"`
	Source Source          `json:"source"`
	Op     string          `json:"op"`
	TsMs   int64           `json:"ts_ms"`
}

type Source struct {
	Version   string `json:"version"`
	Connector string `json:"connector"`
	Db        string `json:"db"`
	Table     string `json:"table"`
	Schema    string `json:"schema"`
	TsMs      int64  `json:"ts_ms"`
}
