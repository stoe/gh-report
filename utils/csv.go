package utils

import (
	"encoding/csv"
	"os"
)

// Report is the interface that wraps report information methods.
type CSVReport interface {
	SetHeader([]string)
	AddData([]string)
	Save() error
}

// report is the implementation of Report
type Report struct {
	path   string
	file   *os.File
	writer *csv.Writer
}

// New returns a new Report instance.
func NewCSVReport(p string) (*Report, error) {
	file, err := os.Create(p)

	if err != nil {
		return nil, err
	}

	return &Report{
		path:   p,
		file:   file,
		writer: csv.NewWriter(file),
	}, nil
}

func (r *Report) SetPath(p string) {
	r.path = p
}

func (r *Report) SetHeader(h []string) {
	r.writer.Write(h)
}

func (r *Report) AddData(d []string) {
	r.writer.Write(d)
	r.writer.Flush()
}

func (r *Report) Save() error {
	r.file.Close()

	return nil
}
