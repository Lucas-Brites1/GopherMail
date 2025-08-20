package models

import "time"

type ErrorTracker struct {
	LastError     error
	ErrorCount    int
	LastErrorCode int
	ErrorHistory  []ErrorEntry
}

type ErrorEntry struct {
	Timestamp time.Time
	Code      int
	Message   string
	Operation string
}

// Structs para ajudar manter noção de errors no handshake smtp e nas funções de envio do smtp, 4xx retry, 5xx abort
