package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/keshavrathinvael/Big-O-Solution/internal/storage"
)

type RequestData struct {
	ID              string  `json:"id"`
	SeismicActivity float32 `json:"seismic_activity"`
	TemperatureC    float32 `json:"temperature_c"`
	RadiationLevel  float32 `json:"radiation_level"`
}

type Server struct {
	store    *storage.SegmentedHashTable
	memPool  *storage.PoolManager
	isReady  bool
	keyRegex *regexp.Regexp
}

func CreateServer(store *storage.SegmentedHashTable, memPool *storage.PoolManager) *Server {
	keyRegex := regexp.MustCompile(`^[A-Z]+-[a-zA-Z0-9]{1,6}$`)

	return &Server{
		store:    store,
		memPool:  memPool,
		isReady:  true,
		keyRegex: keyRegex,
	}
}

func (s *Server) SetReady(ready bool) {
	s.isReady = ready
}

func (s *Server) Start(port int) error {
	http.HandleFunc("/health", s.healthHandler)
	http.HandleFunc("/", s.mainHandler)

	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.isReady {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Service unavailable")
	}
}

func (s *Server) mainHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")

	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r, path)
	case http.MethodPut:
		s.handlePut(w, r, path)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request, locationID string) {
	data, err := s.store.Get(locationID)
	if err != nil {
		if err == storage.ErrKeyNotFound {
			http.Error(w, "Location ID not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handlePut(w http.ResponseWriter, r *http.Request, locationID string) {
	var reqData RequestData

	d := json.NewDecoder(r.Body)
	d.Decode(&reqData)

	id, err := uuid.Parse(reqData.ID)
	if err != nil {
		http.Error(w, "Invalid UUID format", http.StatusBadRequest)
	}

	var data storage.DataEntry
	existingData, err := s.store.Get(locationID)
	if err == nil {
		data = existingData
		data.ModificationCount++
	} else if err == storage.ErrKeyNotFound {
		data = storage.DataEntry{
			Id:                id,
			ModificationCount: 1,
			LocationId:        locationID,
		}
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data.SeismicActivity = reqData.SeismicActivity
	data.TemperatureC = reqData.TemperatureC
	data.RadiationLevel = reqData.RadiationLevel

	err = s.store.Put(locationID, data)
	if err != nil {
		if err == storage.ErrInsufficientMemory {
			http.Error(w, "Insufficient storage", http.StatusInsufficientStorage)
		} else {
			http.Error(w, "Write rejected", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
}
