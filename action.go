package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

const (
	actionHTTPGet   = "httpGet"
	actionExec      = "exec"
	actionShell     = "shell"
	actionDockerRun = "dockerRun"
)

var actions = []string{
	actionHTTPGet,
	actionExec,
	actionShell,
	actionDockerRun,
}

var actionLocks = func() *Locks {
	var l Locks
	l.Init()
	return &l
}()

// Action is an operation triggered in response to an fsnotify event
type Action struct {
	*ActionHTTPGet   `json:"httpGet,omitempty" yaml:"httpGet,omitempty"`
	*ActionExec      `json:"exec,omitempty" yaml:"exec,omitempty"`
	*ActionShell     `json:"shell,omitempty" yaml:"shell,omitempty"`
	*ActionDockerRun `json:"dockerRun,omitempty" yaml:"dockerRun,omitempty"`
	Filter           `yaml:",inline,omitempty"`
	Ignore           *Filter  `json:"ignore,omitempty" yaml:"ignore,omitempty"`
	Delay            string   `json:"delay,omitempty" yaml:"delay,omitempty"`
	Locks            []string `json:"locks,omitempty" yaml:"locks,flow,omitempty"`

	trigger chan Event
	delay   time.Duration
	tick    <-chan time.Time
}

func (a *Action) makeCanonical() {
	a.Filter.makeCanonical()
	if a.Ignore != nil {
		a.Ignore.makeCanonical()
	}
	if n, err := strconv.ParseInt(a.Delay, 10, 64); err == nil {
		a.Delay = fmt.Sprint(time.Millisecond * time.Duration(n))
	}
	a.delay, _ = time.ParseDuration(a.Delay)
	if a.delay > 0 {
		a.tick = time.Tick(a.delay)
	}
	switch {
	case a.ActionExec != nil:
		a.ActionExec.makeCanonical()
	case a.ActionShell != nil:
		a.ActionShell.makeCanonical()
	case a.ActionDockerRun != nil:
		a.ActionDockerRun.makeCanonical()
	}
}

// Match returns whether an event passes the action's filters.
func (a *Action) Match(e Event) bool {
	if all, any := a.Filter.Match(e); !(all || any) {
		return false
	}
	if a.Ignore != nil {
		if all, any := a.Ignore.Match(e); all && any {
			return false
		}
	}
	return true
}

// Notify notifies the action about a filesystem event
func (a *Action) Notify(e Event) (bool, error) {
	switch {
	case a.ActionHTTPGet != nil:
		return a.ActionHTTPGet.Notify(e)
	case a.ActionExec != nil:
		return a.ActionExec.Notify(e)
	case a.ActionShell != nil:
		return a.ActionShell.Notify(e)
	case a.ActionDockerRun != nil:
		return a.ActionDockerRun.Notify(e)
	}
	return false, nil
}

// Run runs the action
func (a *Action) Run(ctx context.Context) error {
	actionLocks.Lock(a.Locks)
	defer actionLocks.Unlock(a.Locks)
	switch {
	case a.ActionHTTPGet != nil:
		return a.ActionHTTPGet.Run(ctx)
	case a.ActionExec != nil:
		return a.ActionExec.Run(ctx)
	case a.ActionShell != nil:
		return a.ActionShell.Run(ctx)
	case a.ActionDockerRun != nil:
		return a.ActionDockerRun.Run(ctx)
	}
	return nil
}

// ActionHTTPGet performs an HTTP GET to the given endpoint
type ActionHTTPGet struct {
	URL string `json:"url" yaml:"url"`
}

// Notify notifies the action about a filesystem event
func (a *ActionHTTPGet) Notify(e Event) (bool, error) {
	return false, nil
}

// Run runs the action
func (a *ActionHTTPGet) Run(ctx context.Context) error {
	parsed, err := url.Parse(a.URL)
	if err != nil {
		return err
	}
	if parsed.Scheme == "" {
		parsed.Scheme = "http"
	}
	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	return resp.Write(os.Stdout)
}

// ActionExec runs the given command
type ActionExec struct {
	Command       []string          `json:"command,omitempty" yaml:"command,flow,omitempty"`
	Env           map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	Signal        string            `json:"signal,omitempty" yaml:"signal,omitempty"`
	IgnoreSignals bool              `json:"ignoreSignals,omitempty" yaml:"ignoreSignals,omitempty"`
	command       *exec.Cmd
	signal        *os.Signal
}

func (a *ActionExec) makeCanonical() {
	if a.Signal != "" {
		signal, ok := parseSignal[a.Signal]
		if !ok {
			signal = defaultSignal
		}
		a.signal = &signal
	}
}

// Notify notifies the action about a filesystem event
func (a *ActionExec) Notify(e Event) (bool, error) {
	if a.command == nil {
		return false, nil
	}
	if a.command.Process == nil {
		return false, nil
	}
	if a.IgnoreSignals {
		return true, nil
	}
	s := config.signal
	if a.signal != nil {
		s = *a.signal
	}
	err := a.command.Process.Signal(s)
	return err == nil, err
}

