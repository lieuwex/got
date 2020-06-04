package types

import "io"

type FormatterInput struct {
	Sheet   string
	Entries []*Entry
}

type Formatter interface {
	Write(w io.Writer, input *FormatterInput) error
}
