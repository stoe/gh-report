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
	"fmt"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/shurcooL/graphql"
	"github.com/spf13/cobra"
	"github.com/stoe/gh-report/utils"
	"gopkg.in/yaml.v2"
)

var (
	actionsCmd = &cobra.Command{
		Use:   "actions",
		Short: "Report on GitHub Actions",
		Long:  "Report on GitHub Actions",
		RunE:  GetActionsReport,
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
		Jobs map[string]interface{} `yaml:"jobs,omitempty"`
	}

	ActionUsesReport struct {
		Owner     string           `json:"owner"`
		Repo      string           `json:"repo"`
		Workflows []ActionWorkflow `json:"workflows"`
	}

	ActionWorkflow struct {
		Path        string       `json:"path"`
		Uses        []ActionUses `json:"uses"`
		Permissions []string     `json:"permissions"`
	}

	ActionUses struct {
		Action  string `json:"action"`
		Version string `json:"version"`
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
	rootCmd.AddCommand(actionsCmd)

	actionsCmd.Flags().BoolVar(&exclude, "exclude", false, "Exclude Github Actions authored by GitHub")
}

// GetActionsReport returns a report on GitHub Actions
func GetActionsReport(cmd *cobra.Command, args []string) (err error) {
	if hostname != "" {
		return fmt.Errorf("GitHub Enterprise Server not (yet) supported for this report")
	}

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
				cyan(owner),
				hiBlack(fmt.Sprintf("(page %d)", i)),
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

				text := []byte(e.Object.Blob.Text)

				var wu WorkflowUses
				if err := yaml.Unmarshal(text, &wu); err != nil {
					fmt.Println(
						"WorkflowUses",
						r.NameWithOwner,
						e.Path,
					)

					return err
				}

				var wp ActionPermissions
				if err := yaml.Unmarshal(text, &wp); err != nil {
					fmt.Println(
						"ActionPermissions",
						r.NameWithOwner,
						e.Path,
					)

					return err
				}

				var uses []ActionUses
				for _, job := range wu.Jobs {
					u := job.(map[interface{}]interface{})["uses"]
					s := job.(map[interface{}]interface{})["steps"]

					switch {
					case u == nil && s != nil:
						for _, s := range s.([]interface{}) {
							step := s.(map[interface{}]interface{})

							if step["uses"] != nil {
								if ExcludeGitHubAuthored(step["uses"].(string)) {
									a := strings.Split(step["uses"].(string), "@")

									var an string
									var av string

									an = a[0]
									if len(a) == 2 {
										av = a[1]
									}

									uses = append(uses, ActionUses{
										Action:  an,
										Version: av,
									})
								}
							}
						}
					case u != nil && s == nil:
						if ExcludeGitHubAuthored(u.(string)) {
							a := strings.Split(u.(string), "@")

							var an string
							var av string

							an = a[0]
							if len(a) == 2 {
								av = a[1]
							}

							uses = append(uses, ActionUses{
								Action:  an,
								Version: av,
							})
						}
					}
				}

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

				wfs = append(wfs, ActionWorkflow{
					Path:        e.Path,
					Uses:        uses,
					Permissions: t,
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
				strings.Join(UsesToString(w.Uses), ", "),
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

func ExcludeGitHubAuthored(s string) bool {
	if exclude {
		return !strings.HasPrefix(s, "actions/") && !strings.HasPrefix(s, "github/")
	}

	return true
}

func UsesToString(u []ActionUses) []string {
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
