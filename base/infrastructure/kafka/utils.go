package baseKafka

import (
	"strconv"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/segmentio/kafka-go"
)

// Parse all headers at once into a struct
type MessageMetadata struct {
	TraceID     snowflake.ID `json:"trace_id"`
	UserID      snowflake.ID `json:"user_id"`
	IngestTime  time.Time    `json:"ingest_time"`  // When request first hit the API Gateway
	PublishTime time.Time    `json:"publish_time"` // When the previous producer wrote to Kafka
}

// Parse the kafka headers into the fix struct
func ParseKafkaMessageHeaders(msg *kafka.Message) *MessageMetadata {
	// Initialize with safe defaults
	meta := &MessageMetadata{
		IngestTime:  time.Now(),
		PublishTime: time.Now(),
	}

	for _, h := range msg.Headers {
		val := string(h.Value)

		switch h.Key {
		case "trace_id":
			if id, err := utils.ValidSnowflakeID(val); err == nil {
				meta.TraceID = id
			}

		case "user_id":
			if id, err := utils.ValidSnowflakeID(val); err == nil {
				meta.UserID = id
			}

		case "ingest_time":
			// Parse Unix Milliseconds (int64)
			if ms, err := strconv.ParseInt(val, 10, 64); err == nil {
				meta.IngestTime = time.UnixMilli(ms)
			}

		case "publish_time":
			// Parse Unix Milliseconds (int64)
			if ms, err := strconv.ParseInt(val, 10, 64); err == nil {
				meta.PublishTime = time.UnixMilli(ms)
			}
		}
	}

	return meta
}
