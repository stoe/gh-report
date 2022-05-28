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

	"github.com/fatih/color"
	"github.com/pterm/pterm"
	"github.com/shurcooL/graphql"
	"github.com/spf13/cobra"
	"github.com/stoe/gh-report/utils"
)

var (
	internal = false
	private  = false
	public   = false

	repoCmd = &cobra.Command{
		Use:   "repo",
		Short: "Report on GitHub repositories",
		Long:  "Report on GitHub repositories",
		RunE:  GetRepos,
	}

	orgRepositoriesQuery struct {
		Organization struct {
			Repositories Repos `graphql:"repositories(first: 100, after: $page, orderBy: {field: NAME, direction: ASC})"`
		} `graphql:"organization(login: $owner)"`
	}

	userRepositoriesQuery struct {
		User struct {
			Repositories Repos `graphql:"repositories(first: 100, after: $page, orderBy: {field: NAME, direction: ASC}, affiliations: OWNER)"`
		} `graphql:"user(login: $owner)"`
	}

	repositories []Repository

	repoReport utils.CSVReport
)

type Repos struct {
	PageInfo struct {
		HasNextPage bool
		EndCursor   graphql.String
	}
	Nodes []Repository
}

type Repository struct {
	Name             string
	NameWithOwner    string
	Owner            Organization
	Description      string
	URL              string
	Visibility       string
	IsArchived       bool
	IsTemplate       bool
	DefaultBranchRef struct {
		Name string
	}
	HasIssuesEnabled   bool
	HasProjectsEnabled bool
	HasWikiEnabled     bool
	IsFork             bool
	ForkCount          int
	ForkingAllowed     bool
	DiskUsage          int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func init() {
	rootCmd.AddCommand(repoCmd)

	repoCmd.Flags().BoolVar(&internal, "internal", false, "Show internal repositories only")
	repoCmd.Flags().BoolVar(&private, "private", false, "Show private repositories only")
	repoCmd.Flags().BoolVar(&public, "public", false, "Show public repositories only")
}

// GetUses returns GitHub Actions used in workflows
func GetRepos(cmd *cobra.Command, args []string) (err error) {
	if hostname != "" {
		ExitOnError(fmt.Errorf("GitHub Enterprise Server not supported for this report"))
	}

	sp.Start()

	variables := map[string]interface{}{
		"enterprise": graphql.String(enterprise),
		"page":       (*graphql.String)(nil),
	}

	if enterprise != "" {
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

	if repo != "" {
		sp.Stop()
		return fmt.Errorf("Repository not implemented")
	}

	var i = 1
	if user.Type == "User" {
		variables := map[string]interface{}{
			"owner": graphql.String(user.Login),
			"page":  (*graphql.String)(nil),
		}

		for {
			sp.Suffix = fmt.Sprintf(
				" fetching user repositories %s %s",
				user.Login,
				hiBlack(fmt.Sprintf("(page %d)", i)),
			)

			graphqlClient.Query("RepoList", &userRepositoriesQuery, variables)
			repositories = append(repositories, userRepositoriesQuery.User.Repositories.Nodes...)

			if !userRepositoriesQuery.User.Repositories.PageInfo.HasNextPage {
				break
			}

			i++

			// sleep for 1 second to avoid rate limiting
			time.Sleep(1 * time.Second)

			variables["page"] = &userRepositoriesQuery.User.Repositories.PageInfo.EndCursor
		}
	} else if user.Type == "Organization" || len(organizations) > 0 {
		for _, org := range organizations {
			variables := map[string]interface{}{
				"owner": graphql.String(org.Login),
				"page":  (*graphql.String)(nil),
			}

			for {
				sp.Suffix = fmt.Sprintf(
					" fetching organization repositories %s %s",
					org.Login,
					hiBlack(fmt.Sprintf("(page %d)", i)),
				)

				graphqlClient.Query("RepoList", &orgRepositoriesQuery, variables)
				repositories = append(repositories, orgRepositoriesQuery.Organization.Repositories.Nodes...)

				if !orgRepositoriesQuery.Organization.Repositories.PageInfo.HasNextPage {
					break
				}

				i++

				// sleep for 1 second to avoid rate limiting
				time.Sleep(1 * time.Second)

				variables["page"] = &orgRepositoriesQuery.Organization.Repositories.PageInfo.EndCursor
			}
		}
	}

	sp.Stop()

	var td = pterm.TableData{
		{"owner", "repo", "visibility", "default_branch", "fork?", "disk", "created_at", "updated_at"},
	}

	// start CSV file
	if csvPath != "" {
		repoReport, err = utils.NewCSVReport(csvPath)

		if err != nil {
			return err
		}

		repoReport.SetHeader([]string{"owner", "repo", "visibility", "default_branch", "fork?", "disk", "created_at", "updated_at"})
	}

	for _, repo := range repositories {
		if internal && repo.Visibility != "INTERNAL" {
			continue
		}
		if private && repo.Visibility != "PRIVATE" {
			continue
		}
		if public && repo.Visibility != "PUBLIC" {
			continue
		}

		var data = []string{
			repo.Owner.Login,
			repo.Name,
			strings.ToLower(repo.Visibility),
			repo.DefaultBranchRef.Name,
			fmt.Sprintf("%t", repo.IsFork),
			fmt.Sprintf("%d", repo.DiskUsage),
			repo.CreatedAt.Format("2006-01-02 15:04:05"),
			repo.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		td = append(td, data)

		if csvPath != "" {
			repoReport.AddData(data)
		}
	}

	pterm.DefaultTable.WithHasHeader().WithData(td).Render()

	if csvPath != "" {
		if err := repoReport.Save(); err != nil {
			return err
		}

		fmt.Fprintf(color.Output, "\n%s %s\n", hiBlack("CSV saved to:"), csvPath)
	}

	return err
}
