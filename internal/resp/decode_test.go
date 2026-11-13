package resp

import (
	"errors"
	"reflect"
	"testing"
)

func TestDecodeOne(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      interface{}
		wantN     int
		wantErr   error
		extraTail string // bytes after the value that must not be consumed
	}{
		{
			name:  "simple string",
			input: "+OK\r\n",
			want:  "OK",
			wantN: 5,
		},
		{
			name:  "error string",
			input: "-ERR unknown\r\n",
			want:  "ERR unknown",
			wantN: 14,
		},
		{
			name:  "integer positive",
			input: ":1000\r\n",
			want:  int64(1000),
			wantN: 7,
		},
		{
			name:  "integer negative",
			input: ":-3\r\n",
			want:  int64(-3),
			wantN: 5,
		},
		{
			name:  "integer zero",
			input: ":0\r\n",
			want:  int64(0),
			wantN: 4,
		},
		{
			name:  "bulk string",
			input: "$5\r\nhello\r\n",
			want:  "hello",
			wantN: 11,
		},
		{
			name:  "empty bulk string",
			input: "$0\r\n\r\n",
			want:  "",
			wantN: 6,
		},
		{
			name:  "null bulk string",
			input: "$-1\r\n",
			want:  nil,
			wantN: 5,
		},
		{
			name:  "array of bulk strings",
			input: "*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
			want:  []interface{}{"foo", "bar"},
			wantN: 22,
		},
		{
			name:  "empty array",
			input: "*0\r\n",
			want:  []interface{}{},
			wantN: 4,
		},
		{
			name:  "null array",
			input: "*-1\r\n",
			want:  nil,
			wantN: 5,
		},
		{
			name:  "nested array",
			input: "*2\r\n*1\r\n:1\r\n+OK\r\n",
			want:  []interface{}{[]interface{}{int64(1)}, "OK"},
			wantN: 17,
		},
		{
			name:      "consumes only first value",
			input:     "+PONG\r\n+OK\r\n",
			want:      "PONG",
			wantN:     7,
			extraTail: "+OK\r\n",
		},
		{
			name:    "incomplete simple string",
			input:   "+OK\r",
			wantErr: ErrIncomplete,
		},
		{
			name:    "incomplete bulk payload",
			input:   "$5\r\nhel",
			wantErr: ErrIncomplete,
		},
		{
			name:    "incomplete array element",
			input:   "*1\r\n$3\r\nfo",
			wantErr: ErrIncomplete,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: ErrIncomplete,
		},
		{
			name:    "unknown type",
			input:   "?\r\n",
			wantErr: ErrInvalid,
		},
		{
			name:    "bad integer",
			input:   ":12a\r\n",
			wantErr: ErrInvalid,
		},
		{
			name:    "bad length",
			input:   "$-2\r\n",
			wantErr: ErrInvalid,
		},
		{
			name:    "bulk missing trailing crlf",
			input:   "$5\r\nhelloXX",
			wantErr: ErrInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []byte(tt.input)
			got, n, err := DecodeOne(input)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("DecodeOne() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeOne() unexpected error: %v", err)
			}
			if n != tt.wantN {
				t.Fatalf("DecodeOne() n = %d, want %d", n, tt.wantN)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("DecodeOne() = %#v, want %#v", got, tt.want)
			}
			if tt.extraTail != "" && string(input[n:]) != tt.extraTail {
				t.Fatalf("remaining = %q, want %q", input[n:], tt.extraTail)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	got, err := Decode([]byte("*1\r\n$4\r\nPING\r\n"))
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	want := []interface{}{"PING"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Decode() = %#v, want %#v", got, want)
	}
}
