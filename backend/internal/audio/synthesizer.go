package audio

import (
	"context"
	"fmt"
	"strings"
	"time"

	"read_article/backend/internal/config"
	"read_article/backend/internal/inference"
)

type EmotionPreset struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	EmotionPrompt string `json:"emotion_prompt"`
}

type VoicePreset struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ReferenceAudio string `json:"reference_audio"`
}

var EmotionPresets = []EmotionPreset{
	{
		ID:            "calm-bass-upright",
		Name:          "低音,正派,冷静",
		EmotionPrompt: "https://cdn.kuse.ai/tutorials/reademo.wav",
	},
}

var VoicePresets = []VoicePreset{
	{
		ID:             "none",
		Name:           "不使用参考音色",
		ReferenceAudio: "",
	},
	{
		ID:             "boy",
		Name:           "正太",
		ReferenceAudio: "https://cdn.kuse.ai/tutorials/readvoice.wav",
	},
}

type Synthesizer struct {
	client     *inference.Client
	defaultGap float64
}

type SynthesizeOptions struct {
	Text            string
	ReferenceAudio  string
	GapSeconds      *float64
	EmotionPresetID string
}

type SynthesisResult struct {
	Audio         []byte
	Segments      []string
	GapSeconds    float64
	EmotionPreset EmotionPreset
}

type ProgressUpdate struct {
	Stage             string `json:"stage"`
	Message           string `json:"message"`
	TotalSegments     int    `json:"total_segments"`
	CompletedSegments int    `json:"completed_segments"`
}

type ProgressReporter func(ProgressUpdate)

func NewSynthesizer(cfg config.Config, client *inference.Client) *Synthesizer {
	return &Synthesizer{
		client:     client,
		defaultGap: cfg.DefaultGap,
	}
}

func (s *Synthesizer) Synthesize(ctx context.Context, opts SynthesizeOptions) (*SynthesisResult, error) {
	return s.synthesize(ctx, opts, nil)
}

func (s *Synthesizer) SynthesizeWithProgress(ctx context.Context, opts SynthesizeOptions, reporter ProgressReporter) (*SynthesisResult, error) {
	return s.synthesize(ctx, opts, reporter)
}

func (s *Synthesizer) synthesize(ctx context.Context, opts SynthesizeOptions, reporter ProgressReporter) (*SynthesisResult, error) {
	reportProgress(reporter, ProgressUpdate{
		Stage:   "splitting",
		Message: "切片中",
	})

	segments := SplitText(opts.Text)
	if len(segments) == 0 {
		return nil, fmt.Errorf("text is empty after splitting")
	}

	gapSeconds := s.defaultGap
	if opts.GapSeconds != nil {
		gapSeconds = *opts.GapSeconds
	}
	if gapSeconds < 0 {
		return nil, fmt.Errorf("gap_seconds must be >= 0")
	}

	preset, err := presetByID(opts.EmotionPresetID)
	if err != nil {
		return nil, err
	}

	audioParts := make([][]byte, 0, len(segments))
	referenceAudio := strings.TrimSpace(opts.ReferenceAudio)
	reportProgress(reporter, ProgressUpdate{
		Stage:             "generating",
		Message:           fmt.Sprintf("语音生成中 (0/%d)", len(segments)),
		TotalSegments:     len(segments),
		CompletedSegments: 0,
	})
	for _, segment := range segments {
		raw, err := s.client.Generate(ctx, inference.TTSRequest{
			Text:           segment,
			ReferenceAudio: referenceAudio,
			EmotionPrompt:  preset.EmotionPrompt,
		})
		if err != nil {
			return nil, fmt.Errorf("generate audio for %q: %w", segment, err)
		}
		audioParts = append(audioParts, raw)
		reportProgress(reporter, ProgressUpdate{
			Stage:             "generating",
			Message:           fmt.Sprintf("语音生成中 (%d/%d)", len(audioParts), len(segments)),
			TotalSegments:     len(segments),
			CompletedSegments: len(audioParts),
		})
	}

	reportProgress(reporter, ProgressUpdate{
		Stage:             "merging",
		Message:           "拼接中",
		TotalSegments:     len(segments),
		CompletedSegments: len(segments),
	})
	merged, err := MergeWAVSegments(audioParts, time.Duration(gapSeconds*float64(time.Second)))
	if err != nil {
		return nil, fmt.Errorf("merge wav segments: %w", err)
	}

	return &SynthesisResult{
		Audio:         merged,
		Segments:      segments,
		GapSeconds:    gapSeconds,
		EmotionPreset: preset,
	}, nil
}

func reportProgress(reporter ProgressReporter, update ProgressUpdate) {
	if reporter != nil {
		reporter(update)
	}
}

func presetByID(id string) (EmotionPreset, error) {
	normalized := strings.TrimSpace(id)
	if normalized == "" {
		return EmotionPresets[0], nil
	}

	for _, preset := range EmotionPresets {
		if preset.ID == normalized {
			return preset, nil
		}
	}

	return EmotionPreset{}, fmt.Errorf("unknown emotion preset: %s", normalized)
}
