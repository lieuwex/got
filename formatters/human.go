package formatters

import (
	"fmt"
	"got/types"
	"got/utils"
	"io"
	"text/tabwriter"
	"time"
)

type Human struct{}

func (*Human) Write(out io.Writer, f *types.FormatterInput) error {
	fmt.Printf("Timesheet: %s\n", f.Sheet)
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)

	fmt.Fprintln(w, "Id\tDay\tStart      End\tDuration\tNotes")

	var date time.Time
	var dayDuration time.Duration
	var dateString string

	for i, entry := range f.Entries {
		end := ""
		if entry.End != nil {
			end = entry.End.Format("15:04:05")
		}

		duration, _ := entry.Duration()
		dayDuration += duration

		fmt.Fprintf(
			w,
			"%d\t%s\t%s - %s\t%s\t%s\n",
			entry.ID,
			dateString,
			entry.Start.Format("15:04:05"),
			end,
			utils.FormatDuration(duration),
			entry.Note,
		)

		var next *types.Entry
		if i+1 < len(f.Entries) {
			next = f.Entries[i+1]
		}

		if next == nil || !utils.SameDate(next.Start, date) {
			// new day

			if i > 0 {
				fmt.Fprintf(
					w,
					"\t\t\t%s\t\n",
					utils.FormatDuration(dayDuration),
				)
			}

			if next != nil {
				date = next.Start
			}

			dayDuration = 0
			dateString = entry.Start.Format("Mon Jan 2, 2006")
		} else {
			dateString = ""
		}
	}

	return w.Flush()
}
