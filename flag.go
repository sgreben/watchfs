package main

import (
	"fmt"
	"strings"
)

type enumVar struct {
	Choices []string
	Value   string
}

// Set implements the flag.Value interface.
func (so *enumVar) Set(v string) error {
	for _, c := range so.Choices {
		if strings.EqualFold(c, v) {
			so.Value = c
			return nil
		}
	}
	return fmt.Errorf(`"%s" must be one of [%s]`, v, strings.Join(so.Choices, " "))
}

func (so *enumVar) String() string {
	return so.Value
}

type stringsSetVar struct {
	Value map[string]bool
}

// Values returns a string slice of specified values.
func (so *stringsSetVar) Values() (out []string) {
	for v := range so.Value {
		out = append(out, v)
	}
	return
}

// Set implements the flag.Value interface.
func (so *stringsSetVar) Set(v string) error {
	if so.Value == nil {
		so.Value = make(map[string]bool)
	}
	so.Value[v] = true
	return nil
}

func (so *stringsSetVar) String() string {
	return strings.Join(so.Values(), ",")
}

type enumSetVar struct {
	Choices []string
	Value   map[string]bool
}

// Values returns a string slice of specified values.
func (so *enumSetVar) Values() (out []string) {
	for v := range so.Value {
		out = append(out, v)
	}
	return
}

// Set implements the flag.Value interface.
func (so *enumSetVar) Set(v string) error {
	var ok bool
	for _, c := range so.Choices {
		if strings.EqualFold(c, v) {
			v = c
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf(`"%s" must be one of [%s]`, v, strings.Join(so.Choices, " "))
	}
	if so.Value == nil {
		so.Value = make(map[string]bool)
	}
	so.Value[v] = true
	return nil
}

func (so *enumSetVar) String() string {
	return strings.Join(so.Values(), ",")
}

type enumSetVarCSV struct{ enumSetVar }

func (so *enumSetVarCSV) Set(vs string) error {
	if len(vs) > 0 {
		for _, v := range strings.Split(vs, ",") {
			if err := so.enumSetVar.Set(v); err != nil {
				return err
			}
		}
		return nil
	}
	return so.enumSetVar.Set(vs)
}
