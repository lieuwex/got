package types

import (
	"time"
)

type Entry struct {
	ID    uint64
	Start time.Time
	End   *time.Time
	Sheet string
	Note  string
}

// TODO: dit gaat nog steeds niet geweldig wanneer gestart van ruby timetrap
func (e *Entry) Duration() (time.Duration, bool) {
	isRunning := e.End == nil
	if isRunning {
		return time.Now().Sub(e.Start), true
	} else {
		return e.End.Sub(e.Start), false
	}
}

type DatabaseEntry struct {
	ID    uint64
	Start time.Time
	End   *time.Time
	Sheet string
	Note  string
}

func DatabaseEntryFromEntry(e *Entry) DatabaseEntry {
	var end *time.Time
	if e.End != nil {
		*end = e.End.Local()
	}

	return DatabaseEntry{
		ID:    e.ID,
		Start: e.Start.UTC(),
		End:   end,
		Sheet: e.Sheet,
		Note:  e.Note,
	}
}
func (e DatabaseEntry) ToEntry() (*Entry, error) {
	return &Entry{
		ID:    e.ID,
		Start: e.Start,
		End:   e.End,
		Sheet: e.Sheet,
		Note:  e.Note,
	}, nil
}
