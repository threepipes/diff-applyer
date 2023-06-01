package main

import (
	"fmt"
	"strings"
	"testing"
)

func toStr(ps map[string]*RevisePair) string {
	res := make([]string, 0, len(ps))
	for id, p := range ps {
		res = append(res, fmt.Sprint("id=", id))
		res = append(res, p.String())
	}
	return strings.Join(res, "\n")
}

func Test_locateTargets(t *testing.T) {
	tests := []struct {
		name string
		ls   Lines
		want map[string]string
	}{
		{
			name: "normal case",
			ls:   readFileAsLines("fixtures/test1.md"),
			want: map[string]string{
				"1": strings.Join([]string{
					"id:1",
					"%%% req-start id:1 %%%",
					"hello, I am DIFF-MD.",
					"%%% req-end %%%",
					"%%% rev-start id:1 %%%",
					"hello, I am DIFF-MC.",
					"%%% rev-end %%%",
				}, "\n"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := locateTargets(tt.ls)
			if len(got) != len(tt.want) {
				t.Errorf("lengths are different: got = %v, want %v", toStr(got), tt.want)
			}
			for id, pw := range tt.want {
				pg, prs := got[id]
				if !prs {
					t.Errorf("id:%v does not exist", id)
				}
				if pw != pg.String() {
					t.Errorf("got = %v, want %v", pg.String(), pw)
				}
			}
		})
	}
}
