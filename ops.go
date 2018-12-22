package main

import (
	"sort"

	"github.com/fsnotify/fsnotify"
)

var parseOp = map[string]fsnotify.Op{
	"create": fsnotify.Create,
	"write":  fsnotify.Write,
	"remove": fsnotify.Remove,
	"rename": fsnotify.Rename,
	"chmod":  fsnotify.Chmod,
}

var ops = func() (ops []string) {
	for op := range parseOp {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	return
}()
