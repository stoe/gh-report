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

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	licenseCmd = &cobra.Command{
		Use:   "license",
		Short: "Report on GitHub Enterprise licensing",
		Long:  "Report on GitHub Enterprise licensing",
		RunE:  GetLicensing,
	}

	licenseData LicenseData
	// licenseReport utils.CSVReport
)

func init() {
	rootCmd.AddCommand(licenseCmd)
}

type LicenseData struct {
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

// GetLicensing returns GitHub billing information
func GetLicensing(cmd *cobra.Command, args []string) (err error) {
	if enterprise == "" {
		return fmt.Errorf("--enterprise is required")
	}

	if repo != "" {
		return fmt.Errorf("--repo not allowed")
	}

	if owner != "" {
		return fmt.Errorf("--owner not allowed")
	}

	if csvPath != "" {
		return fmt.Errorf("--csv not supported")
	}

	sp.Start()

	sp.Suffix = fmt.Sprintf(
		" fetching %s license data",
		blue(enterprise),
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

	fmt.Printf(
		"license data for %s\n\n",
		blue(enterprise),
	)

	td := pterm.TableData{[]string{
		"purchased",
		"consumed",
	}}

	td = append(td, []string{
		fmt.Sprintf("%d", licenseData.TotalSeatsPurchased),
		fmt.Sprintf("%d", licenseData.TotalSeatsConsumed),
	})

	pterm.DefaultTable.WithHasHeader().WithData(td).WithRightAlignment().Render()
	fmt.Println("")

	utd := pterm.TableData{[]string{
		"login",
		"name",
		"verified_emails",
		"license_type",
		"ghec",
		"ghes",
		"vss",
		"accounts",
	}}

	// TODO
	for _, u := range licenseData.Users {
		ud, us, uv := "❌", "❌", "❌"

		if u.DotcomUser {
			ud = "✅"
		}
		if u.ServerUser {
			us = "✅"
		}
		if u.VSSUser {
			uv = "✅"
		}

		utd = append(utd, []string{
			u.DotcomLogin,
			u.DotcomName,
			strings.Join(u.DotcomVerifiedDomainEmails, ", "),
			u.LicesneType,
			ud,
			us,
			uv,
			fmt.Sprintf("%d", u.TotalUserAccounts),
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(utd).Render()

	return nil
}
