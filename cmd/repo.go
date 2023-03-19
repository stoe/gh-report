/*
Copyright © 2022 Stefan Stölzle <stefan@stoelzle.me>

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
)

var (
	internal = false
	private  = false
	public   = false

	RepoCmd = &cobra.Command{
		Use:   "repo",
		Short: "Report on GitHub repositories",
		Long:  "Report on GitHub repositories",
		RunE:  GetRepos,
		Aliases: []string{"repos"},
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

type (
	Repos struct {
		PageInfo struct {
			HasNextPage bool
			EndCursor   graphql.String
		}
		Nodes []Repository
	}

	Repository struct {
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

	RepoReportJSON struct {
		Owner         string    `json:"owner"`
		Repo          string    `json:"repo"`
		Visibility    string    `json:"visibility"`
		Archived      bool      `json:"is_archived"`
		Fork          bool      `json:"is_fork"`
		DefaultBranch string    `json:"default_branch"`
		Disk          int       `json:"disk_usage"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
	}
)

func init() {
	RootCmd.AddCommand(RepoCmd)

	RepoCmd.Flags().BoolVar(&internal, "internal", false, "Show internal repositories only")
	RepoCmd.Flags().BoolVar(&private, "private", false, "Show private repositories only")
	RepoCmd.Flags().BoolVar(&public, "public", false, "Show public repositories only")
}

// GetUses returns GitHub Actions used in workflows
func GetRepos(cmd *cobra.Command, args []string) (err error) {
	if repo != "" {
		return fmt.Errorf("Repository not (yet) supported for this report")
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

	var i = 1
	if user.Type == "User" {
		variables := map[string]interface{}{
			"owner": graphql.String(user.Login),
			"page":  (*graphql.String)(nil),
		}

		for {
			sp.Suffix = fmt.Sprintf(
				" fetching repositories report %s %s",
				utils.Cyan(user.Login),
				utils.HiBlack(fmt.Sprintf("(page %d)", i)),
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
					" fetching repositories report %s %s",
					utils.Cyan(org.Login),
					utils.HiBlack(fmt.Sprintf("(page %d)", i)),
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
		{"owner", "repo", "visibility", "archived?", "fork?", "default_branch", "disk", "created_at", "updated_at"},
	}

	// start CSV file
	if csvPath != "" {
		repoReport, err = utils.NewCSVReport(csvPath)

		if err != nil {
			return err
		}

		repoReport.SetHeader([]string{"owner", "repo", "visibility", "archived?", "fork?", "default_branch", "disk", "created_at", "updated_at"})
	}

	var res []RepoReportJSON

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
			fmt.Sprintf("%t", repo.IsArchived),
			fmt.Sprintf("%t", repo.IsFork),
			repo.DefaultBranchRef.Name,
			fmt.Sprintf("%d", repo.DiskUsage),
			repo.CreatedAt.Format("2006-01-02"),
			repo.UpdatedAt.Format("2006-01-02"),
		}

		res = append(res, RepoReportJSON{
			Owner:         repo.Owner.Login,
			Repo:          repo.Name,
			Visibility:    strings.ToLower(repo.Visibility),
			Archived:      repo.IsArchived,
			Fork:          repo.IsFork,
			DefaultBranch: repo.DefaultBranchRef.Name,
			Disk:          repo.DiskUsage,
			CreatedAt:     repo.CreatedAt,
			UpdatedAt:     repo.UpdatedAt,
		})

		td = append(td, data)

		if csvPath != "" {
			repoReport.AddData(data)
		}
	}

	if !silent {
		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(td).Render()
	}

	if csvPath != "" {
		repoReport.Save()
	}

	if jsonPath != "" {
		utils.SaveJsonReport(jsonPath, res)
	}

	return err
}
