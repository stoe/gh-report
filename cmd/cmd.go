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
	"os"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/briandowns/spinner"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/pkg/repository"
	"github.com/shurcooL/graphql"
	"github.com/spf13/cobra"
	"github.com/stoe/gh-report/internal/utils"
)

var (
	noCache = false
	silent  = false

	enterprise string
	owner      string
	repo       string

	token    string
	hostname string

	csvPath  string
	jsonPath string
	mdPath   string

	user struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	}

	restClient    api.RESTClient
	graphqlClient api.GQLClient

	sp = spinner.New(spinner.CharSets[14], 40*time.Millisecond)

	RootCmd = &cobra.Command{
		Use:               "report",
		Short:             "gh cli extension to generate reports",
		Long:              "gh cli extension to generate enterprise/organization/user/repository reports",
		Version:           "2.2.1",
		PersistentPreRunE: run,
	}

	enterpriseQuery struct {
		Enterprise struct {
			Organizations struct {
				PageInfo struct {
					HasNextPage bool
					EndCursor   graphql.String
				}
				Nodes []Organization
			} `graphql:"organizations(first: 100, after: $page, orderBy: {field: LOGIN, direction: ASC})"`
		} `graphql:"enterprise(slug: $enterprise)"`
	}

	organizations []Organization
)

type (
	Organization struct {
		Login string
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		ExitOnError(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false,
		`Do not cache results for one hour (default "false")`,
	)
	RootCmd.PersistentFlags().BoolVar(&silent, "silent", false, `Do not print any output (default: "false")`)

	RootCmd.PersistentFlags().StringVarP(
		&enterprise, "enterprise", "e", "",
		heredoc.Docf(
			`GitHub Enterprise Cloud account (requires %[1]sread:enterprise%[1]s scope)`,
			"`",
		),
	)
	RootCmd.PersistentFlags().StringVarP(
		&owner, "owner", "o", "",
		heredoc.Docf(
			`GitHub account organization (requires %[1]sread:org%[1]s scope) or user account (requires %[1]sn/a%[1]s scope)`,
			"`",
		),
	)
	RootCmd.PersistentFlags().StringVarP(
		&repo, "repo", "r", "",
		heredoc.Docf(
			`GitHub repository (owner/repo), requires %[1]srepo%[1]s scope`,
			"`",
		),
	)

	RootCmd.PersistentFlags().StringVarP(
		&token, "token", "t", "",
		`GitHub Personal Access Token (default "gh auth token")`,
	)
	RootCmd.PersistentFlags().StringVar(&hostname, "hostname", "github.com", "GitHub Enterprise Server hostname")

	RootCmd.PersistentFlags().StringVar(&csvPath, "csv", "", "Path to CSV file, to save report to file")
	RootCmd.PersistentFlags().StringVar(&jsonPath, "json", "", "Path to JSON file, to save report to file")
	RootCmd.PersistentFlags().StringVar(&mdPath, "md", "", "Path to MD file, to save report to file")

	RootCmd.MarkFlagsMutuallyExclusive("enterprise", "owner")
	RootCmd.MarkFlagsMutuallyExclusive("enterprise", "repo")
	RootCmd.MarkFlagsMutuallyExclusive("owner", "repo")
}

func initConfig() {
	cache := time.Hour

	if noCache {
		cache = 0
	}

	opts := api.ClientOptions{
		EnableCache: !noCache,
		CacheTTL:    cache,
		Host:        hostname,
	}

	if token != "" {
		opts.AuthToken = token
	} else {
		t, _ := auth.TokenForHost(hostname)
		opts.AuthToken = t
	}

	restClient, _ = gh.RESTClient(&opts)
	graphqlClient, _ = gh.GQLClient(&opts)
}

func run(cmd *cobra.Command, args []string) (err error) {
	if enterprise == "" && owner == "" && repo == "" {
		var r repository.Repository

		r, err = gh.CurrentRepository()

		if err != nil {
			return err
		}

		owner = r.Owner()
		repo = r.Name()
	} else if strings.Contains(repo, "/") && owner == "" {
		r := strings.Split(repo, "/")

		owner = r[0]
		repo = r[1]
	}

	if owner != "" || repo != "" {
		err = restClient.Get(fmt.Sprintf("users/%s", owner), &user)
	}

	return err
}

func ExitOnError(err error) {
	if err != nil {
		RootCmd.PrintErrln(utils.Red(err.Error()))
		os.Exit(1)
	}
}
