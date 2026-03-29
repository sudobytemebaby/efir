package mapper_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sudobytemebaby/efir/services/shared/pkg/mapper"
)

func TestSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []int
		fn    func(int) string
		want  []string
	}{
		{
			name:  "maps each element",
			input: []int{1, 2, 3},
			fn:    strconv.Itoa,
			want:  []string{"1", "2", "3"},
		},
		{
			name:  "empty slice returns empty slice",
			input: []int{},
			fn:    strconv.Itoa,
			want:  []string{},
		},
		{
			name:  "nil slice returns empty slice",
			input: nil,
			fn:    strconv.Itoa,
			want:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mapper.Slice(tc.input, tc.fn)
			require.Len(t, got, len(tc.want))
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEnum(t *testing.T) {
	t.Parallel()

	m := map[string]int{
		"one": 1,
		"two": 2,
	}

	tests := []struct {
		name     string
		from     string
		fallback int
		want     int
	}{
		{
			name:     "known value returns mapped value",
			from:     "one",
			fallback: -1,
			want:     1,
		},
		{
			name:     "unknown value returns fallback",
			from:     "unknown",
			fallback: -1,
			want:     -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mapper.Enum(m, tc.from, tc.fallback)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEnumWithOk(t *testing.T) {
	t.Parallel()

	m := map[string]int{
		"one": 1,
	}

	tests := []struct {
		name   string
		from   string
		wantV  int
		wantOk bool
	}{
		{
			name:   "known value returns value and true",
			from:   "one",
			wantV:  1,
			wantOk: true,
		},
		{
			name:   "unknown value returns zero and false",
			from:   "unknown",
			wantV:  0,
			wantOk: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := mapper.EnumWithOk(m, tc.from)
			assert.Equal(t, tc.wantV, got)
			assert.Equal(t, tc.wantOk, ok)
		})
	}
}
