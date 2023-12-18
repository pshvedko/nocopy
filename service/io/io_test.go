package io

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestReadBytes(t *testing.T) {
	type args struct {
		r io.Reader
		b []byte
	}
	tests := []struct {
		name    string
		args    args
		wantN   int
		wantErr error
	}{
		// TODO: Add test cases.
		{
			name: "",
			args: args{
				r: strings.NewReader("1234567890"),
				b: make([]byte, 0),
			},
			wantN:   0,
			wantErr: nil,
		}, {
			name: "",
			args: args{
				r: strings.NewReader("1234567890"),
				b: make([]byte, 5),
			},
			wantN:   5,
			wantErr: nil,
		}, {
			name: "",
			args: args{
				r: strings.NewReader("1234567890"),
				b: make([]byte, 10),
			},
			wantN:   10,
			wantErr: nil,
		}, {
			name: "",
			args: args{
				r: strings.NewReader("1234567890"),
				b: make([]byte, 20),
			},
			wantN:   10,
			wantErr: io.EOF,
		}, {
			name: "",
			args: args{
				r: iotest.ErrReader(io.ErrClosedPipe),
				b: make([]byte, 20),
			},
			wantN:   0,
			wantErr: io.ErrClosedPipe,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotN, err := ReadBytes(tt.args.r, tt.args.b)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("ReadBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("ReadBytes() gotN = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}
