package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"read_article/backend/internal/audio"
	"read_article/backend/internal/config"
	"read_article/backend/internal/inference"
)

type Server struct {
	synthesizer *audio.Synthesizer
	jobs        *JobManager
}

type synthesizeRequest struct {
	Text            string   `json:"text"`
	ReferenceAudio  string   `json:"reference_audio"`
	GapSeconds      *float64 `json:"gap_seconds"`
	EmotionPresetID string   `json:"emotion_preset_id"`
}

func NewServer(cfg config.Config, client *inference.Client) *Server {
	return &Server{
		synthesizer: audio.NewSynthesizer(cfg, client),
		jobs:        NewJobManager(),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/emotion-presets", s.handleEmotionPresets)
	mux.HandleFunc("/api/v1/voice-presets", s.handleVoicePresets)
	mux.HandleFunc("/api/v1/synthesize-jobs", s.handleSynthesizeJobs)
	mux.HandleFunc("/api/v1/synthesize-jobs/", s.handleSynthesizeJob)
	mux.HandleFunc("/api/v1/synthesize", s.handleSynthesize)
	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleEmotionPresets(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"items": audio.EmotionPresets,
	})
}

func (s *Server) handleVoicePresets(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"items": audio.VoicePresets,
	})
}

func (s *Server) handleSynthesize(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost+", OPTIONS")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req synthesizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	result, err := s.synthesizer.Synthesize(r.Context(), audio.SynthesizeOptions{
		Text:            req.Text,
		ReferenceAudio:  req.ReferenceAudio,
		GapSeconds:      req.GapSeconds,
		EmotionPresetID: req.EmotionPresetID,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", `inline; filename="reading.wav"`)
	w.Header().Set("X-Segment-Count", strconv.Itoa(len(result.Segments)))
	w.Header().Set("X-Emotion-Preset", result.EmotionPreset.ID)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Audio)
}

func (s *Server) handleSynthesizeJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost+", OPTIONS")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req synthesizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	job, err := s.jobs.Create(s.synthesizer, audio.SynthesizeOptions{
		Text:            req.Text,
		ReferenceAudio:  req.ReferenceAudio,
		GapSeconds:      req.GapSeconds,
		EmotionPresetID: req.EmotionPresetID,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) handleSynthesizeJob(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet+", OPTIONS")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/synthesize-jobs/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}

	if strings.HasSuffix(path, "/audio") {
		jobID := strings.TrimSuffix(path, "/audio")
		jobID = strings.TrimSuffix(jobID, "/")
		s.handleSynthesizeJobAudio(w, jobID)
		return
	}

	job, ok := s.jobs.Get(path)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleSynthesizeJobAudio(w http.ResponseWriter, jobID string) {
	audioData, exists, ready := s.jobs.GetAudio(jobID)
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	if !ready {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "audio is not ready"})
		return
	}

	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", `inline; filename="reading.wav"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(audioData)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
