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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/cli/go-gh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/stoe/gh-report/utils"
	"gopkg.in/yaml.v2"
)

var (
	exclude = false

	usesCmd = &cobra.Command{
		Use:   "uses",
		Short: "Report on GitHub Actions uses",
		Long:  "Report on GitHub Actions uses",
		RunE:  GetUses,
	}

	usesSearch Search
	usesReport utils.CSVReport
)

type UsesContent struct {
	Content string `json:"content"`
}

type WorkflowUses struct {
	Jobs map[string]interface{} `yaml:"jobs,omitempty"`
}

func init() {
	actionsCmd.AddCommand(usesCmd)

	usesCmd.Flags().BoolVar(&exclude, "exclude", false, "Exclude Github Actions authored by GitHub")
}

// GetUses returns GitHub Actions used in workflows
func GetUses(cmd *cobra.Command, args []string) (err error) {
	q := `uses in:file path:.github/workflows extension:yml language:yaml`

	switch {
	case repo != ".":
		q += fmt.Sprintf(" repo:%s/%s", owner, repo)
	default:
		q += fmt.Sprintf(" user:%s", owner)
	}

	sp.FinalMSG = fmt.Sprintf(
		"%s %s\n",
		hiBlack("search:"),
		q,
	)
	sp.Start()

	var i = 1
	var items = []SearchItems{}
	for {
		a := []string{
			"api",
			fmt.Sprintf("search/code?q=%s&page=%d&per_page=100", url.QueryEscape(q), i),
			"--cache",
			"3600s",
		}

		sp.Suffix = fmt.Sprintf(
			" Fetching GitHub Actions uses %s...",
			hiBlack(fmt.Sprintf("(page %d)", i)),
		)
		out, _, _ := gh.Exec(a...)

		json.Unmarshal(out.Bytes(), &usesSearch)
		items = append(items, usesSearch.Items...)

		if len(items) >= usesSearch.TotalCount {
			break
		}

		i++

		// sleept to not hit 30 requests per minute rate limit
		time.Sleep(3 * time.Second)
	}
	sp.Stop()

	// start CSV file
	if csvPath != "" {
		usesReport, err = utils.NewCSVReport(csvPath)

		if err != nil {
			return err
		}

		usesReport.SetHeader([]string{"owner", "repo", "workflow", "actions"})
	}

	for i, item := range items {
		var fl string
		var sl string

		if i == len(items)-1 {
			fl = "└─"
			sl = " "
		} else {
			fl = "├─"
			sl = "│"
		}

		var data UsesContent
		if err := restClient.Get(
			fmt.Sprintf(
				"repos/%s/%s/contents/%s",
				item.Repository.Owner.Login,
				item.Repository.Name,
				item.Path,
			),
			&data,
		); err != nil {
			return err
		}

		content, err := base64.StdEncoding.DecodeString(data.Content)
		if err != nil {
			return err
		}

		var wu WorkflowUses
		if err := yaml.Unmarshal(content, &wu); err != nil {
			return err
		}

		fmt.Fprintf(color.Output,
			"  %s https://github.com/%s/%s/blob/HEAD/%s\n",
			hiBlack(fl),
			item.Repository.Owner.Login,
			item.Repository.Name,
			item.Path,
		)

		var uses []string
		for _, job := range wu.Jobs {
			u := job.(map[interface{}]interface{})["uses"]
			s := job.(map[interface{}]interface{})["steps"]

			switch {
			case u == nil && s != nil:
				for _, s := range s.([]interface{}) {
					step := s.(map[interface{}]interface{})

					if step["uses"] != nil {
						if ExcludeGitHubAuthored(step["uses"].(string)) {
							uses = append(uses, step["uses"].(string))
						}
					}
				}
			case u != nil && s == nil:
				if ExcludeGitHubAuthored(u.(string)) {
					uses = append(uses, u.(string))
				}
			}
		}

		var ul string
		for j, u := range uses {
			if j == len(uses)-1 {
				ul = "└─"
			} else {
				ul = "├─"
			}

			a := strings.Split(u, "@")

			fmt.Fprintf(color.Output,
				"  %s   %s %s %s\n",
				hiBlack(sl),
				hiBlack(ul),
				a[0],
				hiBlack(fmt.Sprintf("(%s)", a[1])),
			)
		}

		if csvPath != "" {
			if len(uses) > 0 {
				usesReport.AddData([]string{
					item.Repository.Owner.Login,
					item.Repository.Name,
					item.Name,
					strings.Join(uses, ","),
				})
			}
		}
	}

	if csvPath != "" {
		if err := usesReport.Save(); err != nil {
			return err
		}

		fmt.Fprintf(color.Output, "\n%s %s\n", hiBlack("CSV saved to:"), csvPath)
	}

	return nil
}

func ExcludeGitHubAuthored(s string) bool {
	if exclude {
		return !strings.HasPrefix(s, "actions/") && !strings.HasPrefix(s, "github/")
	}

	return true
}
