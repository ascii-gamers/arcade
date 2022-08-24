package raft

import (
	"fmt"
	"sync"
)

type LogEntry struct {
	Term     int
	Index    int
	Command  interface{}
	Timestep int
}

type Log struct {
	sync.RWMutex

	entries           []LogEntry
	lastIncludedIndex int
	lastIncludedTerm  int
}

func (l *Log) AppendEntry(entry LogEntry) {
	l.AppendEntries([]LogEntry{entry})
}

func (l *Log) AppendEntries(entries []LogEntry) {
	l.Lock()
	defer l.Unlock()

	l.entries = append(l.entries, entries...)
}

func (l *Log) GetEntry(index int) (LogEntry, bool) {
	l.RLock()
	defer l.RUnlock()

	originalIndex := index

	index -= l.lastIncludedIndex

	if index < 1 || index > len(l.entries) {
		return LogEntry{Command: l.lastIncludedIndex}, false
	}

	if l.entries[index-1].Index != originalIndex {
		fmt.Printf("%d %d %d %v\n", originalIndex, l.entries[index-1].Index, l.lastIncludedIndex, l.entries)
		panic("index mismatch")
	}

	return l.entries[index-1], true
}

func (l *Log) GetEntryAndFollowing(index int) []LogEntry {
	l.RLock()
	defer l.RUnlock()

	index -= l.lastIncludedIndex

	if index < 1 || index > len(l.entries) {
		return nil
	}

	entries := make([]LogEntry, len(l.entries[index-1:]))
	copy(entries, l.entries[index-1:])

	return entries
}

func (l *Log) GetEntries() []LogEntry {
	l.Lock()
	defer l.Unlock()

	entries := make([]LogEntry, len(l.entries))
	copy(entries, l.entries)

	return entries
}

func (l *Log) GetLastIncludedIndex() int {
	l.RLock()
	defer l.RUnlock()

	return l.lastIncludedIndex
}

func (l *Log) GetLastIncludedTerm() int {
	l.RLock()
	defer l.RUnlock()

	return l.lastIncludedTerm
}

func (l *Log) LastIndex() int {
	l.RLock()
	defer l.RUnlock()

	if len(l.entries) == 0 {
		return l.lastIncludedIndex
	}

	return l.entries[len(l.entries)-1].Index
}

func (l *Log) LastTerm() int {
	l.RLock()
	defer l.RUnlock()

	if len(l.entries) == 0 {
		return l.lastIncludedTerm
	}

	return l.entries[len(l.entries)-1].Term
}

func (l *Log) DeleteEntryAndFollowing(index int) {
	l.Lock()
	defer l.Unlock()

	index -= l.lastIncludedIndex

	if index < 1 || index > len(l.entries) {
		return
	}

	l.entries = l.entries[:index-1]
}

func (l *Log) DeleteEntriesPreceding(index int) error {
	l.Lock()
	defer l.Unlock()

	index -= l.lastIncludedIndex

	if index < 1 {
		return nil
	}

	if index > len(l.entries) {
		index = len(l.entries) + 1
	}

	l.entries = l.entries[index-1:]
	return nil
}

func (l *Log) Iter(f func(entry LogEntry) bool) {
	l.RLock()
	defer l.RUnlock()

	for _, entry := range l.entries {
		if !f(entry) {
			break
		}
	}
}

func (l *Log) SetEntries(entries []LogEntry) {
	l.Lock()
	defer l.Unlock()

	l.entries = entries
}

func (l *Log) SetLastIncludedIndex(index int) {
	l.Lock()
	defer l.Unlock()

	l.lastIncludedIndex = index
}

func (l *Log) SetLastIncludedTerm(term int) {
	l.Lock()
	defer l.Unlock()

	l.lastIncludedTerm = term
}
