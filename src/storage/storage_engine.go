package storage

import (
	"errors"
	"github.com/google/uuid"
	"sync"
	"time"
)

type DataEntry struct {
	Id                uuid.UUID `json:"id"`
	SeismicActivity   float32   `json:"seismic_activity"`
	TemperatureC      float32   `json:"temperature_c"`
	RadiationLevel    float32   `json:"radiation_level"`
	LocationId        string    `json:"location_id"`
	ModificationCount int       `json:"modification_count"`
}

var (
	ErrKeyNotFound = errors.New("key not found") // to be cascaded to 404
	ErrInsufficientMemory = errors.New("insufficient memory") // to be cascaded to 507
)


type segment struct {
	data map[string]DataEntry
	mu sync.RWMutex
}

type SegmentedHashTable struct {
	segments []*segment
	segmentMask uint64 // used to determine which segment a key belongs to
	maxSize uint64 // sets max storage capacity
	currentSize uint64
	sizeLock sync.RWMutex // for thread-safe concurrent access to all the *Size fields
}

