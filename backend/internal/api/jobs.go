package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"read_article/backend/internal/audio"
)

type SynthesisJob struct {
	ID                string    `json:"id"`
	Status            string    `json:"status"`
	Stage             string    `json:"stage"`
	Message           string    `json:"message"`
	Progress          float64   `json:"progress"`
	TotalSegments     int       `json:"total_segments"`
	CompletedSegments int       `json:"completed_segments"`
	Error             string    `json:"error,omitempty"`
	AudioURL          string    `json:"audio_url,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	audio []byte
}

type JobManager struct {
	mu   sync.RWMutex
	jobs map[string]*SynthesisJob
}

func NewJobManager() *JobManager {
	return &JobManager{
		jobs: make(map[string]*SynthesisJob),
	}
}

func (m *JobManager) Create(synthesizer *audio.Synthesizer, opts audio.SynthesizeOptions) (*SynthesisJob, error) {
	id, err := newJobID()
	if err != nil {
		return nil, fmt.Errorf("create job id: %w", err)
	}

	now := time.Now()
	job := &SynthesisJob{
		ID:        id,
		Status:    "running",
		Stage:     "queued",
		Message:   "准备中",
		Progress:  0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.mu.Lock()
	m.jobs[id] = job
	m.mu.Unlock()

	go m.run(context.Background(), synthesizer, opts, id)
	return cloneJob(job), nil
}

func (m *JobManager) run(ctx context.Context, synthesizer *audio.Synthesizer, opts audio.SynthesizeOptions, jobID string) {
	result, err := synthesizer.SynthesizeWithProgress(ctx, opts, func(update audio.ProgressUpdate) {
		m.update(jobID, func(job *SynthesisJob) {
			job.Stage = update.Stage
			job.Message = update.Message
			job.TotalSegments = update.TotalSegments
			job.CompletedSegments = update.CompletedSegments
			job.Progress = calculateProgress(update.Stage, update.CompletedSegments, update.TotalSegments)
		})
	})
	if err != nil {
		m.update(jobID, func(job *SynthesisJob) {
			job.Status = "failed"
			job.Stage = "failed"
			job.Message = "生成失败"
			job.Error = err.Error()
		})
		return
	}

	m.update(jobID, func(job *SynthesisJob) {
		job.Status = "completed"
		job.Stage = "completed"
		job.Message = "已完成"
		job.Progress = 100
		job.TotalSegments = len(result.Segments)
		job.CompletedSegments = len(result.Segments)
		job.AudioURL = "/api/v1/synthesize-jobs/" + job.ID + "/audio"
		job.audio = append([]byte(nil), result.Audio...)
	})
}

func (m *JobManager) Get(id string) (*SynthesisJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[id]
	if !ok {
		return nil, false
	}
	return cloneJob(job), true
}

func (m *JobManager) GetAudio(id string) ([]byte, bool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[id]
	if !ok {
		return nil, false, false
	}
	if len(job.audio) == 0 {
		return nil, true, false
	}
	return append([]byte(nil), job.audio...), true, true
}

func (m *JobManager) update(id string, mutate func(*SynthesisJob)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[id]
	if !ok {
		return
	}
	mutate(job)
	job.UpdatedAt = time.Now()
}

func cloneJob(job *SynthesisJob) *SynthesisJob {
	copy := *job
	copy.audio = nil
	return &copy
}

func calculateProgress(stage string, completed, total int) float64 {
	switch stage {
	case "splitting":
		return 5
	case "generating":
		if total <= 0 {
			return 10
		}
		return 10 + (float64(completed)/float64(total))*80
	case "merging":
		return 95
	case "completed":
		return 100
	case "failed":
		return 0
	default:
		return 0
	}
}

func newJobID() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
