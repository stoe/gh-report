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
	permissionsCmd = &cobra.Command{
		Use:   "permissions",
		Short: "Report on GitHub Actions permissions",
		Long:  "Report on GitHub Actions permissions",
		RunE:  GetPermissions,
	}

	permissionsSearch Search
	permissionsReport *utils.Report
)

type PermissionsContent struct {
	Content string `json:"content"`
}

type WorkflowPermissions struct {
	Permissions Permissions `yaml:"permissions,omitempty"`
	Jobs        map[string]struct {
		Permissions Permissions `yaml:"permissions,omitempty"`
	} `yaml:"jobs,omitempty"`
}

type Permissions interface{}

func init() {
	actionsCmd.AddCommand(permissionsCmd)
}

// GetPermissions returns if permissions are set in GitHub Actions workflows
func GetPermissions(cmd *cobra.Command, args []string) (err error) {
	q := `GITHUB_TOKEN in:file path:.github/workflows extension:yml language:yaml`

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
			" Fetching GitHub Actions permissions %s...",
			hiBlack(fmt.Sprintf("(page %d)", i)),
		)
		out, _, _ := gh.Exec(a...)

		json.Unmarshal(out.Bytes(), &permissionsSearch)
		items = append(items, permissionsSearch.Items...)

		if len(items) >= permissionsSearch.TotalCount {
			break
		}

		i++

		// sleept to not hit 30 requests per minute rate limit
		time.Sleep(3 * time.Second)
	}
	sp.Stop()

	if err != nil {
		return err
	}

	// start CSV file
	if csvPath != "" {
		permissionsReport, err = utils.NewCSVReport(csvPath)

		if err != nil {
			return err
		}

		permissionsReport.SetHeader([]string{"owner", "repo", "workflow", "permissions"})
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

		var data PermissionsContent
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

		var wp WorkflowPermissions
		if err := yaml.Unmarshal(content, &wp); err != nil {
			return err
		}

		fmt.Fprintf(color.Output,
			"  %s https://github.com/%s/%s/blob/HEAD/%s\n",
			hiBlack(fl),
			item.Repository.Owner.Login,
			item.Repository.Name,
			item.Path,
		)

		var t []string
		if wp.Permissions != nil {
			switch wp.Permissions.(type) {
			case string:
				t = []string{wp.Permissions.(string)}
			case map[interface{}]interface{}:
				for g, h := range wp.Permissions.(map[interface{}]interface{}) {
					t = append(t, fmt.Sprintf("%v: %v", g, h))
				}
			}
		}

		for _, job := range wp.Jobs {
			switch job.Permissions.(type) {
			case string:
				t = append(t, job.Permissions.(string))
			case map[interface{}]interface{}:
				for k, v := range job.Permissions.(map[interface{}]interface{}) {
					t = append(t, fmt.Sprintf("%v: %v", k, v))
				}
			}
		}

		var ul string
		if len(t) > 0 {
			for j := range t {
				if j == len(t)-1 {
					ul = "└─"
				} else {
					ul = "├─"
				}
			}

		} else {
			ul = "└─"
		}
		printSecondLine(sl, ul, t)

		if csvPath != "" {
			permissionsReport.AddData([]string{
				item.Repository.Owner.Login,
				item.Repository.Name,
				item.Name,
				strings.Join(t, ","),
			})
		}
	}

	if csvPath != "" {
		if err := permissionsReport.Save(); err != nil {
			return err
		}

		fmt.Fprintf(color.Output, "\n%s %s\n", hiBlack("CSV saved to:"), csvPath)
	}

	return nil
}

func printSecondLine(sl, ul string, res []string) {
	msg := green(strings.Join(res, ", "))

	if len(res) < 1 {
		msg = red("n/a")
	}

	fmt.Fprintf(color.Output,
		"  %s   %s %s\n",
		hiBlack(sl),
		hiBlack(ul),
		msg,
	)
}
