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
	LastUpdated       int64     `json:"-"`
}

var (
	ErrKeyNotFound        = errors.New("key not found")       // to be cascaded to 404
	ErrInsufficientMemory = errors.New("insufficient memory") // to be cascaded to 507
)

type segment struct {
	data map[string]DataEntry
	mu   sync.RWMutex
}

type SegmentedHashTable struct {
	segments    []*segment
	segmentMask uint64 // used to determine which segment a key belongs to
	maxSize     uint64 // sets max storage capacity
	currentSize uint64
	sizeLock    sync.RWMutex // for thread-safe concurrent access to all the *Size fields
}

func NewSegmentedHashTable(numSegments int, maxSizeBytes uint64) *SegmentedHashTable {
	// numSegments should always be a power of 2 for effiicient modulo with bit masking
	if numSegments <= 0 || (numSegments&(numSegments-1)) != 0 {
		numSegments--
		numSegments |= numSegments >> 1
		numSegments |= numSegments >> 2
		numSegments |= numSegments >> 4
		numSegments |= numSegments >> 8
		numSegments |= numSegments >> 16
		numSegments++
	}

	segments := make([]*segment, numSegments)
	for i := 0; i < numSegments; i++ {
		segments[i] = &segment{
			data: make(map[string]DataEntry),
		}
	}

	return &SegmentedHashTable{
		segments:    segments,
		segmentMask: uint64(numSegments - 1),
		maxSize:     maxSizeBytes,
		currentSize: 0,
	}
}

func (sht *SegmentedHashTable) getSegment(key string) *segment {
	h := fnv1a(key)
	return sht.segments[h&sht.segmentMask]
}

func (sht *SegmentedHashTable) Get(key string) (DataEntry, error) {
	segment := sht.getSegment(key)
	segment.mu.RLock()
	defer segment.mu.RUnlock()

	if entry, ok := segment.data[key]; ok {
		return entry, nil
	}
	return DataEntry{}, ErrKeyNotFound
}

func (sht *SegmentedHashTable) Put(key string, entry DataEntry) error {
	sht.sizeLock.RLock()
	if sht.currentSize >= sht.maxSize {
		sht.sizeLock.RUnlock()
		return ErrInsufficientMemory
	}
	sht.sizeLock.RUnlock()

	segment := sht.getSegment(key)
	segment.mu.Lock()
	defer segment.mu.Unlock()

	var entrySize uint64 = 100
	entrySize += uint64(len(key))
	entrySize += uint64(len(entry.Id))

	exists := false
	var oldSize uint64 = 0
	if oldEntry, exists := segment.data[key]; exists {
		oldSize = 100 + uint64(len(key)) + uint64(len(oldEntry.Id))
	}
	sht.sizeLock.Lock()
	if entrySize > oldSize {
		if sht.currentSize+(entrySize-oldSize) > sht.maxSize {
			sht.sizeLock.Unlock()
			return ErrInsufficientMemory
		}
		sht.currentSize += (entrySize - oldSize)
	} else if exists {
		sht.currentSize -= (oldSize - entrySize)
	} else {
		sht.currentSize += entrySize
	}
	sht.sizeLock.Unlock()

	entry.LastUpdated = time.Now().UnixNano()
	segment.data[key] = entry
	return nil
}

func (sht *SegmentedHashTable) Delete(key string) error {
	segment := sht.getSegment(key)
	segment.mu.Lock()
	defer segment.mu.Unlock()

	if entry, exists := segment.data[key]; exists {
		entrySize := 100 + uint64(len(key)) + uint64(len(entry.Id))

		sht.sizeLock.Lock()
		sht.currentSize -= entrySize
		sht.sizeLock.Unlock()

		delete(segment.data, key)
		return nil
	}
	return ErrKeyNotFound
}

// Size returns the current size in bytes of the hash table
func (sht *SegmentedHashTable) Size() uint64 {
	sht.sizeLock.RLock()
	defer sht.sizeLock.RUnlock()
	return sht.currentSize
}

// MaxSize returns the maximum size in bytes of the hash table
func (sht *SegmentedHashTable) MaxSize() uint64 {
	return sht.maxSize
}

func (sht *SegmentedHashTable) Count() int {
	count := 0
	for _, segment := range sht.segments {
		segment.mu.RLock()
		count += len(segment.data)
		segment.mu.RUnlock()
	}
	return count
}

func (sht *SegmentedHashTable) GetKeys() []string {
	keys := make([]string, 0)
	for _, segment := range sht.segments {
		segment.mu.RLock()
		for k := range segment.data {
			keys = append(keys, k)
		}
		segment.mu.RUnlock()
	}
	return keys
}

// fnv1a is a simple non-cryptographic hash function
func fnv1a(s string) uint64 {
	var h uint64 = 0xcbf29ce484222325
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 0x100000001b3
	}
	return h
}
