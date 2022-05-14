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
	"github.com/shurcooL/graphql"
	"github.com/spf13/cobra"
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

	enterpriseQuery struct {
		Enterprise struct {
			Organizations struct {
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				}
				Nodes []Organization
			} `graphql:"organizations(first: 100, after: $page)"`
		} `graphql:"enterprise(slug: $enterprise)"`
	}

	repositoriesQuery struct {
		Organization struct {
			Repositories struct {
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				}
				Nodes []Repository
			} `graphql:"repositories(first: 100, after: $page)"`
		} `graphql:"organization(login: $owner)"`
	}

	organizations []Organization
	repositories  []Repository
)

type Organization struct {
	Login string
}

type Repository struct {
	Name          string
	NameWithOwner string
	Description   string
	URL           string
	Visibility    string
	IsArchived    bool
	IsTemplate    bool

	DefaultBranchRef struct {
		Name string
	}

	HasIssuesEnabled   bool
	HasProjectsEnabled bool
	HasWikiEnabled     bool

	IsFork         bool
	ForkCount      int
	ForkingAllowed bool

	DiskUsage int

	CreatedAt time.Time
	UpdatedAt time.Time
}

func init() {
	rootCmd.AddCommand(repoCmd)

	repoCmd.Flags().BoolVar(&internal, "internal", false, "Show internal repositories only")
	repoCmd.Flags().BoolVar(&private, "private", false, "Show private repositories only")
	repoCmd.Flags().BoolVar(&public, "public", false, "Show public repositories only")
}

// GetUses returns GitHub Actions used in workflows
func GetRepos(cmd *cobra.Command, args []string) (err error) {
	e := enterprise

	if e == "" {
		e = owner
	}

	fmt.Printf(
		"%s\n",
		hiBlack(
			fmt.Sprintf("garthering repositories for %s...", e),
		),
	)

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

	if user.Type == "User" {
		sp.Stop()
		return fmt.Errorf("%s not implemented", user.Type)
	}

	for _, org := range organizations {
		variables := map[string]interface{}{
			"owner": graphql.String(org.Login),
			"page":  (*graphql.String)(nil),
		}

		for {
			graphqlClient.Query("RepoList", &repositoriesQuery, variables)
			repositories = append(repositories, repositoriesQuery.Organization.Repositories.Nodes...)

			if !repositoriesQuery.Organization.Repositories.PageInfo.HasNextPage {
				break
			}

			variables["page"] = &repositoriesQuery.Organization.Repositories.PageInfo.EndCursor
		}
	}

	sp.Stop()

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

		fmt.Fprintf(color.Output,
			"%s %s\n",
			repo.NameWithOwner,
			hiBlack(
				fmt.Sprintf("[%s]", strings.ToLower(repo.Visibility)),
			),
		)
	}

	return err
}
