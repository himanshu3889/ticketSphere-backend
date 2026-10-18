package bloomFilter

import "github.com/himanshu3889/ticketSphere-backend/configs"

// BloomFilterKey is the enum type for bloom filter identifiers

const (
	UserIDBloomFilter    configs.BloomFilterKey = "ticketsphere:bloom:user:id"
	UserEmailBloomFilter configs.BloomFilterKey = "ticketsphere:bloom:user:email"
	EventIDBloomFilter   configs.BloomFilterKey = "ticketsphere:bloom:event:id"
	SeatIDBloomFilter    configs.BloomFilterKey = "ticketsphere:bloom:seat:id"
)
