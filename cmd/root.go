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
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/cli/go-gh/pkg/auth"
	"github.com/cli/go-gh/pkg/repository"
	"github.com/fatih/color"
	"github.com/shurcooL/graphql"
	"github.com/spf13/cobra"
)

var (
	noCache = false
	silent  = false

	enterprise string
	owner      string
	repo       string

	token    string
	hostname = "github.com"

	csvPath  string
	jsonPath string

	user struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	}

	restClient    api.RESTClient
	graphqlClient api.GQLClient

	bold    = color.New(color.Bold).SprintFunc()
	hiBlack = color.New(color.FgHiBlack).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()

	sp = spinner.New(spinner.CharSets[14], 40*time.Millisecond)

	RootCmd = &cobra.Command{
		Use:               "gh-report",
		Short:             "gh cli extension to generate reports",
		Long:              `gh cli extension to generate enterprise/organization/user/repository reports`,
		Version:           "2.1.0",
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

	RootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "do not cache results for one hour (default: false)")
	RootCmd.PersistentFlags().BoolVar(&silent, "silent", false, "do not print any output (default: false)")

	RootCmd.PersistentFlags().StringVarP(&enterprise, "enterprise", "e", "", "GitHub Enterprise Cloud account")
	RootCmd.PersistentFlags().StringVarP(&owner, "owner", "o", "", "GitHub account (organization or user account)")
	RootCmd.PersistentFlags().StringVarP(&repo, "repo", "r", "", "GitHub repository (owner/repo)")

	RootCmd.PersistentFlags().StringVar(&token, "token", "", "GitHub Personal Access Token (default: gh auth token)")
	RootCmd.PersistentFlags().StringVar(&hostname, "hostname", "", "GitHub Enterprise Server hostname")

	RootCmd.PersistentFlags().StringVar(&csvPath, "csv", "", "Path to CSV file")
	RootCmd.PersistentFlags().StringVar(&jsonPath, "json", "", "Path to JSON file")

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
	}

	if token != "" {
		opts.AuthToken = token
	} else {
		t, _ := auth.TokenForHost(hostname)
		opts.AuthToken = t
	}

	if hostname != "" {
		opts.Host = strings.ToLower(hostname)
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
		RootCmd.PrintErrln(red(err.Error()))
		os.Exit(1)
	}
}
