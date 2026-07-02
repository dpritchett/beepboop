package player

import (
	"errors"
	"os/exec"
)

var errNotFound = errors.New("not found")

type Command struct {
	Name string
	Args []string
}

type LookPath func(string) (string, error)

func FirstAvailable(lookPath LookPath) (Command, bool) {
	for _, candidate := range []Command{
		{Name: "aplay"},
		{Name: "paplay"},
		{Name: "ffplay", Args: []string{"-nodisp", "-autoexit", "-loglevel", "quiet"}},
	} {
		if _, err := lookPath(candidate.Name); err == nil {
			return candidate, true
		}
	}
	return Command{}, false
}

func SystemLookPath(name string) (string, error) {
	return exec.LookPath(name)
}
