package cmd

import (
	"testing"
)

func Test_Root(t *testing.T) {
	want := []string{
		"actions",
		"billing",
		"license",
		"repo",
		"verified-emails",
	}

	have := []string{}

	for _, child := range RootCmd.Commands() {
		have = append(have, child.Name())
	}

	if len(want) != len(have) {
		t.Errorf("want %v, have %v", len(want), len(have))
	}

	for i := range want {
		if want[i] != have[i] {
			t.Errorf("want %v, have %v", want, have)
		}
	}
}
