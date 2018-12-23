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
go get -u github.com/sgreben/${APP}
```

### Pre-built binary

[Download a binary](https://github.com/sgreben/${APP}/releases/latest) from the releases page or from the shell:

```sh
# Linux
curl -L https://github.com/sgreben/${APP}/releases/download/${VERSION}/${APP}_${VERSION}_linux_x86_64.tar.gz | tar xz

# OS X
curl -L https://github.com/sgreben/${APP}/releases/download/${VERSION}/${APP}_${VERSION}_osx_x86_64.tar.gz | tar xz

# Windows
curl -LO https://github.com/sgreben/${APP}/releases/download/${VERSION}/${APP}_${VERSION}_windows_x86_64.zip
unzip ${APP}_${VERSION}_windows_x86_64.zip
```

## Usage

### CLI

```text
${APP} [OPTIONS] [COMMAND [ARGS...]]
```

```text
${USAGE}
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

- `paths`: (path or glob) list
- `exts`: filename extension list
- `ops`: [op](#schema-op) list
- `signal`: [signal](#schema-signal) string
- `ignores`: [filter](#schema-filter) list
- `env`: key/value map
- `delay`: duration string
- `actions`: [action](#schema-action) list

#### Schema: Filter

A predicate over filesystem events; an object with the keys:

- `paths`: (path or glob) list
- `exts`: filename extension list
- `ops`: [op](#schema-op) list

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
- `exec`: object
  - ([exec fields](#exec-fields))
- `dockerRun`: object
  - ([dockerRun fields](#dockerrun-fields))
- `httpGet`: object
  - ([httpGet fields](#httpget-fields))

##### common fields

- `delay`: duration string
- `ignore`: [filter](#schema-filter) list
- `signal`: [signal](#schema-signal)

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

To convert a nodemon.json to a canonical watchfs YAML config, you can use `watchfs -c path/to/nodemon.json -print-config`.
