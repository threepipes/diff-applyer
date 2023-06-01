package editdist

import (
	"reflect"
	"strings"
	"testing"
)

func TestWordBased(t *testing.T) {
	type args struct {
		src []string
		tgt []string
	}
	tests := []struct {
		name string
		args args
		want []Edit
	}{
		{
			name: "one word",
			args: args{
				src: []string{"foo"},
				tgt: []string{"bar"},
			},
			want: []Edit{
				{
					Cmd:  Rpl,
					Word: "bar",
				},
			},
		},
		{
			name: "two words",
			args: args{
				src: []string{"foo", "bar"},
				tgt: []string{"bar"},
			},
			want: []Edit{
				{
					Cmd: Del,
				},
				{
					Cmd: Ign,
				},
			},
		},
		{
			name: "some words",
			args: args{
				src: strings.Split("hello I am a student. Good bye", " "),
				tgt: strings.Split("hello everyone I'm a teacher. Good", " "),
			},
			want: []Edit{
				{
					Cmd: Ign,
				},
				{
					Cmd:  Rpl,
					Word: "everyone",
				},
				{
					Cmd:  Rpl,
					Word: "I'm",
				},
				{
					Cmd: Ign,
				},
				{
					Cmd:  Rpl,
					Word: "teacher.",
				},
				{
					Cmd: Ign,
				},
				{
					Cmd: Del,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WordBased(tt.args.src, tt.args.tgt); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WordBased() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
