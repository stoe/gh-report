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
	"github.com/stoe/gh-report/internal/utils"
)

var (
	VerifiedEmailsCmd = &cobra.Command{
		Use:   "verified-emails",
		Short: "List enterprise/organization members' verified emails",
		Long: heredoc.Docf(
			`List enterprise/organization members' verified emails, requires %[1]suser:email%[1]s, %[1]sread:enterprise%[1]s and/or %[1]sread:org%[1]s scope`,
			"`",
		),
		Aliases: []string{"emails", "email"},
		RunE:    GetUserEmails,
	}

	memberQuery struct {
		Organization struct {
			MembersWithRole struct {
				PageInfo struct {
					HasNextPage bool
					EndCursor   graphql.String
				}
				TotalCount int
				Nodes      []memberDetails
			} `graphql:"membersWithRole(first: 100, after: $page)"`
		} `graphql:"organization(login: $org)"`
	}

	members []memberDetails

	emailReport utils.CSVReport

	mdEmailReport = `# GitHub Emails Report

| Login | Name | Email | Verified Domain Emails |
| --- | --- | --- | --- |
{{ range . }}| {{ .Login }} | {{ .Name }} | {{ .Email }} | {{ range $i, $v := .VerifiedDomainEmails }}{{ if $i }}<br/>{{ end }}{{ $v }}{{ end }} |
{{ end }}
`
)

type (
	memberDetails struct {
		Login                            string
		Name                             string
		Email                            string
		CreatedAt                        time.Time
		UpdatedAt                        time.Time
		OrganizationVerifiedDomainEmails []string `graphql:"organizationVerifiedDomainEmails(login: $org)"`
	}

	VerifiedEmailsJSON struct {
		Login                string   `json:"login"`
		Name                 string   `json:"name"`
		Email                string   `json:"email"`
		VerifiedDomainEmails []string `json:"verified_emails"`
	}
)

func init() {
	RootCmd.AddCommand(VerifiedEmailsCmd)
}

func GetUserEmails(cmd *cobra.Command, args []string) (err error) {
	if repo != "" {
		return fmt.Errorf("Repository not supported for this report")
	}

	if user.Type == "User" {
		return fmt.Errorf("%s not supported for this report", user.Type)
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

			// sleep for 1 second to avoid rate limiting
			time.Sleep(1 * time.Second)

			variables["page"] = &enterpriseQuery.Enterprise.Organizations.PageInfo.EndCursor
		}
	}

	if owner != "" {
		organizations = append(organizations, Organization{Login: owner})
	}

	for _, org := range organizations {
		variables := map[string]interface{}{
			"org":  graphql.String(org.Login),
			"page": (*graphql.String)(nil),
		}

		var i = 1
		for {
			sp.Suffix = fmt.Sprintf(
				" fetching verified emails report %s %s",
				utils.Cyan(org.Login),
				utils.HiBlack(fmt.Sprintf("(page %d)", i)),
			)

			graphqlClient.Query("MemberList", &memberQuery, variables)
			members = append(members, memberQuery.Organization.MembersWithRole.Nodes...)

			if !memberQuery.Organization.MembersWithRole.PageInfo.HasNextPage {
				break
			}

			i++

			// sleep for 1 second to avoid rate limiting
			time.Sleep(1 * time.Second)

			variables["page"] = &memberQuery.Organization.MembersWithRole.PageInfo.EndCursor
		}
	}

	sp.Stop()

	var td = pterm.TableData{
		{"login", "full_name", "email", "verified_emails"},
	}

	// start CSV file
	if csvPath != "" {
		emailReport, err = utils.NewCSVReport(csvPath)

		if err != nil {
			return err
		}

		emailReport.SetHeader([]string{"login", "full_name", "email", "verified_emails"})
	}

	var verifiedEmails = make(map[string][]string)
	var res []VerifiedEmailsJSON

	for _, member := range members {
		var data = []string{
			member.Login,
			member.Name,
			member.Email,
			strings.Join(member.OrganizationVerifiedDomainEmails, ","),
		}

		if _, ok := verifiedEmails[member.Login]; !ok {
			verifiedEmails[member.Login] = data

			td = append(td, data)

			if csvPath != "" {
				emailReport.AddData(data)
			}

			res = append(res, VerifiedEmailsJSON{
				Login:                member.Login,
				Name:                 member.Name,
				Email:                member.Email,
				VerifiedDomainEmails: member.OrganizationVerifiedDomainEmails,
			})
		}
	}

	if !silent {
		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(td).Render()
	}

	if csvPath != "" {
		emailReport.Save()
	}

	if jsonPath != "" {
		err = utils.SaveJsonReport(jsonPath, res)
	}

	if mdPath != "" {
		err = utils.SaveMDReport(mdPath, mdEmailReport, res)
	}

	return err
}
