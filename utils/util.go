package utils

import (
	"fmt"
	"got/types"
	"os"
	"strings"
	"time"
)

func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	h := d / time.Hour
	d -= h * time.Hour

	m := d / time.Minute
	d -= m * time.Minute

	s := d / time.Second

	return fmt.Sprintf("%01d:%02d:%02d", h, m, s)
}

func SameDate(a, b time.Time) bool {
	yA, mA, dA := a.Date()
	yB, mB, dB := b.Date()

	return yA == yB && mA == mB && dA == dB
}

func FormatDate(date time.Time) string {
	return date.Format("2006-01-02 15:04:05.000000")
}
func ParseDate(str string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05.000000", str)
}

func Confirm(prompt string, defaultValue bool) bool {
	var hint string
	if defaultValue {
		hint = "Y/n"
	} else {
		hint = "y/N"
	}

	defer func() {
		fmt.Fprintln(os.Stderr, "")
	}()

	for {
		fmt.Fprintf(
			os.Stderr,
			"%s (%s) ",
			prompt,
			hint,
		)

		var str string
		if _, err := fmt.Scanf("%s", &str); err != nil {
			return defaultValue
		}

		lower := strings.ToLower(str)
		if lower[0] == 'y' || lower[0] == 'n' {
			return lower[0] == 'y'
		} else if lower == "" {
			return defaultValue
		}
	}
}

func SumDuration(entries []*types.Entry, fn func(*types.Entry) bool) time.Duration {
	var res time.Duration

	for _, entry := range entries {
		if fn(entry) {
			duration, _ := entry.Duration()
			res += duration
		}
	}

	return res
}

func GetNth(entries []*types.Entry, i int) *types.Entry {
	if i >= 0 && i < len(entries) {
		return entries[i]
	}

	return nil
}
