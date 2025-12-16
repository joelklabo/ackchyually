package app

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"foo", []string{"foo"}},
		{"  foo  bar  ", []string{"foo", "bar"}},
		{"FOO", []string{"foo"}},
	}

	for _, tt := range tests {
		got := tokenize(tt.in)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("tokenize(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
