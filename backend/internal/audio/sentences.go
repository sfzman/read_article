package audio

import "strings"

func SplitText(text string) []string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	var (
		segments []string
		builder  strings.Builder
	)

	flush := func() {
		segment := strings.TrimSpace(builder.String())
		builder.Reset()
		if segment == "" || strings.Trim(segment, "。\n\t ") == "" {
			return
		}
		if segment != "" {
			segments = append(segments, segment)
		}
	}

	for _, r := range normalized {
		builder.WriteRune(r)
		if r == '。' {
			flush()
		}
	}

	flush()
	return segments
}
