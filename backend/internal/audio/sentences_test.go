package audio

import (
	"reflect"
	"testing"
)

func TestSplitText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "split on chinese full stop and keep punctuation",
			text: "第一段。第二段。第三段。",
			want: []string{"第一段。", "第二段。", "第三段。"},
		},
		{
			name: "trim blanks and preserve trailing fragment",
			text: "  第一段。 \n\n 第二段  ",
			want: []string{"第一段。", "第二段"},
		},
		{
			name: "ignore empty fragments",
			text: "。。第一段。。",
			want: []string{"第一段。"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := SplitText(test.text)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("SplitText() = %#v, want %#v", got, test.want)
			}
		})
	}
}
