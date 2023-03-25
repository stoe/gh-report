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

	"github.com/MakeNowJust/heredoc"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/stoe/gh-report/internal/utils"
)

var (
	LicenseCmd = &cobra.Command{
		Use:   "license",
		Short: "Report on GitHub Enterprise licensing",
		Long: heredoc.Docf(
			`Report on GitHub Enterprise licensing, requires %s and %s scope`,
			utils.HiBlack("read:enterprise"),
			utils.HiBlack("user:email"),
		),
		RunE: GetLicensing,
	}

	licenseData   LicenseData
	licenseReport utils.CSVReport

	mdLicenseReport = `# GitHub License Report

**Purchased**: {{ .Purchased }}
**Consumed**: {{ .Consumed }}
**Free**: {{ .Free }}

## Users

| Login | Name | Verified Emails | License Type | GitHub Enterprise Cloud User | GitHub Enterprise Server User | Visual Studio User | Accounts |
| --- | --- | --- | --- | --- | --- | --- | --: |
{{ range .Users }}| {{ .Login }} | {{ .Name }} | {{ range $i, $v := .VerifiedDomainEmails }}{{ if $i }}<br/>{{ end }}{{ $v }}{{ end }} | {{ .LicenseType }} | ` + "`" + `{{ .GHEC }}` + "`" + ` | ` + "`" + `{{ .GHES }}` + "`" + ` | ` + "`" + `{{ .VSS }}` + "`" + ` | {{ .Accounts }} |
{{ end }}
`
)

func init() {
	RootCmd.AddCommand(LicenseCmd)
}

type (
	LicenseData struct {
		TotalSeatsConsumed  int `json:"total_seats_consumed"`
		TotalSeatsPurchased int `json:"total_seats_purchased"`
		Users               []struct {
			LicesneType       string `json:"license_type"`
			TotalUserAccounts int    `json:"total_user_accounts"`
			// GitHub Enterprise Cloud fields
			DotcomUser                 bool     `json:"github_com_user"`
			DotcomLogin                string   `json:"github_com_login"`
			DotcomName                 string   `json:"github_com_name"`
			DotcomProfile              string   `json:"github_com_profile"`
			DotcomMemberRoles          []string `json:"github_com_member_roles"`
			DotcomEnterpriseRole       string   `json:"github_com_enterprise_role"`
			DotcomVerifiedDomainEmails []string `json:"github_com_verified_domain_emails"`
			DotcomSamlNameID           string   `json:"github_com_saml_name_id"`
			DotcomOrgsPendingInvites   []string `json:"github_com_orgs_with_pending_invites"`
			// GitHub Enterprise Server fields
			ServerUser    bool     `json:"enterprise_server_user"`
			ServerEmails  []string `json:"enterprise_server_emails"`
			ServerUserIDs []string `json:"enterprise_server_user_ids"`
			// VisualStudio Subscription fields
			VSSUser  bool   `json:"visual_studio_subscription_user"`
			VSSEmail string `json:"visual_studio_subscription_email"`
		} `json:"users"`
	}

	LicenseReportJSON struct {
		Purchased int           `json:"purchased"`
		Consumed  int           `json:"consumed"`
		Free      int           `json:"free"`
		Users     []LicenseUser `json:"users"`
	}

	LicenseUser struct {
		Login                string   `json:"login"`
		Name                 string   `json:"name"`
		VerifiedDomainEmails []string `json:"verified_emails"`
		LicenseType          string   `json:"license_type"`
		GHEC                 bool     `json:"ghec"`
		GHES                 bool     `json:"ghes"`
		VSS                  bool     `json:"vss"`
		Accounts             int      `json:"accounts"`
	}
)

// GetLicensing returns GitHub billing information
func GetLicensing(cmd *cobra.Command, args []string) (err error) {
	if enterprise == "" {
		return fmt.Errorf("--enterprise|-e is required")
	}

	if repo != "" {
		return fmt.Errorf("repository not supported for this report")
	}

	if owner != "" {
		return fmt.Errorf("owner not supported for this report")
	}

	sp.Start()

	sp.Suffix = fmt.Sprintf(
		" fetching %s license data",
		utils.Cyan(enterprise),
	)

	if err := restClient.Get(
		fmt.Sprintf(
			"enterprises/%s/consumed-licenses",
			enterprise,
		),
		&licenseData,
	); err != nil {
		sp.Stop()
		return err
	}

	sp.Stop()

	td := pterm.TableData{[]string{
		"purchased",
		"consumed",
		"free",
	}}
	header := []string{
		"login",
		"name",
		"verified_emails",
		"license_type",
		"ghec",
		"ghes",
		"vss",
		"accounts",
	}

	// start CSV file
	if csvPath != "" {
		licenseReport, err = utils.NewCSVReport(csvPath)

		if err != nil {
			return err
		}

		licenseReport.SetHeader(header)
	}

	td = append(td, []string{
		fmt.Sprintf("%d", licenseData.TotalSeatsPurchased),
		fmt.Sprintf("%d", licenseData.TotalSeatsConsumed),
		fmt.Sprintf("%d", licenseData.TotalSeatsPurchased-licenseData.TotalSeatsConsumed),
	})

	if !silent {
		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(td).WithRightAlignment().Render()
		fmt.Println("")
	}

	utd := pterm.TableData{header}

	var ru []LicenseUser
	for _, u := range licenseData.Users {
		ru = append(ru, LicenseUser{
			Login:                u.DotcomLogin,
			Name:                 u.DotcomName,
			VerifiedDomainEmails: u.DotcomVerifiedDomainEmails,
			LicenseType:          u.LicesneType,
			GHEC:                 u.DotcomUser,
			GHES:                 u.ServerUser,
			VSS:                  u.VSSUser,
			Accounts:             u.TotalUserAccounts,
		})

		data := []string{
			u.DotcomLogin,
			u.DotcomName,
			strings.Join(u.DotcomVerifiedDomainEmails, ", "),
			u.LicesneType,
			fmt.Sprintf("%t", u.DotcomUser),
			fmt.Sprintf("%t", u.ServerUser),
			fmt.Sprintf("%t", u.VSSUser),
			fmt.Sprintf("%d", u.TotalUserAccounts),
		}

		utd = append(utd, data)

		if csvPath != "" {
			licenseReport.AddData(data)
		}
	}

	res := LicenseReportJSON{
		Purchased: licenseData.TotalSeatsPurchased,
		Consumed:  licenseData.TotalSeatsConsumed,
		Free:      licenseData.TotalSeatsPurchased - licenseData.TotalSeatsConsumed,
		Users:     ru,
	}

	if !silent {
		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithData(utd).Render()
	}

	if csvPath != "" {
		licenseReport.Save()
	}

	if jsonPath != "" {
		err = utils.SaveJsonReport(jsonPath, res)
	}

	if mdPath != "" {
		err = utils.SaveMDReport(mdPath, mdLicenseReport, res)
	}

	return err
}
