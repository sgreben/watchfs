package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/google/shlex"

	"gopkg.in/yaml.v2"
)

type configuration struct {
	// User-facing representation
	Filter      `yaml:",inline,omitempty"`
	IgnorePaths []string          `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	Ignore      []Filter          `json:"ignores,omitempty" yaml:"ignores,omitempty"`
	Env         map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	ExecMap     map[string]string `json:"execMap,omitempty" yaml:"execMap,omitempty"`
	Actions     []Action          `json:"actions,omitempty" yaml:"actions,omitempty"`
	Delay       string            `json:"delay,omitempty" yaml:"delay,omitempty"`
	Signal      string            `json:"signal,omitempty" yaml:"signal,omitempty"`

	// Code-facing representation
	signal os.Signal
	delay  time.Duration
}

func (c *configuration) makeCanonical() {
	c.Filter.makeCanonical()
	for i := range c.Ignore {
		c.Ignore[i].makeCanonical()
	}
	s, ok := parseSignal[c.Signal]
	if !ok {
		s = syscall.SIGHUP
	}
	c.signal = s
	for ext, command := range c.ExecMap {
		tokens, err := shlex.Split(command)
		if err != nil {
			tokens = []string{command}
		}
		filter := Filter{Extensions: []string{ext}}
		filter.makeCanonical()
		c.Actions = append(c.Actions, Action{
			ActionExec: &ActionExec{
				Command: tokens,
			},
			Filter: filter,
		})
	}
	c.ExecMap = nil
	if n, err := strconv.ParseInt(c.Delay, 10, 64); err == nil {
		c.Delay = fmt.Sprint(time.Millisecond * time.Duration(n))
	}
	c.delay, _ = time.ParseDuration(c.Delay)
	for i := range c.Actions {
		if c.Actions[i].Delay == "" {
			c.Actions[i].Delay = c.Delay
		}
		c.Actions[i].makeCanonical()
	}
	if len(c.IgnorePaths) > 0 {
		c.Ignore = append(c.Ignore, Filter{
			Paths: c.IgnorePaths,
		})
		c.IgnorePaths = nil
	}
}

func (c *configuration) load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	dec.SetStrict(true)
	return dec.Decode(c)
}

func (c *configuration) writeJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}

func (c *configuration) writeYAML(w io.Writer) error {
	enc := yaml.NewEncoder(w)
	return enc.Encode(c)
}
