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
package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/pterm/pterm"
	"github.com/shurcooL/graphql"
	"github.com/spf13/cobra"
	"github.com/stoe/gh-report/utils"
	"gopkg.in/yaml.v2"
)

var (
	ActionsCmd = &cobra.Command{
		Use:   "actions",
		Short: "Report on GitHub Actions",
		Long: heredoc.Docf(
			`Report on GitHub Actions, requires %s scope`,
			utils.HiBlack("repo"),
		),
		RunE: GetActionsReport,
	}

	exclude       = false
	actionsReport utils.CSVReport

	ActionUsesQuery struct {
		RepositoryOwner struct {
			Repositories struct {
				PageInfo struct {
					HasNextPage bool
					EndCursor   graphql.String
				}
				Nodes []ActionUsesRepository
			} `graphql:"repositories(first: 10, after: $page, affiliations: OWNER, orderBy: { field: NAME, direction: DESC })"`
		} `graphql:"repositoryOwner(login: $owner)"`
	}

	aur []ActionUsesRepository

	ce = map[string]bool{
		".yml":  true,
		".yaml": true,
	}
)

type (
	ActionUsesRepository struct {
		Name          string
		NameWithOwner string
		Owner         Organization
		IsArchived    bool
		IsFork        bool
		Object        struct {
			Tree struct {
				Entries []struct {
					Path   string
					Name   string
					Object struct {
						Blob struct {
							Text           string
							AbbreviatedOid string
							ByteSize       int
							IsBinary       bool
							IsTruncated    bool
						} `graphql:"... on Blob"`
					}
					Extension string
					Type      string
				}
			} `graphql:"... on Tree"`
		} `graphql:"object(expression: $ref)"`
	}

	WorkflowUses struct {
		Jobs map[string]struct {
			Steps []struct {
				Uses string
			} `yaml:"steps"`
		} `yaml:"jobs,omitempty"`
	}

	ActionUsesReport struct {
		Owner     string           `json:"owner"`
		Repo      string           `json:"repo"`
		Workflows []ActionWorkflow `json:"workflows"`
	}

	ActionWorkflow struct {
		Path        string       `json:"path"`
		URL         string       `json:"url"`
		Uses        []ActionUses `json:"uses"`
		Permissions []string     `json:"permissions"`
	}

	ActionUses struct {
		Action  string `json:"action"`
		Version string `json:"version"`
		URL     string `json:"url"`
	}

	ActionPermissions struct {
		Permissions Permissions `yaml:"permissions,omitempty"`
		Jobs        map[string]struct {
			Permissions Permissions `yaml:"permissions,omitempty"`
		} `yaml:"jobs,omitempty"`
	}

	Permissions interface{}
)

func init() {
	RootCmd.AddCommand(ActionsCmd)

	ActionsCmd.Flags().BoolVar(&exclude, "exclude", false, "Exclude Github Actions authored by GitHub")
}

