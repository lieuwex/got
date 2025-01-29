package main

import (
	"database/sql"
	"errors"
	"fmt"
	"got/types"
	"got/utils"
	"io"
	"os"
	"path"
	"strings"
	"text/tabwriter"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var commands = MakeManager()

func writeEntry(e *types.Entry, w io.Writer) error {
	end := ""
	if e.End != nil {
		end = e.End.Format("15:04:05")
	}

	var duration time.Duration
	if e.End != nil {
		duration = e.End.Sub(e.Start)
	} else {
		duration = time.Now().Sub(e.Start)
	}

	_, err := fmt.Fprintf(
		w,
		"%d\t%s\t%s - %s\t%s\t%s\n",
		e.ID,
		e.Start.Format("Mon Jan 2, 2006"),
		e.Start.Format("15:04:05"),
		end,
		utils.FormatDuration(duration),
		e.Note,
	)
	return err
}

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
	printFlag("formatter", "the formatter to use.  can be 'human' or 'json'")
	printFlag("filter", "filter some outputs based on entry note")

	fmt.Fprintf(os.Stderr, "\ncommands:\n")
	cmds := commands.GetByPrefix("")
	for _, cmd := range cmds {
		fmt.Fprintf(os.Stderr, "\t%s: %s\n", strings.Join(cmd.Names, ", "), cmd.Description)
	}

	os.Exit(1)
}

