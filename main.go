package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	nativeConfigBasenameYAML = "watchfs.yaml"
	nativeConfigBasenameJSON = "watchfs.json"
	nodemonConfigBasename    = "nodemon.json"
)

var defaultConfigBasenames = []string{
	nativeConfigBasenameYAML,
	nativeConfigBasenameJSON,
	nodemonConfigBasename,
}

const (
	formatJSON = "json"
	formatYAML = "yaml"
)

var formats = []string{
	formatJSON,
	formatYAML,
}

var config configuration
var (
	configPath          string
	extensions          stringsSetVar
	extensionsCSV       string
	watch               stringsSetVar
	watchCSV            string
	ignore              stringsSetVar
	ignoreCSV           string
	ignoreExtensions    stringsSetVar
	ignoreExtensionsCSV string
	signal              = enumVar{Choices: signals}
	ignoreOps           = enumSetVar{Choices: ops}
	ignoreOpsCSV        = enumSetVarCSV{enumSetVar{Choices: ops}}
	watchOps            = enumSetVar{Choices: ops}
	watchOpsCSV         = enumSetVarCSV{enumSetVar{Choices: ops}}
	action              = enumVar{Choices: actions, Value: "exec"}
	stdoutJSON          = json.NewEncoder(os.Stdout)
	stdoutJSONMu        sync.Mutex
	stderrJSON          = json.NewEncoder(os.Stderr)
	stderrJSONMu        sync.Mutex
	printConfigAndExit  bool
	printConfigFormat   = enumVar{Choices: formats, Value: formatYAML}
	quiet               bool
	ctx, ctxCancel      = context.WithCancel(context.Background())
)

func init() {
	log.SetOutput(ioutil.Discard)
	flag.StringVar(&configPath, "config", configPath, fmt.Sprintf("use the config file (JSON or YAML) at this path (defaults: %v)", defaultConfigBasenames))
	flag.StringVar(&configPath, "c", configPath, "(alias for -config)")
	flag.Var(&extensions, "ext", "add an extension to watch")
	flag.StringVar(&extensionsCSV, "exts", extensionsCSV, "add multiple watched extensions (CSV)")
	flag.StringVar(&extensionsCSV, "e", extensionsCSV, "(alias for -exts)")
	flag.Var(&watch, "watch", "add a path to watch")
	flag.StringVar(&watchCSV, "watches", watchCSV, "add multiple watched paths (CSV)")
	flag.StringVar(&watchCSV, "w", watchCSV, "(alias for -watches)")
	flag.Var(&ignore, "ignore", "add a path/glob to ignore")
	flag.Var(&ignore, "i", "(alias for -ignore)")
	flag.Var(&ignoreExtensions, "ignore-ext", "add an extension to ignore")
	flag.StringVar(&ignoreExtensionsCSV, "ignore-exts", "", "add multiple ignored extensions (CSV)")
	flag.Var(&signal, "signal", fmt.Sprintf("signal to send on changes (choices: %v)", signals))
	flag.Var(&signal, "s", "(alias for -signal)")
	flag.Var(&action, "action", fmt.Sprintf("set the action type for the default action (choices %v)", actions))
	flag.Var(&action, "a", "(alias for -action)")
	flag.Var(&watchOps, "op", fmt.Sprintf("add a filesystem operation to watch for (choices: %v)", ops))
	flag.Var(&watchOpsCSV, "ops", fmt.Sprintf("add filesystem operations to watch for (CSV) (choices: %v)", ops))
	flag.Var(&ignoreOps, "ignore-op", fmt.Sprintf("add a filesystem operation to ignore (choices: %v)", ops))
	flag.Var(&ignoreOpsCSV, "ignore-ops", fmt.Sprintf("add multiple ignored filesystem operations (CSV) (choices: %v)", ops))
	flag.BoolVar(&printConfigAndExit, "print-config", false, "print config to stdout and exit")
	flag.Var(&printConfigFormat, "print-config-format", fmt.Sprintf("print config in this format (choices: %v)", printConfigFormat.Choices))
	flag.BoolVar(&quiet, "quiet", quiet, "do not print events to stdout")
	flag.BoolVar(&quiet, "q", quiet, "(alias for -quiet)")
	flag.Parse()
	loadConfigFile()
	flagsToConfiguration()
	config.makeCanonical()
}

