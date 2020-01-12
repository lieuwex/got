package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	_ "github.com/mattn/go-sqlite3"
	nd "github.com/tj/go-naturaldate"
)

// TODO: make it posisble to use this program without running ruby timetrap
// first

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])

	lines := []string{
		"in: start an entry",
		"out: stop an entry",
		"resume: resume an entry",
		"edit: edit an entry",
		"now: show the current entry",
		"display: show all entries in the current sheet",
		"sheet: change the current sheet",
		"kill: delete an entry or timesheet",
	}

	for _, line := range lines {
		fmt.Fprintf(os.Stderr, "\t%s\n", line)
	}

	os.Exit(1)
}

func main() {
	fs := MakeFlagSet(map[string]string{
		"id": "0",

		"start": "",
		"end":   "",
		"at":    "",
	})

	if err := fs.Parse(); err != nil {
		panic(err)
	}

	id, err := strconv.ParseUint(fs.Values["id"], 10, 64)
	if err != nil {
		panic(err)
	}
	startString := fs.Values["start"]
	endString := fs.Values["end"]
	atString := fs.Values["at"]

	startDate, err := nd.Parse(startString, time.Now())
	if err != nil {
		startDate = time.Time{}
	}
	endDate, err := nd.Parse(endString, time.Now())
	if err != nil {
		endDate = time.Time{}
	}
	atDate, err := nd.Parse(atString, time.Now())
	if err != nil {
		atDate = time.Time{}
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	db, err := sql.Open("sqlite3", path.Join(homedir, ".timetrap.db"))
	if err != nil {
		panic(err)
	}

	state, err := MakeState(db)
	if err != nil {
		panic(err)
	}

	if len(fs.Strings) == 0 {
		usage()
	}
	command := fs.Strings[0]

	if id == 0 {
		if state.CurrentEntry == nil {
			id = state.LastCheckoutID
		} else {
			id = state.CurrentEntry.id
		}
	}

	var note string
	for i := 1; i < len(fs.Strings); i++ {
		if i > 1 {
			note += " "
		}

		note += fs.Strings[i]
	}

	sheet := state.CurrentSheet
	commands := map[string]func() error{
		"in": func() error {
			start := startDate
			if start == (time.Time{}) {
				start = atDate
			}
			if start == (time.Time{}) {
				start = time.Now()
			}

			id, err := state.StartEntry(note, sheet, start)
			if err != nil {
				return err
			}

			fmt.Printf("Checked into sheet \"%s\" (%d).\n", sheet, id)
			return nil
		},
		"out": func() error {
			end := endDate
			if end == (time.Time{}) {
				end = atDate
			}
			if end == (time.Time{}) {
				end = time.Now()
			}

			entry, err := state.GetEntry(id)
			if err != nil {
				return err
			} else if entry == nil {
				return fmt.Errorf("no entry with id %d found", id)
			}

			if err := state.StopEntry(id, end); err != nil {
				return err
			}

			fmt.Printf("Checked out of sheet \"%s\" (%d).\n", sheet, id)
			return nil
		},
		"resume": func() error {
			start := startDate
			if start == (time.Time{}) {
				start = atDate
			}
			if start == (time.Time{}) {
				start = time.Now()
			}

			entry, err := state.GetEntry(id)
			if err != nil {
				return err
			}

			if entry == nil {
				entries, err := state.GetAllEntries(sheet)
				if err != nil {
					return err
				}

				entry = entries[len(entries)-1]
				id = entry.id
			}

			newId, err := state.ResumeEntry(id, start)
			if err != nil {
				return err
			}

			fmt.Printf("Resuming \"%s\" from entry #%d with new ID #%d\n", entry.note, entry.id, newId)
			return nil
		},
		"now": func() error {
			entry, err := state.GetCurrentEntry()
			if err != nil {
				return err
			}

			if entry == nil {
				fmt.Fprintf(os.Stderr, "*%s: not running\n", sheet)
				return nil
			}

			duration, _ := entry.Running()
			fmt.Printf("*%s: %s (%s)\n", sheet, formatDuration(duration), entry.note)
			return nil
		},
		"edit": func() error {
			entry, err := state.GetEntry(id)
			if err != nil {
				return err
			}

			any := false
			if startDate != (time.Time{}) {
				any = true
				entry.start = startDate
			}
			if endDate != (time.Time{}) {
				any = true
				entry.end = &endDate
			}
			if note != "" {
				any = true
				entry.note = note
			}

			if !any {
				fmt.Println("nothing changed")
				return nil
			}

			_, err = state.db.Exec("update entries set note = ?, start = ?, end = ? where id = ?", entry.note, entry.start, entry.end, entry.id)
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "Id\tDay\tStart      End\tDuration\tNotes")
			entry.Write(w)
			return w.Flush()
		},

		"display": func() error {
			entries, err := state.GetAllEntries("")
			if err != nil {
				return err
			}

			fmt.Printf("Timesheet: %s\n", state.CurrentSheet)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

			fmt.Fprintln(w, "Id\tDay\tStart      End\tDuration\tNotes")

			var date time.Time
			var dayDuration time.Duration
			var dateString string

			for i, entry := range entries {
				end := ""
				if entry.end != nil {
					end = entry.end.Format("15:04:05")
				}

				var duration time.Duration
				if entry.end != nil {
					duration = entry.end.Sub(entry.start)
				} else {
					duration = time.Now().Sub(entry.start)
				}
				dayDuration += duration

				fmt.Fprintf(
					w,
					"%d\t%s\t%s - %s\t%s\t%s\n",
					entry.id,
					dateString,
					entry.start.Format("15:04:05"),
					end,
					formatDuration(duration),
					entry.note,
				)

				var next *Entry
				if i+1 < len(entries) {
					next = entries[i+1]
				}

				if next == nil || !sameDate(next.start, date) {
					// new day

					if i > 0 {
						fmt.Fprintf(
							w,
							"\t\t\t%s\t\n",
							formatDuration(dayDuration),
						)
					}

					if next != nil {
						date = next.start
					}

					dayDuration = 0
					dateString = entry.start.Format("Mon Jan 2, 2006")
				} else {
					dateString = ""
				}
			}

			return w.Flush()
		},

		"sheet": func() error {
			if strings.Contains(note, " ") {
				return errors.New("name cannot contain spaces")
			} else if note != "" {
				if err := state.SwitchSheet(note); err != nil {
					return err
				}
				fmt.Printf("Switching to sheet \"%s\"\n", note)
				return nil
			}

			sheets, err := state.GetAllSheets()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

			fmt.Fprintf(w, " Timesheet\tRunning\tToday\tTotal Time\n")
			for _, sheet := range sheets {
				curr, last := sheet == state.CurrentSheet, sheet == state.LastSheet

				prefix := ""
				if curr {
					prefix = "*"
				} else if last {
					prefix = "-"
				}

				running := "TODO"
				today := "TODO"
				total := "TODO"
				fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\n", prefix, sheet, running, today, total)
			}

			return w.Flush()
		},

		"kill": func() error {
			if fs.Values["id"] == "0" { // kill timesheet
				sheets, err := state.GetAllSheets()
				if err != nil {
					return err
				}
				has := false
				for _, sheet := range sheets {
					if sheet == note {
						has = true
						break
					}
				}
				if !has {
					return fmt.Errorf("no sheet with name %s found", note)
				}

				str := fmt.Sprintf("are you sure you want to delete sheet \"%s\"?", note)
				if !confirm(str, false) {
					return nil
				}

				if err := state.RemoveSheet(note); err != nil {
					return err
				}
				fmt.Println("it's killed")
				return nil
			}

			entry, err := state.GetEntry(id)
			if err != nil {
				return err
			} else if entry == nil {
				return fmt.Errorf("not entry with id %d found", id)
			}

			str := fmt.Sprintf("are you sure you want to delete entry #%d?", id)
			if !confirm(str, false) {
				return nil
			}

			if err := state.RemoveEntry(id); err != nil {
				return err
			}
			fmt.Println("it's killed")
			return nil
		},
	}

	var fullCommand string
	for key := range commands {
		hasPrefix := strings.HasPrefix(key, command)
		if hasPrefix && fullCommand == "" {
			fullCommand = key
		} else if hasPrefix {
			fmt.Fprintln(os.Stderr, "ambigious command")
			usage()
			return
		}
	}

	fn := commands[fullCommand]
	if fn == nil {
		fmt.Fprintf(os.Stderr, "unknown command %s\n", command)
		usage()
		return
	}

	if err := fn(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		usage()
	}
}
