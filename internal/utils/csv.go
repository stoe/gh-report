/*
Copyright © 2023 Stefan Stölzle <stefan@stoelzle.me>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package utils

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/fatih/color"
)

// Report is the interface that wraps report information methods.
type (
	CSVReport interface {
		SetHeader([]string)
		AddData([]string)
		Save() error
	}

	// report is the implementation of Report
	Report struct {
		path   string
		file   *os.File
		writer *csv.Writer
	}
)

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

	fmt.Fprintf(color.Output, "%s %s\n", HiBlack("CSV saved to:"), r.path)

	return nil
}