func getState() (*State, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	fname := path.Join(homedir, ".timetrap.db")

	dbNew := false
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		dbNew = true
	}

	db, err := sql.Open("sqlite3", fname)
	if err != nil {
		return nil, err
	}

	if dbNew {
		if err := runSchema(db); err != nil {
			return nil, err
		}
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

	currentEntry, err := state.GetCurrentEntry()
	if err != nil {
		panic(err)
	}

	meta, err := state.GetMeta()
	if err != nil {
		panic(err)
	}

	if input.ID == 0 {
		// if ID is not given..
		if currentEntry != nil {
			// .. ID is the current entry
			input.ID = currentEntry.ID
		} else {
			// .. or the last checkout ID.
			input.ID = meta.LastCheckoutID
		}
	}

	commands.AddCommand([]string{"in", "start"}, "start an entry", "[--start, --at (now)] [note (\"\")]", func() error {
		start := input.Start
		if start == (time.Time{}) {
			start = input.At
		}
		if start == (time.Time{}) {
			start = time.Now()
		}

		sheet := meta.CurrentSheet

		id, err := state.StartEntry(input.Note, sheet, start)
		if err != nil {
			return err
		}

		fmt.Printf("Checked into sheet \"%s\" (%d).\n", sheet, id)
		return nil
	})
	commands.AddCommand([]string{"out", "end"}, "stop an entry", "[--end, --at (now)] [--id (current)]", func() error {
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
			return fmt.Errorf("no entry with ID %d found", input.ID)
		}

		if err := state.StopEntry(input.ID, end); err != nil {
			return err
		}

		sheet := meta.CurrentSheet

		fmt.Printf("Checked out of sheet \"%s\" (%d).\n", sheet, input.ID)
		return nil
	})
	commands.AddCommand([]string{"resume"}, "resume an entry", "[--start, --at (now)] [--id (last)]", func() error {
		start := input.Start
		if start == (time.Time{}) {
			start = input.At
		}
		if start == (time.Time{}) {
			start = time.Now()
		}

		var entry *types.Entry
		if id := input.Raw["id"]; id != "0" {
			var err error
			entry, err = state.GetEntry(input.ID)
			if err != nil {
				return err
			} else if entry == nil {
				return fmt.Errorf("no entry with ID %s found", id)
			}

			if entry.Sheet != meta.CurrentSheet {
				state.SwitchSheet(entry.Sheet)
			}
		} else {
			entries, err := state.GetAllEntries(meta.CurrentSheet)
			if err != nil {
				return err
			}

			entry = utils.GetNth(entries, len(entries)-1)
			if entry == nil {
				return errors.New("no entries")
			}
		}

		newId, err := state.StartEntry(entry.Note, entry.Sheet, start)
		if err != nil {
			return err
		}

		fmt.Printf("Resuming \"%s\" from entry #%d with new ID #%d\n", entry.Note, entry.ID, newId)
		return nil
	})
	commands.AddCommand([]string{"now"}, "show the current entry", "", func() error {
		entry, err := state.GetCurrentEntry()
		if err != nil {
			return err
		}

		if entry == nil {
			fmt.Fprintf(os.Stderr, "*%s: not running\n", meta.CurrentSheet)
			return nil
		}

		duration, _ := entry.Duration()
		fmt.Printf("*%s: %s (%s)\n", entry.Sheet, utils.FormatDuration(duration), entry.Note)
		return nil
	})
	commands.AddCommand([]string{"edit"}, "edit an entry", "[--id (current/last)] [--start] [--end] [note]", func() error {
		entry, err := state.GetEntry(input.ID)
		if err != nil {
			return err
		} else if entry == nil {
			return fmt.Errorf("no entry with ID %d found", input.ID)
		}

		any := false
		if input.Start != (time.Time{}) {
			any = true
			entry.Start = input.Start
		}
		if input.End != (time.Time{}) {
			any = true
			entry.End = &input.End
		}
		if input.Note != "" {
			any = true
			entry.Note = input.Note
		}

		if !any {
			fmt.Println("nothing changed")
			return nil
		}

		if err := state.EditEntry(
			entry.ID,
			entry.Sheet,
			entry.Note,
			entry.Start,
			entry.End,
		); err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "Id\tDay\tStart      End\tDuration\tNotes")
		writeEntry(entry, w)
		return w.Flush()
	})

	commands.AddCommand([]string{"display"}, "show all entries in the given sheet", "[SHEET/all/full (current)]", func() error {
		sheet := input.Note
		switch input.Note {
		case "":
			sheet = meta.CurrentSheet
		case "all", "full": // TODO full /= all
			sheet = ""
		}

		entries, err := state.GetAllEntries(sheet)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			return fmt.Errorf("Can't find sheet matching \"%s\"", sheet)
		}

		if input.Filter != "" {
			filtered := []*types.Entry{}
			for _, entry := range entries {
				if entry.Note != input.Filter {
					continue
				}

				filtered = append(filtered, entry)
			}
			entries = filtered
		}

		return input.Formatter.Write(os.Stdout, &types.FormatterInput{
			Sheet:   sheet,
			Entries: entries[:],
		})
	})

	commands.AddCommand([]string{"sheet"}, "show sheets or change the current sheet", "[sheet]", func() error {
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
				utils.FormatDuration(running),
				utils.FormatDuration(today),
				utils.FormatDuration(total),
			)
		}

		fmt.Fprintf(w, " Timesheet\tRunning\tToday\tTotal Time\n")
		for _, sheet := range sheets {
			curr, last := sheet == meta.CurrentSheet, sheet == meta.LastSheet

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

			running := utils.SumDuration(entries, func(e *types.Entry) bool {
				_, running := e.Duration()
				return running
			})
			today := utils.SumDuration(entries, func(e *types.Entry) bool {
				return utils.SameDate(time.Now(), e.Start)
			})
			total := utils.SumDuration(entries, func(e *types.Entry) bool {
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
			printInfo("*", meta.CurrentSheet, 0, 0, 0)
		}

		return w.Flush()
	})

	commands.AddCommand([]string{"kill"}, "delete an entry or sheet", "--id <id>\n\t<sheet>", func() error {
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
			if !utils.Confirm(str, false) {
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
			return fmt.Errorf("no entry with ID %d found", input.ID)
		}

		str := fmt.Sprintf("are you sure you want to delete entry #%d (\"%s\")?", entry.ID, entry.Note)
		if !utils.Confirm(str, false) {
			return nil
		}

		if err := state.RemoveEntry(entry.ID); err != nil {
			return err
		}
		fmt.Println("it's killed")
		return nil
	})

	commands.AddCommand([]string{"idle"}, "show the time since you last checked out", "[sheet]", func() error {
		sheet := input.Note
		switch sheet {
		case "":
			sheet = meta.CurrentSheet
		case "all", "full": // TODO: full /= all
			sheet = ""
		}

		entries, err := state.GetAllEntries(sheet)
		if err != nil {
			return err
		}

		last := utils.GetNth(entries, len(entries)-1)
		if last == nil {
			return errors.New("no entries")
		}

		var duration time.Duration
		if last.End == nil {
			beforeLast := utils.GetNth(entries, len(entries)-2)
			if beforeLast == nil {
				return errors.New("no entry before current one")
			}
			duration = last.Start.Sub(*beforeLast.End)
		} else {
			duration = time.Now().Sub(*last.End)
		}

		fmt.Println(utils.FormatDuration(duration))
		return nil
	})

	commands.AddCommand([]string{"help"}, "show usage (of a command)", "[command]", func() error {
		if input.Note == "" {
			usage()
			return nil
		}

		cmds := commands.GetByPrefix(input.Note)
		if len(cmds) == 0 {
			return fmt.Errorf("unknown command %s", input.Note)
		} else if len(cmds) > 1 {
			return errors.New("ambigious command")
		}

		cmd := cmds[0]
		fmt.Fprintf(os.Stderr, "%s: %s\n\t%s\n", strings.Join(cmd.Names, ", "), cmd.Description, cmd.Usage)

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
