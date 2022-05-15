/*
Copyright © 2021 Stefan Stölzle <stefan@stoelzle.me>

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
package cmd

import (
	"github.com/spf13/cobra"
)

var (
	actionsCmd = &cobra.Command{
		Use:   "actions",
		Short: "Report on GitHub Actions [permissions|uses]",
		Long:  "Report on GitHub Actions [permissions|uses]",
	}
)

type Search struct {
	TotalCount        int           `json:"total_count"`
	IncompleteResults bool          `json:"incomplete_results"`
	Items             []SearchItems `json:"items"`
}

type SearchItems struct {
	Name       string  `json:"name"`
	Path       string  `json:"path"`
	Score      float32 `json:"score"`
	Repository struct {
		Private bool   `json:"private"`
		Name    string `json:"name"`
		Owner   struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	SHA string `json:"sha"`
}

type Content struct {
	Content string `json:"content"`
}

func init() {
	rootCmd.AddCommand(actionsCmd)
}
