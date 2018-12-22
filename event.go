package main

import "github.com/fsnotify/fsnotify"

// Event is a fsnotify.Event with a timestamp
type Event struct {
	Name string
	Op   fsnotify.Op
	Time string
}
