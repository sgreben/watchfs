// +build windows

package main

import (
	"os"
	"sort"
	"syscall"
)

var parseSignal = map[string]os.Signal{
	"SIGABRT": syscall.SIGABRT,
	"SIGALRM": syscall.SIGALRM,
	"SIGBUS":  syscall.SIGBUS,
	"SIGFPE":  syscall.SIGFPE,
	"SIGHUP":  syscall.SIGHUP,
	"SIGILL":  syscall.SIGILL,
	"SIGKILL": syscall.SIGKILL,
	"SIGPIPE": syscall.SIGPIPE,
	"SIGQUIT": syscall.SIGQUIT,
	"SIGSEGV": syscall.SIGSEGV,
	"SIGTERM": syscall.SIGTERM,
	"SIGTRAP": syscall.SIGTRAP,
}

var signals = func() (signals []string) {
	for signal := range parseSignal {
		signals = append(signals, signal)
	}
	sort.Strings(signals)
	return
}()
