package oid

import (
	"fmt"
	"reflect"
	"testing"
)

func TestHasPrefix(t *testing.T) {
	tests := []struct {
		name   string
		s      []int
		prefix []int
		result bool
	}{
		{
			name:   "完全一致",
			s:      []int{1, 2, 3, 4, 5},
			prefix: []int{1, 2, 3, 4, 5},
			result: true,
		},
		{
			name:   "完全一致",
			s:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 11},
			prefix: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 11},
			result: true,
		},
		{
			name:   "同じ値の長さで一致しない",
			s:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1},
			prefix: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 11},
			result: false,
		},
		{
			name:   "prefixのほうが長い",
			s:      []int{1, 2, 3, 4},
			prefix: []int{1, 2, 3, 4, 5},
			result: false,
		},
		{
			name:   "prefix検出できる",
			s:      []int{1, 2, 3, 4, 5},
			prefix: []int{1, 2, 3, 4},
			result: true,
		},
		{
			name:   "評価できる",
			s:      []int{1, 2, 4, 3, 5},
			prefix: []int{1, 2, 3},
			result: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPrefix(tt.s, tt.prefix)
			if got != tt.result {
				t.Errorf("got: %v want: %v", got, tt.result)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		result []int
		err    error
	}{
		{
			name:   "invalid last empty chars",
			src:    ".1.2.3.4.5.",
			result: nil,
			err:    fmt.Errorf(`invalid ".1.2.3.4.5.", remove a empty characters`),
		},
		{
			name:   "invalid middle empty chars",
			src:    ".1.2..4.5",
			result: nil,
			err:    fmt.Errorf(`invalid ".1.2..4.5", remove a empty characters`),
		},
		{
			name:   "valid",
			src:    ".1.2.3.4.5",
			result: []int{1, 2, 3, 4, 5},
			err:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.src)
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("got: %v want: %v", err, tt.err)
			}
			if !reflect.DeepEqual(got, tt.result) {
				t.Errorf("got: %v want: %v", got, tt.result)
			}
		})
	}

}
