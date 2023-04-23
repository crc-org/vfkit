package util

import "testing"

func TestStringInSlice(t *testing.T) {
	type args struct {
		st string
		sl []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid find",
			args: args{
				st: "apple",
				sl: []string{"apple", "banana", "orange"},
			},
			want: true,
		},
		{
			name: "invalid find",
			args: args{
				st: "grape",
				sl: []string{"apple", "banana", "orange"},
			},
			want: false,
		},
		{
			name: "slice is nil",
			args: args{
				st: "grape",
				sl: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringInSlice(tt.args.st, tt.args.sl); got != tt.want {
				t.Errorf("StringInSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrimQuotes(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "valid",
			args: args{
				str: "\"foobar\"",
			},
			want: "foobar",
		},
		{
			name: "only front quote",
			args: args{
				str: "\"foobar",
			},
			want: "\"foobar",
		},
		{
			name: "only end quote",
			args: args{
				str: "foobar\"",
			},
			want: "foobar\"",
		},
		{
			name: "no quote",
			args: args{
				str: "foobar",
			},
			want: "foobar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrimQuotes(tt.args.str); got != tt.want {
				t.Errorf("TrimQuotes() = %v, want %v", got, tt.want)
			}
		})
	}
}
