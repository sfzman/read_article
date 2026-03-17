package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

type wavFile struct {
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Data          []byte
}

func MergeWAVSegments(segments [][]byte, gap time.Duration) ([]byte, error) {
	if len(segments) == 0 {
		return nil, fmt.Errorf("no audio segments to merge")
	}

	first, err := parseWAV(segments[0])
	if err != nil {
		return nil, fmt.Errorf("parse first wav segment: %w", err)
	}

	mergedData := make([]byte, 0, len(first.Data)*len(segments))
	silence := makeSilence(first, gap)

	for index, raw := range segments {
		current, err := parseWAV(raw)
		if err != nil {
			return nil, fmt.Errorf("parse wav segment %d: %w", index+1, err)
		}
		if err := ensureCompatible(first, current); err != nil {
			return nil, fmt.Errorf("wav segment %d incompatible: %w", index+1, err)
		}

		mergedData = append(mergedData, current.Data...)
		if index < len(segments)-1 && len(silence) > 0 {
			mergedData = append(mergedData, silence...)
		}
	}

	return buildWAV(wavFile{
		AudioFormat:   first.AudioFormat,
		NumChannels:   first.NumChannels,
		SampleRate:    first.SampleRate,
		ByteRate:      first.ByteRate,
		BlockAlign:    first.BlockAlign,
		BitsPerSample: first.BitsPerSample,
		Data:          mergedData,
	})
}

func parseWAV(data []byte) (wavFile, error) {
	if len(data) < 44 {
		return wavFile{}, fmt.Errorf("payload too short")
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return wavFile{}, fmt.Errorf("invalid RIFF/WAVE header")
	}

	var (
		offset    = 12
		foundFmt  bool
		foundData bool
		file      wavFile
	)

	for offset+8 <= len(data) {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8

		if offset+chunkSize > len(data) {
			return wavFile{}, fmt.Errorf("chunk %s exceeds payload length", chunkID)
		}

		chunkData := data[offset : offset+chunkSize]
		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return wavFile{}, fmt.Errorf("fmt chunk too short")
			}
			file.AudioFormat = binary.LittleEndian.Uint16(chunkData[0:2])
			file.NumChannels = binary.LittleEndian.Uint16(chunkData[2:4])
			file.SampleRate = binary.LittleEndian.Uint32(chunkData[4:8])
			file.ByteRate = binary.LittleEndian.Uint32(chunkData[8:12])
			file.BlockAlign = binary.LittleEndian.Uint16(chunkData[12:14])
			file.BitsPerSample = binary.LittleEndian.Uint16(chunkData[14:16])
			foundFmt = true
		case "data":
			file.Data = append([]byte(nil), chunkData...)
			foundData = true
		}

		offset += chunkSize
		if chunkSize%2 == 1 {
			offset++
		}
	}

	if !foundFmt {
		return wavFile{}, fmt.Errorf("missing fmt chunk")
	}
	if !foundData {
		return wavFile{}, fmt.Errorf("missing data chunk")
	}
	return file, nil
}

func ensureCompatible(base, current wavFile) error {
	switch {
	case base.AudioFormat != current.AudioFormat:
		return fmt.Errorf("audio format mismatch")
	case base.NumChannels != current.NumChannels:
		return fmt.Errorf("channel mismatch")
	case base.SampleRate != current.SampleRate:
		return fmt.Errorf("sample rate mismatch")
	case base.BlockAlign != current.BlockAlign:
		return fmt.Errorf("block align mismatch")
	case base.BitsPerSample != current.BitsPerSample:
		return fmt.Errorf("bits per sample mismatch")
	default:
		return nil
	}
}

func makeSilence(file wavFile, gap time.Duration) []byte {
	if gap <= 0 {
		return nil
	}

	frames := int(math.Round(float64(file.SampleRate) * gap.Seconds()))
	if frames <= 0 {
		return nil
	}

	return make([]byte, frames*int(file.BlockAlign))
}

func buildWAV(file wavFile) ([]byte, error) {
	var buffer bytes.Buffer
	dataSize := uint32(len(file.Data))
	riffSize := uint32(36) + dataSize

	buffer.WriteString("RIFF")
	if err := binary.Write(&buffer, binary.LittleEndian, riffSize); err != nil {
		return nil, err
	}
	buffer.WriteString("WAVE")

	buffer.WriteString("fmt ")
	if err := binary.Write(&buffer, binary.LittleEndian, uint32(16)); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, file.AudioFormat); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, file.NumChannels); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, file.SampleRate); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, file.ByteRate); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, file.BlockAlign); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, file.BitsPerSample); err != nil {
		return nil, err
	}

	buffer.WriteString("data")
	if err := binary.Write(&buffer, binary.LittleEndian, dataSize); err != nil {
		return nil, err
	}
	if _, err := buffer.Write(file.Data); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
