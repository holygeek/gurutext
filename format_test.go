package main

import (
	"testing"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		format string
		args   []interface{}
		want   string
	}{
		{
			format: "hello",
			want:   "hello",
		},
		{
			format: "hello\n",
			want:   "hello",
		},
	}

	for i, tt := range tests {
		got := format(tt.format, tt.args...)
		if tt.want != got {
			t.Errorf("tests[%d] failed\nwant: %s\n got: %s", i, tt.want, got)
		}
	}
}