// Run runs the action
func (a *ActionExec) Run(ctx context.Context) error {
	if len(a.Command) == 0 {
		return nil
	}
	name := a.Command[0]
	var args []string
	if len(a.Command) > 1 {
		args = a.Command[1:]
	}
	a.command = exec.CommandContext(ctx, name, args...)
	a.command.Stdout = os.Stdout
	a.command.Stderr = os.Stderr
	if len(a.Env) > 0 || len(config.Env) > 0 {
		a.command.Env = append(a.command.Env, os.Environ()...)
		for k, v := range config.Env {
			a.command.Env = append(a.command.Env, fmt.Sprintf("%s=%s", k, v))
		}
		for k, v := range a.Env {
			a.command.Env = append(a.command.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return a.command.Run()
}

// ActionShell runs the given command
type ActionShell struct {
	Command       string            `json:"command,omitempty" yaml:"command,flow,omitempty"`
	Shell         []string          `json:"shell,omitempty" yaml:"shell,flow,omitempty"`
	Env           map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	IgnoreSignals bool              `json:"ignoreSignals,omitempty" yaml:"ignoreSignals,omitempty"`
	Signal        string            `json:"signal,omitempty" yaml:"signal,omitempty"`

	command *exec.Cmd
	signal  *os.Signal
}

func (a *ActionShell) makeCanonical() {
	if a.Signal != "" {
		signal, ok := parseSignal[a.Signal]
		if !ok {
			signal = defaultSignal
		}
		a.signal = &signal
	}
}

// Notify notifies the action about a filesystem event
func (a *ActionShell) Notify(e Event) (bool, error) {
	if a.command == nil {
		return false, nil
	}
	if a.command.Process == nil {
		return false, nil
	}
	if a.IgnoreSignals {
		return true, nil
	}
	s := config.signal
	if a.signal != nil {
		s = *a.signal
	}
	err := a.command.Process.Signal(s)
	return err == nil, err
}

// Run runs the action
func (a *ActionShell) Run(ctx context.Context) error {
	if len(a.Command) == 0 {
		return nil
	}
	name := defaultShell
	args := append([]string(nil), defaultShellArgs...)
	if len(a.Shell) > 0 {
		name = a.Shell[0]
		if len(a.Shell) > 1 {
			args = a.Shell[1:]
		}
	}
	args = append(args, a.Command)
	a.command = exec.CommandContext(ctx, name, args...)
	a.command.Stdout = os.Stdout
	a.command.Stderr = os.Stderr
	if len(a.Env) > 0 || len(config.Env) > 0 {
		a.command.Env = append(a.command.Env, os.Environ()...)
		for k, v := range config.Env {
			a.command.Env = append(a.command.Env, fmt.Sprintf("%s=%s", k, v))
		}
		for k, v := range a.Env {
			a.command.Env = append(a.command.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return a.command.Run()
}

// ActionDockerRun runs a docker container for the given image
type ActionDockerRun struct {
	Image      string            `json:"image" yaml:"image"`
	Entrypoint *string           `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Command    *[]string         `json:"command,omitempty" yaml:"command,flow,omitempty"`
	Env        map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
	ExtraArgs  []string          `json:"extraArgs,omitempty" yaml:"extraArgs,omitempty"`
	WorkDir    *string           `json:"workdir,omitempty" yaml:"workdir,omitempty"`
	Volumes    []struct {
		Source string `json:"source,omitempty" yaml:"source,omitempty"`
		Target string `json:"target,omitempty" yaml:"target,omitempty"`
		Type   string `json:"type,omitempty" yaml:"type,omitempty"`
	} `json:"volumes,omitempty" yaml:"volumes,omitempty"`
	IgnoreSignals bool   `json:"ignoreSignals,omitempty" yaml:"ignoreSignals,omitempty"`
	Signal        string `json:"signal,omitempty" yaml:"signal,omitempty"`

	signal  *os.Signal
	command *exec.Cmd
}

func (a *ActionDockerRun) makeCanonical() {
	if a.Signal != "" {
		signal, ok := parseSignal[a.Signal]
		if !ok {
			signal = defaultSignal
		}
		a.signal = &signal
	}
}

// Notify notifies the action about a filesystem event
func (a *ActionDockerRun) Notify(e Event) (bool, error) {
	if a.command == nil {
		return false, nil
	}
	if a.command.Process == nil {
		return false, nil
	}
	if a.IgnoreSignals {
		return true, nil
	}
	s := config.signal
	if a.signal != nil {
		s = *a.signal
	}
	err := a.command.Process.Signal(s)
	return err == nil, err
}

// Run runs the action
func (a *ActionDockerRun) Run(ctx context.Context) error {
	args := []string{"run", "--init", "--rm", "-t", "-a", "stdout", "-a", "stderr"}
	if a.Entrypoint != nil {
		args = append(args, "--entrypoint", *a.Entrypoint)
	}
	for k, v := range config.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range a.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}
	for _, v := range a.Volumes {
		volumeType := "bind"
		if v.Type != "" {
			volumeType = v.Type
		}
		if volumeType == "bind" {
			v.Source, _ = filepath.Abs(v.Source)
		}
		args = append(args, "--mount", fmt.Sprintf("type=%s,source=%s,target=%s", volumeType, v.Source, v.Target))
	}
	if a.WorkDir != nil {
		args = append(args, "--workdir", *a.WorkDir)
	}
	args = append(args, a.ExtraArgs...)
	args = append(args, a.Image)
	if a.Command != nil {
		args = append(args, *a.Command...)
	}
	a.command = exec.CommandContext(ctx, "docker", args...)
	a.command.Stdout = os.Stdout
	a.command.Stderr = os.Stderr
	return a.command.Run()
}