// GetActionsReport returns a report on GitHub Actions
func GetActionsReport(cmd *cobra.Command, args []string) (err error) {
	if repo != "" {
		return fmt.Errorf("Repository not (yet) supported for this report")
	}

	sp.Start()

	if enterprise != "" {
		variables := map[string]interface{}{
			"enterprise": graphql.String(enterprise),
			"page":       (*graphql.String)(nil),
		}

		for {
			graphqlClient.Query("OrgList", &enterpriseQuery, variables)
			organizations = append(organizations, enterpriseQuery.Enterprise.Organizations.Nodes...)

			if !enterpriseQuery.Enterprise.Organizations.PageInfo.HasNextPage {
				break
			}

			variables["page"] = &enterpriseQuery.Enterprise.Organizations.PageInfo.EndCursor
		}
	}

	if owner != "" {
		organizations = append(organizations, Organization{Login: owner})
	}

	var res = []ActionUsesReport{}

	for _, o := range organizations {
		owner = o.Login
		variables := map[string]interface{}{
			"owner": graphql.String(owner),
			"page":  (*graphql.String)(nil),
			"ref":   graphql.String("HEAD:.github/workflows"),
		}

		var i = 1
		for {
			sp.Suffix = fmt.Sprintf(
				" fetching actions report %s %s",
				utils.Cyan(owner),
				utils.HiBlack(fmt.Sprintf("(page %d)", i)),
			)

			graphqlClient.Query("ActionUses", &ActionUsesQuery, variables)
			aur = append(aur, ActionUsesQuery.RepositoryOwner.Repositories.Nodes...)

			if !ActionUsesQuery.RepositoryOwner.Repositories.PageInfo.HasNextPage {
				break
			}

			// sleep for 1 second to avoid rate limiting
			time.Sleep(1 * time.Second)

			variables["page"] = &ActionUsesQuery.RepositoryOwner.Repositories.PageInfo.EndCursor
			i++
		}

		for _, r := range aur {
			// skip if repo is archived or fork
			if r.IsArchived || r.IsFork {
				continue
			}

			// skip if repo has no workflows
			if len(r.Object.Tree.Entries) == 0 {
				continue
			}

			var wfs = []ActionWorkflow{}
			for _, e := range r.Object.Tree.Entries {
				// skip if not a yml|yaml file
				if _, ok := ce[e.Extension]; !ok {
					continue
				}

				text := e.Object.Blob.Text

				// get Action uses
				var wu WorkflowUses
				if err := yaml.Unmarshal([]byte(text), &wu); err != nil && !silent {
					fmt.Println(
						utils.Red(
							fmt.Sprintf(
								"\nerror: parsing https://github.com/%s/blob/HEAD/%s",
								r.NameWithOwner, e.Path,
							),
						),
					)
				}

				var uses []ActionUses
				for _, job := range wu.Jobs {
					for _, step := range job.Steps {
						if step.Uses != "" && excludeGitHubAuthored(step.Uses) {
							a := strings.Split(step.Uses, "@")

							var an string
							var av string
							var url string

							an = a[0]
							if len(a) == 2 {
								av = a[1]
								url = fmt.Sprintf(
									"https://github.com/%s/tree/%s",
									an,
									av,
								)
							} else {
								url = fmt.Sprintf(
									"https://github.com/%s/tree/HEAD",
									an,
								)
							}

							if strings.Contains(url, "./") {
								url = fmt.Sprintf(
									"https://github.com/%s/%s/tree/HEAD/%s",
									r.Owner.Login,
									r.Name,
									strings.ReplaceAll(an, "./", ""),
								)
							}

							uses = append(uses, ActionUses{
								Action:  an,
								Version: av,
								URL:     url,
							})
						}
					}
				}

				// get Action permissions
				var wp ActionPermissions
				if err := yaml.Unmarshal([]byte(text), &wp); err != nil && !silent {
					fmt.Println(
						utils.Red(
							fmt.Sprintf(
								"\nerror: parsing https://github.com/%s/blob/HEAD/%s",
								r.NameWithOwner, e.Path,
							),
						),
					)
				}

				var permissions []string
				// if permissions are defined at the workflow level
				if wp.Permissions != nil {
					permissions = append(permissions, getPermissions(wp.Permissions)...)
				}

				// if permissions are defined at the job level
				for _, job := range wp.Jobs {
					permissions = append(permissions, getPermissions(job.Permissions)...)
				}

				// put it all together
				wfs = append(wfs, ActionWorkflow{
					Path: e.Path,
					URL: fmt.Sprintf(
						"https://github.com/%s/%s/blob/HEAD/%s",
						r.Owner.Login,
						r.Name,
						e.Path,
					),
					Uses:        uniqueUses(uses),
					Permissions: uniquePermissions(permissions),
				})
			}

			res = append(res, ActionUsesReport{
				Owner:     r.Owner.Login,
				Repo:      r.Name,
				Workflows: wfs,
			})
		}

		// sleep for 1 second to avoid rate limiting
		time.Sleep(1 * time.Second)
	}

	sp.Stop()

	var td = pterm.TableData{
		{"owner", "repo", "workflow_path", "uses", "permissions"},
	}

	// start CSV file
	if csvPath != "" {
		actionsReport, err = utils.NewCSVReport(csvPath)

		if err != nil {
			return err
		}

		actionsReport.SetHeader([]string{"owner", "repo", "workflow_path", "uses", "permissions"})
	}

	for _, r := range res {
		for _, w := range r.Workflows {
			var data = []string{
				r.Owner,
				r.Repo,
				w.Path,
				strings.Join(usesToString(w.Uses), ", "),
				strings.Join(w.Permissions, ", "),
			}

			td = append(td, data)
			if csvPath != "" {
				actionsReport.AddData(data)
			}
		}
	}

	if !silent {
		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(td).Render()
	}

	if csvPath != "" {
		actionsReport.Save()
	}

	if jsonPath != "" {
		utils.SaveJsonReport(jsonPath, res)
	}

	return err
}

func excludeGitHubAuthored(s string) bool {
	if exclude {
		return !strings.HasPrefix(s, "actions/") && !strings.HasPrefix(s, "github/")
	}

	return true
}

func usesToString(u []ActionUses) []string {
	var s = []string{}

	for _, v := range u {
		s = append(s, fmt.Sprintf(
			"%s (%s)",
			v.Action,
			v.Version,
		))
	}

	return s
}

func getPermissions(p interface{}) []string {
	var permissions []string

	switch p := p.(type) {
	case string:
		permissions = append(permissions, p)
	case map[interface{}]interface{}:
		for k, v := range p {
			permissions = append(permissions, fmt.Sprintf("%v: %v", k, v))
		}
	}

	return permissions
}

func uniquePermissions(e []string) []string {
	r := []string{}

	for _, s := range e {
		if !containsPermissions(r[:], s) {
			r = append(r, s)
		}
	}
	return r
}

func containsPermissions(e []string, c string) bool {
	for _, s := range e {
		if s == c {
			return true
		}
	}
	return false
}

func uniqueUses(e []ActionUses) []ActionUses {
	r := []ActionUses{}

	for _, s := range e {
		if !containsUses(r[:], s) {
			r = append(r, s)
		}
	}
	return r
}

func containsUses(s []ActionUses, e ActionUses) bool {
	for _, a := range s {
		if a.Action == e.Action && a.Version == e.Version {
			return true
		}
	}
	return false
}