func main() {
	defer ctxCancel()
	if printConfigAndExit {
		switch printConfigFormat.Value {
		case formatJSON:
			config.writeJSON(os.Stdout)
		case formatYAML:
			config.writeYAML(os.Stdout)
		}
		return
	}
	if config.Paths == nil || len(config.Paths) == 0 {
		stderrJSONEncode(struct {
			Warning string `json:"warning"`
		}{
			Warning: "no paths to watch specified. to watch the current directory, use `watchfs -watch .`",
		})
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		onError(err)
		os.Exit(1)
	}
	defer w.Close()

	if config.Paths != nil {
		for _, path := range config.Paths {
			watchRecursive(w, path)
		}
	}
	for i := range config.Actions {
		action := &config.Actions[i]
		action.trigger = make(chan Event, 1)
		run := make(chan struct{}, 1)
		run <- struct{}{}
		go func() {
			for range run {
				if err := action.Run(ctx); err != nil {
					onError(struct {
						Action  *Action `json:"action"`
						Message string  `json:"message"`
					}{
						Action:  action,
						Message: err.Error(),
					})
				}
			}
		}()
		go func() {
			for e := range action.trigger {
				if action.tick != nil {
				outer:
					for {
						select {
						case <-action.tick:
							break outer
						case <-action.trigger:
						}
					}
				}
				select {
				case run <- struct{}{}:
				default:
					action.Notify(e)
				}
			}
		}()
	}
	go func() {
		for e := range w.Events {
			onEvent(Event{
				Name: e.Name,
				Op:   e.Op,
				Time: time.Now().Format(time.RFC3339),
			})
		}
	}()
	go func() {
		for err := range w.Errors {
			onError(struct {
				Message string `json:"message"`
			}{
				Message: err.Error(),
			})
		}
	}()

	<-make(chan struct{})
}

func flagsToConfiguration() {
	if len(extensions.Value) > 0 {
		config.Extensions = extensions.Values()
	}
	if len(extensionsCSV) > 0 {
		config.ExtensionsCSV = extensionsCSV
	}
	if len(watch.Value) > 0 {
		config.Paths = watch.Values()
	}
	if len(watchCSV) > 0 {
		for _, v := range strings.Split(watchCSV, ",") {
			config.Paths = append(config.Paths, strings.TrimSpace(v))
		}
	}
	if len(watchOps.Value) > 0 {
		config.Ops = watchOps.Values()
	}
	if len(watchOpsCSV.Value) > 0 {
		config.OpsCSV = watchOpsCSV.String()
	}
	if len(ignoreExtensions.Value) > 0 {
		config.Ignore = append(config.Ignore, Filter{
			Extensions: ignoreExtensions.Values(),
		})
	}
	if len(ignoreExtensionsCSV) > 0 {
		config.Ignore = append(config.Ignore, Filter{
			ExtensionsCSV: ignoreExtensionsCSV,
		})
	}
	if len(ignoreOps.Value) > 0 {
		config.Ignore = append(config.Ignore, Filter{
			Ops: ignoreOps.Values(),
		})
	}
	if len(ignoreOpsCSV.Values()) > 0 {
		config.Ignore = append(config.Ignore, Filter{
			OpsCSV: ignoreOpsCSV.String(),
		})
	}
	if len(ignore.Value) > 0 {
		config.Ignore = append(config.Ignore, Filter{
			Watch: ignore.Values(),
		})
	}
	if len(signal.Value) > 0 {
		config.Signal = signal.Value
	}
	if flag.NArg() > 0 {
		switch action.Value {
		case actionShell:
			var shellCommand []string
			for _, a := range flag.Args() {
				shellCommand = append(shellCommand, fmt.Sprintf("%q", a))
			}
			config.Actions = append(config.Actions, Action{
				ActionShell: &ActionShell{
					Command: strings.Join(shellCommand, " "),
				},
			})
		case actionExec:
			config.Actions = append(config.Actions, Action{
				ActionExec: &ActionExec{
					Command: flag.Args(),
				},
			})
		case actionDockerRun:
			var args []string
			if flag.NArg() > 1 {
				args = flag.Args()[1:]
			}
			config.Actions = append(config.Actions, Action{
				ActionDockerRun: &ActionDockerRun{
					Image:   flag.Arg(0),
					Command: &args,
				},
			})
		case actionHTTPGet:
			if flag.NArg() > 1 {
				onError(fmt.Sprintf("too many arguments for action '%s': %v", action.Value, flag.Args()))
			}
			config.Actions = append(config.Actions, Action{
				ActionHTTPGet: &ActionHTTPGet{
					URL: flag.Arg(0),
				},
			})
		}
	}
}

