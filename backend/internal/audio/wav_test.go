package audio

import (
	"bytes"
	"testing"
	"time"
)

func TestMergeWAVSegments(t *testing.T) {
	t.Parallel()

	segmentOne, err := buildWAV(wavFile{
		AudioFormat:   1,
		NumChannels:   1,
		SampleRate:    4,
		ByteRate:      8,
		BlockAlign:    2,
		BitsPerSample: 16,
		Data:          []byte{1, 2, 3, 4},
	})
	if err != nil {
		t.Fatalf("build segment one: %v", err)
	}

	segmentTwo, err := buildWAV(wavFile{
		AudioFormat:   1,
		NumChannels:   1,
		SampleRate:    4,
		ByteRate:      8,
		BlockAlign:    2,
		BitsPerSample: 16,
		Data:          []byte{5, 6},
	})
	if err != nil {
		t.Fatalf("build segment two: %v", err)
	}

	merged, err := MergeWAVSegments([][]byte{segmentOne, segmentTwo}, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("MergeWAVSegments() error = %v", err)
	}

	parsed, err := parseWAV(merged)
	if err != nil {
		t.Fatalf("parse merged wav: %v", err)
	}

	wantData := []byte{1, 2, 3, 4, 0, 0, 0, 0, 5, 6}
	if !bytes.Equal(parsed.Data, wantData) {
		t.Fatalf("merged data = %v, want %v", parsed.Data, wantData)
	}
}

func TestMergeWAVSegmentsRejectsDifferentFormats(t *testing.T) {
	t.Parallel()

	segmentOne, err := buildWAV(wavFile{
		AudioFormat:   1,
		NumChannels:   1,
		SampleRate:    16000,
		ByteRate:      32000,
		BlockAlign:    2,
		BitsPerSample: 16,
		Data:          []byte{1, 2},
	})
	if err != nil {
		t.Fatalf("build segment one: %v", err)
	}

	segmentTwo, err := buildWAV(wavFile{
		AudioFormat:   1,
		NumChannels:   2,
		SampleRate:    16000,
		ByteRate:      64000,
		BlockAlign:    4,
		BitsPerSample: 16,
		Data:          []byte{3, 4, 5, 6},
	})
	if err != nil {
		t.Fatalf("build segment two: %v", err)
	}

	if _, err := MergeWAVSegments([][]byte{segmentOne, segmentTwo}, 0); err == nil {
		t.Fatal("expected incompatible wav formats to fail")
	}
}
