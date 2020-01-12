package main

import (
	"fmt"
	"io"
	"time"
)

type Entry struct {
	id    uint64
	start time.Time
	end   *time.Time
	sheet string
	note  string
}

// TODO: dit gaat nog steeds niet geweldig wanneer gestart van ruby timetrap
func (e *Entry) Running() (time.Duration, bool) {
	isRunning := e.end == nil
	if isRunning {
		return time.Now().Sub(e.start.UTC()), true
	} else {
		return 0, false
	}
}

func (e *Entry) Write(w io.Writer) error {
	end := ""
	if e.end != nil {
		end = e.end.Format("15:04:05")
	}

	var duration time.Duration
	if e.end != nil {
		duration = e.end.Sub(e.start)
	} else {
		duration = time.Now().Sub(e.start)
	}

	_, err := fmt.Fprintf(
		w,
		"%d\t%s\t%s - %s\t%s\t%s\n",
		e.id,
		e.start.Format("Mon Jan 2, 2006"),
		e.start.Format("15:04:05"),
		end,
		formatDuration(duration),
		e.note,
	)
	return err
}

type DatabaseEntry struct {
	id    uint64
	start time.Time
	end   *time.Time
	sheet string
	note  string
}

func DatabaseEntryFromEntry(e *Entry) DatabaseEntry {
	var end *time.Time
	if e.end != nil {
		*end = e.end.Local()
	}

	return DatabaseEntry{
		id:    e.id,
		start: e.start.UTC(),
		end:   end,
		sheet: e.sheet,
		note:  e.note,
	}
}
func (e DatabaseEntry) ToEntry() (*Entry, error) {
	return &Entry{
		id:    e.id,
		start: e.start,
		end:   e.end,
		sheet: e.sheet,
		note:  e.note,
	}, nil
}
