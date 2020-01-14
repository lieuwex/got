package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"text/tabwriter"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// TODO: make it posisble to use this program without running ruby timetrap
// first

var commands = MakeManager()

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])

	printFlag := func(name, description string) {
		fmt.Fprintf(os.Stderr, "\t--%s: %s\n", name, description)
	}
	fmt.Fprintf(os.Stderr, "\nflags:\n")
	printFlag("id", "the ID to manipulate/copy.  defaults to the current or last entry.")
	printFlag("at", "the time to use, this can be equal to --start or --end depending on the context.  always has a lower priority than --start or --end.")
	printFlag("start", "the start time to use")
	printFlag("end", "the end time to use")

	fmt.Fprintf(os.Stderr, "\ncommands:\n")
	cmds := commands.GetByPrefix("")
	for _, cmd := range cmds {
		fmt.Fprintf(os.Stderr, "\t%s: %s\n", cmd.Name, cmd.Description)
	}

	os.Exit(1)
}

func getState() (*State, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path.Join(homedir, ".timetrap.db"))
	if err != nil {
		return nil, err
	}

	return MakeState(db)
}

func main() {
	input, err := GetInput()
	if err != nil {
		panic(err)
	}

	state, err := getState()
	if err != nil {
		panic(err)
	}

	if input.ID == 0 {
		// if ID is not given..
		if state.CurrentEntry != nil {
			// .. ID is the current entry
			input.ID = state.CurrentEntry.id
		} else {
			// .. or the last checkout ID.
			input.ID = state.LastCheckoutID
		}
	}

	commands.AddCommand("in", "start an entry", "[--start, --at (now)] [note (\"\")]", func() error {
		start := input.Start
		if start == (time.Time{}) {
			start = input.At
		}
		if start == (time.Time{}) {
			start = time.Now()
		}

		sheet := state.CurrentSheet

		id, err := state.StartEntry(input.Note, sheet, start)
		if err != nil {
			return err
		}

		fmt.Printf("Checked into sheet \"%s\" (%d).\n", sheet, id)
		return nil
	})
	commands.AddCommand("out", "stop an entry", "[--end, --at (now)] [--id (current)]", func() error {
		end := input.End
		if end == (time.Time{}) {
			end = input.At
		}
		if end == (time.Time{}) {
			end = time.Now()
		}

		entry, err := state.GetEntry(input.ID)
		if err != nil {
			return err
		} else if entry == nil {
			return fmt.Errorf("no entry with id %d found", input.ID)
		}

		if err := state.StopEntry(input.ID, end); err != nil {
			return err
		}

		sheet := state.CurrentSheet

		fmt.Printf("Checked out of sheet \"%s\" (%d).\n", sheet, input.ID)
		return nil
	})
	commands.AddCommand("resume", "resume an entry", "[--start, --at (now)] [--id (last)]", func() error {
		start := input.Start
		if start == (time.Time{}) {
			start = input.At
		}
		if start == (time.Time{}) {
			start = time.Now()
		}

		entry, err := state.GetEntry(input.ID)
		if err != nil {
			return err
		}

		sheet := state.CurrentSheet
		id := input.ID

		if entry == nil {
			if rawId := input.Raw["id"]; rawId != "0" {
				return fmt.Errorf("no entry with ID %s found", rawId)
			}

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
	})
	commands.AddCommand("now", "shot the current entry", "", func() error {
		entry, err := state.GetCurrentEntry()
		if err != nil {
			return err
		}

		sheet := state.CurrentSheet

		if entry == nil {
			fmt.Fprintf(os.Stderr, "*%s: not running\n", sheet)
			return nil
		}

		duration, _ := entry.Duration()
		fmt.Printf("*%s: %s (%s)\n", sheet, formatDuration(duration), entry.note)
		return nil
	})
	commands.AddCommand("edit", "edit an entry", "[--id (current/last)] [--start] [--end] [note]", func() error {
		entry, err := state.GetEntry(input.ID)
		if err != nil {
			return err
		}

		any := false
		if input.Start != (time.Time{}) {
			any = true
			entry.start = input.Start
		}
		if input.End != (time.Time{}) {
			any = true
			entry.end = &input.End
		}
		if input.Note != "" {
			any = true
			entry.note = input.Note
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
	})

	commands.AddCommand("display", "show all entries in the given sheet", "[SHEET/all/full (current)]", func() error {
		var sheet string
		switch input.Note {
		case "":
			sheet = state.CurrentSheet
		case "all", "full": // TODO full /= all
			sheet = ""
		}

		entries, err := state.GetAllEntries(sheet)
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

			duration, _ := entry.Duration()
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
	})

	commands.AddCommand("sheet", "show sheets or change the current sheet", "[sheet]", func() error {
		if strings.Contains(input.Note, " ") {
			return errors.New("name cannot contain spaces")
		} else if input.Note != "" {
			if err := state.SwitchSheet(input.Note); err != nil {
				return err
			}
			fmt.Printf("Switching to sheet \"%s\"\n", input.Note)
			return nil
		}

		sheets, err := state.GetAllSheets()
		if err != nil {
			return err
		}

		foundCurrent := false

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		printInfo := func(prefix, sheet string, running, today, total time.Duration) {
			fmt.Fprintf(
				w,
				"%s%s\t%s\t%s\t%s\n",
				prefix,
				sheet,
				formatDuration(running),
				formatDuration(today),
				formatDuration(total),
			)
		}

		fmt.Fprintf(w, " Timesheet\tRunning\tToday\tTotal Time\n")
		for _, sheet := range sheets {
			curr, last := sheet == state.CurrentSheet, sheet == state.LastSheet

			prefix := " "
			if curr {
				foundCurrent = true
				prefix = "*"
			} else if last {
				prefix = "-"
			}

			entries, err := state.GetAllEntries(sheet)
			if err != nil {
				return err
			}

			running := SumDuration(entries, func(e *Entry) bool {
				_, running := e.Duration()
				return running
			})
			today := SumDuration(entries, func(e *Entry) bool {
				return sameDate(time.Now(), e.start)
			})
			total := SumDuration(entries, func(e *Entry) bool {
				return true
			})

			if running == 0 && today == 0 {
				for _, entry := range entries {
					duration, isRunning := entry.Duration()
					if isRunning {
						running += duration
					}
				}
			}

			printInfo(prefix, sheet, running, today, total)
		}

		if !foundCurrent {
			printInfo("*", state.CurrentSheet, 0, 0, 0)
		}

		return w.Flush()
	})

	commands.AddCommand("kill", "delete an entry or sheet", "--id <id>\n\t<sheet>", func() error {
		if input.Raw["id"] == "0" { // kill timesheet
			sheets, err := state.GetAllSheets()
			if err != nil {
				return err
			}
			has := false
			for _, sheet := range sheets {
				if sheet == input.Note {
					has = true
					break
				}
			}
			if !has {
				return fmt.Errorf("no sheet with name %s found", input.Note)
			}

			str := fmt.Sprintf("are you sure you want to delete sheet \"%s\"?", input.Note)
			if !confirm(str, false) {
				return nil
			}

			if err := state.RemoveSheet(input.Note); err != nil {
				return err
			}
			fmt.Println("it's killed")
			return nil
		}

		entry, err := state.GetEntry(input.ID)
		if err != nil {
			return err
		} else if entry == nil {
			return fmt.Errorf("not entry with id %d found", input.ID)
		}

		str := fmt.Sprintf("are you sure you want to delete entry #%d?", input.ID)
		if !confirm(str, false) {
			return nil
		}

		if err := state.RemoveEntry(input.ID); err != nil {
			return err
		}
		fmt.Println("it's killed")
		return nil
	})

	commands.AddCommand("help", "show usage of a command", "<command>", func() error {
		if input.Note == "" {
			return errors.New("help requires a command")
		}

		cmds := commands.GetByPrefix(input.Note)
		if len(cmds) == 0 {
			return fmt.Errorf("unknown command %s", input.Command)
		} else if len(cmds) > 1 {
			return errors.New("ambigious command")
		}

		cmd := cmds[0]
		fmt.Fprintf(os.Stderr, "%s: %s\n\t%s\n", cmd.Name, cmd.Description, cmd.Usage)

		return nil
	})

	if input.Command == "" {
		usage()
	}

	cmds := commands.GetByPrefix(input.Command)
	if len(cmds) == 0 {
		fmt.Fprintf(os.Stderr, "unknown command %s\n", input.Command)
		usage()
		return
	} else if len(cmds) > 1 {
		fmt.Fprintln(os.Stderr, "ambigious command")
		usage()
		return
	}

	if err := cmds[0].Fn(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n\n", err)
		usage()
	}
}
