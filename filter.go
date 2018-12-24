package main

import (
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Filter is an filesystem event filter
type Filter struct {
	ExtensionsCSV string   `json:"ext,omitempty" yaml:"ext,omitempty"`
	Extensions    []string `json:"exts,omitempty" yaml:"exts,flow,omitempty"`
	OpsCSV        string   `json:"op,omitempty" yaml:"op,omitempty"`
	Ops           []string `json:"ops,omitempty" yaml:"ops,flow,omitempty"`

	extensions map[string]bool
	ops        map[fsnotify.Op]bool
}

// Match returns whether an event satisfies `all` or `any` of its predicates.
func (f *Filter) Match(e Event) (all, any bool) {
	ext := ext(e.Name)
	ext = strings.ToLower(ext)
	extensionsOk := f.extensions[ext]
	opsOk := f.ops[e.Op]
	empty := f.extensions == nil && f.ops == nil
	all = extensionsOk && opsOk
	all = all || empty
	any = extensionsOk || opsOk
	return
}

func (f *Filter) makeCanonical() {
	if f == nil {
		return
	}
	if len(f.ExtensionsCSV) > 0 {
		for _, v := range strings.Split(f.ExtensionsCSV, ",") {
			f.Extensions = append(f.Extensions, v)
		}
		f.ExtensionsCSV = ""
	}
	if len(f.OpsCSV) > 0 {
		for _, v := range strings.Split(f.OpsCSV, ",") {
			f.Ops = append(f.Ops, v)
		}
		f.OpsCSV = ""
	}
	if len(f.Extensions) > 0 {
		f.extensions = make(map[string]bool, len(f.Extensions))
		for _, ext := range f.Extensions {
			ext = strings.TrimSpace(ext)
			ext = strings.TrimPrefix(ext, ".")
			ext = strings.ToLower(ext)
			f.extensions[ext] = true
		}
	}
	if len(f.Ops) > 0 {
		f.ops = make(map[fsnotify.Op]bool, len(f.Ops))
		for _, opName := range f.Ops {
			opName = strings.TrimSpace(opName)
			if op, ok := parseOp[opName]; ok {
				f.ops[op] = true
			}
		}
	}
}
