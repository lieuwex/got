package formatters

import (
	"encoding/json"
	"got/types"
	"got/utils"
	"io"
	"time"
)

type outputEntry struct {
	Id       uint64     `json:"id"`
	Start    time.Time  `json:"start"`
	End      *time.Time `json:"end"`
	Note     string     `json:"note"`
	Duration string     `json:"duration"`
}

type outputSheet struct {
	Name      string        `json:"name"`
	SheetTime string        `json:"sheet_time"`
	Entries   []outputEntry `json:"entries"`
}

type output struct {
	Sheets    []outputSheet `json:"sheets"`
	TotalTime string        `json:"total_time"`
}

type JSON struct{}

func (JSON) Write(out io.Writer, f *types.FormatterInput) error {
	var res output

	sheets := make(map[string]*outputSheet)

	for _, entry := range f.Entries {
		sheetName := entry.Sheet

		if _, has := sheets[sheetName]; !has {
			sheetTime := utils.SumDuration(f.Entries, func(x *types.Entry) bool {
				return x.Sheet == sheetName
			})

			sheets[sheetName] = &outputSheet{
				Name:      sheetName,
				SheetTime: utils.FormatDuration(sheetTime),
			}
		}

		entryDuration, _ := entry.Duration()

		sheet := sheets[sheetName]
		sheet.Entries = append(sheet.Entries, outputEntry{
			Id:       entry.ID,
			Start:    entry.Start,
			End:      entry.End,
			Note:     entry.Note,
			Duration: utils.FormatDuration(entryDuration),
		})
	}

	for _, sheet := range sheets {
		res.Sheets = append(res.Sheets, *sheet)
	}

	totalTime := utils.SumDuration(f.Entries, func(*types.Entry) bool { return true })
	res.TotalTime = utils.FormatDuration(totalTime)

	enc := json.NewEncoder(out)
	return enc.Encode(res)
}
