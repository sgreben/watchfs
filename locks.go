package main

import "sync"

// Locks is a set of named mutexes
type Locks struct {
	Map map[string]*sync.Mutex
	mu  sync.RWMutex
}

// Init initializes the lock map
func (l *Locks) Init() {
	l.Map = make(map[string]*sync.Mutex)
}

// Lock locks the mutexes with the given names
func (l *Locks) Lock(names []string) {
	if len(names) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, name := range names {
		name := name
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.mu.Lock()
			lock, ok := l.Map[name]
			if !ok {
				lock = &sync.Mutex{}
				l.Map[name] = lock
			}
			l.mu.Unlock()
			lock.Lock()
		}()
	}
	wg.Wait()
}

// Unlock unlocks the mutexes with the given names
func (l *Locks) Unlock(names []string) {
	if len(names) == 0 {
		return
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, name := range names {
		if lock, ok := l.Map[name]; ok {
			lock.Unlock()
		}
	}
}