func stdoutJSONEncode(v interface{}) error {
	stdoutJSONMu.Lock()
	defer stdoutJSONMu.Unlock()
	return stdoutJSON.Encode(v)
}

func stderrJSONEncode(v interface{}) error {
	stderrJSONMu.Lock()
	defer stderrJSONMu.Unlock()
	return stderrJSON.Encode(v)
}

func loadConfigFile() {
	load := func(name string) bool {
		if _, err := os.Stat(name); err == nil {
			err := config.load(name)
			if err != nil {
				onError(err)
			}
			config.makeCanonical()
			return true
		}
		return false
	}
	if configPath != "" {
		load(configPath)
	}
	for _, name := range defaultConfigBasenames {
		if load(name) {
			return
		}
	}
}

func onError(err interface{}) {
	if v, ok := err.(error); ok {
		err = v.Error()
	}
	stderrJSONEncode(struct {
		Error interface{} `json:"error"`
	}{
		Error: err,
	})
}

func onInfo(info interface{}) {
	stderrJSONEncode(struct {
		Info interface{} `json:"info"`
	}{
		Info: info,
	})
}

func shouldNotify(e Event) bool {
	if all, any := config.Filter.Match(e); !(all || any) {
		return false
	}
	for _, f := range config.Ignore {
		if all, any := f.Match(e); all && any {
			return false
		}
	}
	return true
}

func onEvent(e Event) {
	if !shouldNotify(e) {
		return
	}
	for _, action := range config.Actions {
		if action.Match(e) {
			action.trigger <- e
		}
	}
	if quiet {
		return
	}
	stdoutJSONEncode(struct {
		Op   string `json:"op"`
		Path string `json:"path"`
	}{
		Path: e.Name,
		Op:   strings.ToLower(e.Op.String()),
	})
}

func shouldExclude(path string, info os.FileInfo) bool {
	for _, f := range config.Ignore {
		if _, any := f.Match(Event{Name: path}); any {
			return true
		}
	}
	return false
}

func watchRecursive(w *fsnotify.Watcher, path string) {
	path, err := filepath.Abs(path)
	if err != nil {
		return
	}
	_, err = os.Stat(path)
	if err != nil {
		return
	}
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			switch v := err.(type) {
			case *os.PathError:
				path := v.Path
				if absPath, err := filepath.Abs(path); err == nil {
					path = absPath
				}
				onError(struct {
					Op      string `json:"op"`
					Path    string `json:"path"`
					Message string `json:"message"`
				}{
					Op:      v.Op,
					Path:    v.Path,
					Message: v.Err.Error(),
				})
			default:
				onError(err)
			}
			return err
		}
		if shouldExclude(path, info) {
			return nil
		}
		if info.IsDir() {
			err := w.Add(path)
			if err != nil {
				onError(err)
			}
		}
		return nil
	})
}
