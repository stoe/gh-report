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
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/fatih/color"
	"github.com/shurcooL/graphql"
	"github.com/spf13/cobra"
)

var (
	enterprise string
	owner      string
	repo       string

	hostname string

	csvPath string

	user struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	}

	restClient    api.RESTClient
	graphqlClient api.GQLClient

	hiBlack = color.New(color.FgHiBlack).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	green   = color.New(color.FgGreen).SprintFunc()

	sp = spinner.New(spinner.CharSets[14], 40*time.Millisecond)

	rootCmd = &cobra.Command{
		Use:     "gh-report",
		Short:   "gh cli extension to generate reports",
		Long:    "gh cli extension to generate organization/user/repository reports",
		Version: "0.0.0-development",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			if enterprise != "" && owner != "" {
				return fmt.Errorf("cannot use --enterprise and --owner together")
			}

			if enterprise != "" && repo != "" {
				return fmt.Errorf("cannot use --enterprise and --repo together")
			}

			if owner != "" && repo != "" {
				return fmt.Errorf("cannot use --owner and --repo together")
			}

			if enterprise == "" && owner == "" && repo == "" {
				var r gh.Repository

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
		},
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

type Organization struct {
	Login string
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ExitOnError(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&enterprise, "enterprise", "e", "", "GitHub Enterprise Cloud account")
	rootCmd.PersistentFlags().StringVarP(&owner, "owner", "o", "", "GitHub account (organization or user account)")
	rootCmd.PersistentFlags().StringVarP(&repo, "repo", "r", "", "GitHub repository (owner/repo)")

	rootCmd.PersistentFlags().StringVar(&hostname, "hostname", "", "GitHub Enterprise Server hostname")

	rootCmd.PersistentFlags().StringVar(&csvPath, "csv", "", "Path to CSV file")
}

func initConfig() {
	opts := api.ClientOptions{
		EnableCache: true,
		Timeout:     1 * time.Hour,
	}

	if hostname != "" {
		// TODO: check if hostname is valid
		opts.Host = hostname
	}

	restClient, _ = gh.RESTClient(&opts)
	graphqlClient, _ = gh.GQLClient(&opts)
}

func ExitOnError(err error) {
	if err != nil {
		e := fmt.Errorf("error: %w", err)
		fmt.Println(e)
		os.Exit(1)
	}
}
