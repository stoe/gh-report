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

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/spf13/cobra"
)

var (
	hostname string
	owner    string
	repo     string
	csvPath  string

	user struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	}

	restClient api.RESTClient
	// graphqlClient api.GQLClient

	rootCmd = &cobra.Command{
		Use:     "gh-report",
		Short:   "gh cli extension to generate reports",
		Long:    "gh cli extension to generate organization/user/repository reports",
		Version: "0.0.0-development",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error

			if repo == "." && owner == "" {
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

			err = restClient.Get(fmt.Sprintf("users/%s", owner), &user)

			return err
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ExitOnError(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&hostname, "hostname", "", "GitHub Enterprise Server hostname")
	rootCmd.PersistentFlags().StringVarP(&owner, "owner", "o", "", "GitHub account (organization or user account)")
	rootCmd.PersistentFlags().StringVarP(&repo, "repo", "r", ".", "GitHub repository (owner/repo)")
	rootCmd.PersistentFlags().StringVar(&csvPath, "csv", "", "Path to CSV file")
}

func initConfig() {
	var err error

	opts := api.ClientOptions{
		EnableCache: true,
		Timeout:     5 * time.Second,
	}

	if hostname != "" {
		// TODO: check if hostname is valid
		opts.Host = hostname
	}

	restClient, err = gh.RESTClient(&opts)
	// graphqlClient, err = gh.GQLClient(&opts)
	ExitOnError(err)
}

func ExitOnError(err error) {
	if err != nil {
		e := fmt.Errorf("error: %w", err)
		fmt.Println(e)
		os.Exit(1)
	}
}
