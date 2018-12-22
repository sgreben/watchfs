# watchfs

`watchfs` is a self-contained nodemon-like (and partially [nodemon-compatible](#nodemonjson-config)) filesystem event watcher that is independent of the Node.js ecosystem. A notable addition is a convenient shortcut for `docker run` commands instead of raw `exec` calls.

Probably the best overview is given by

- a [small example config file](#yaml-config)
- the [config file schema](#schema-configuration)

## Contents

- [Contents](#contents)
- [Get it](#get-it)
  - [Using `go get`](#using-go-get)
  - [Pre-built binary](#pre-built-binary)
- [Usage](#usage)
  - [CLI](#cli)
  - [YAML config](#yaml-config)
    - [Schema: Configuration](#schema-configuration)
    - [Schema: Filter](#schema-filter)
    - [Schema: Signal](#schema-signal)
    - [Schema: Op](#schema-op)
    - [Schema: Action](#schema-action)
  - [`nodemon.json` config](#nodemonjson-config)

## Get it

### Using `go get`

```sh
go get -u github.com/sgreben/watchfs
```

### Pre-built binary

[Download a binary](https://github.com/sgreben/watchfs/releases/latest) from the releases page or from the shell:

```sh
# Linux
curl -L https://github.com/sgreben/watchfs/releases/download/0.9.0/watchfs_0.9.0_linux_x86_64.tar.gz | tar xz

# OS X
curl -L https://github.com/sgreben/watchfs/releases/download/0.9.0/watchfs_0.9.0_osx_x86_64.tar.gz | tar xz

# Windows
curl -LO https://github.com/sgreben/watchfs/releases/download/0.9.0/watchfs_0.9.0_windows_x86_64.zip
unzip watchfs_0.9.0_windows_x86_64.zip
```

## Usage

### CLI

```text
watchfs [OPTIONS] [COMMAND [ARGS...]]
```

```text
Usage of watchfs:
  -a value
    	(alias for -action) (default exec)
  -action value
    	set the action type for the default action (choices [httpGet exec dockerRun]) (default exec)
  -c string
    	(alias for -config)
  -config string
    	use the config file (JSON or YAML) at this path (defaults: [watchfs.yaml watchfs.json nodemon.json])
  -e string
    	(alias for -exts)
  -ext value
    	add an extension to watch
  -exts string
    	add multiple watched extensions (CSV)
  -i value
    	(alias for -ignore)
  -ignore value
    	add a path/glob to ignore
  -ignore-ext value
    	add an extension to ignore
  -ignore-exts string
    	add multiple ignored extensions (CSV)
  -ignore-op value
    	add a filesystem operation to ignore (choices: [chmod create remove rename write])
  -ignore-ops value
    	add multiple ignored filesystem operations (CSV) (choices: [chmod create remove rename write])
  -op value
    	add a filesystem operation to watch for (choices: [chmod create remove rename write])
  -ops value
    	add filesystem operations to watch for (CSV) (choices: [chmod create remove rename write])
  -print-config
    	print config to stdout and exit
  -print-config-format value
    	print config in this format (choices: [json yaml]) (default yaml)
  -q	(alias for -quiet)
  -quiet
    	do not print events to stdout
  -s value
    	(alias for -signal)
  -signal value
    	signal to send on changes (choices: [SIGABRT SIGALRM SIGBUS SIGCHLD SIGCONT SIGFPE SIGHUP SIGILL SIGINT SIGIO SIGIOT SIGKILL SIGPIPE SIGPROF SIGQUIT SIGSEGV SIGSTOP SIGSYS SIGTERM SIGTRAP SIGTSTP SIGTTIN SIGTTOU SIGURG SIGUSR1 SIGUSR2 SIGVTALRM SIGWINCH SIGXCPU SIGXFSZ])
  -w string
    	(alias for -watches)
  -watch value
    	add a path to watch
  -watches string
    	add multiple watched paths (CSV)
```

### YAML config

A (contrived) sample config that runs `go test .` using an `exec` action as well as using a `dockerRun` action whenever `.go` files change in `.` (the current directory).

```yaml
paths: [.]
exts: [go]
actions:
- exec:
    command: [go, test, .]
  exts: [go]
- dockerRun:
    image: golang:1.11.4-alpine3.8
    command: [go, test, .]
    env:
        CGO_ENABLED: 0
    volumes:
    - source: .
      target: /go/src/app
    workdir: /go/src/app
  delay: 1s
  exts: [go]
```

The `watchfs.yaml` file is expected to consist of one top-level [configuration object](#schema-configuration).

#### Schema: Configuration

An object with the keys:

- `exts`: filename extension list
- `ops`: [op](#schema-op) list
- `paths`: (path or glob) list
- `signal`: [signal](#schema-signal) string
- `ignores`: [filter](#schema-filter) list
- `env`: key/value map
- `delay`: duration
- `actions`: [action](#schema-action) list

#### Schema: Filter

A predicate over filesystem events; an object with the keys:

- `exts`: filename extension list
- `ops`: [op](#schema-op) list
- `paths`: (path or glob) list

#### Schema: Signal

A POSIX signal; one of the strings:

- `SIGABRT`
- `SIGALRM`
- `SIGBUS`
- `SIGCHLD`
- `SIGCONT`
- `SIGFPE`
- `SIGHUP`
- `SIGILL`
- `SIGINT`
- `SIGIO`
- `SIGIOT`
- `SIGKILL`
- `SIGPIPE`
- `SIGPROF`
- `SIGQUIT`
- `SIGSEGV`
- `SIGSTOP`
- `SIGSYS`
- `SIGTERM`
- `SIGTRAP`
- `SIGTSTP`
- `SIGTTIN`
- `SIGTTOU`
- `SIGURG`
- `SIGUSR1`
- `SIGUSR2`
- `SIGVTALRM`
- `SIGWINCH`
- `SIGXCPU`
- `SIGXFSZ`

#### Schema: Op

A filesystem operation; one of the strings:
- `chmod`
- `create`
- `remove`
- `rename`
- `write`

#### Schema: Action

Description of something that can be executed; an object with [filter](#schema-filter) fields, [common fields](#common-fields) and one of the action-specific sets of fields:
- ([common fields](#common-fields))
- ([filter](#schema-filter) fields)
- `exec`
  - ([exec fields](#exec-fields))
- `dockerRun`
  - ([dockerRun fields](#dockerrun-fields))
- `httpGet`
  - ([httpGet fields](#httpget-fields))

##### common fields

- `delay`: duration
- `ignore`: [filter](#schema-filter) list

##### `exec` fields

- `command`: string list
- `env`: key/value map
- `ignoreSignals`: boolean

##### `dockerRun` fields

- `image`: string
- `entrypoint`: string
- `command`: string list
- `env`: key/value map
- `workdir`: string
- `volumes`: [volume](#volume) list
- `extraArgs`: string list
- `ignoreSignals`: boolean

###### `volume` fields

- `source`: volume name or path
- `target`: path
- `type`: docker volume type string

##### `httpGet` fields

- `url`: URL string

### `nodemon.json` config

Most options from `nodemon`'s config file `nodemon.json` are supported. Exceptions will be documented here.

To convert a nodemon.json to a canonical watchfs YAML config, you can use `watchfs -c -print-config`.