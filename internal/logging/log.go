package logging

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// generateUUID creates a random UUID for correlation
func generateUUID() string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		rand.Uint32(),
		rand.Uint32()&0xffff,
		rand.Uint32()&0xffff|0x4000,
		rand.Uint32()&0x3fff|0x8000,
		rand.Uint64()&0xffffffffffff)
}

// LogWithRequest adds request ID and timestamp to the log message
func LogWithRequest(message string) {
	timestamp := time.Now().Format(time.RFC3339)
	requestID := generateUUID()
	log.Printf("[%s] [%s] %s", timestamp, requestID, message)
}
