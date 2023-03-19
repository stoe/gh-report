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

	"github.com/pterm/pterm"
	"github.com/shurcooL/graphql"
	"github.com/spf13/cobra"
	"github.com/stoe/gh-report/utils"
)

var (
	BillingCmd = &cobra.Command{
		Use:   "billing",
		Short: "Report on GitHub billing",
		Long:  "Report on GitHub billing",
		RunE:  GetBilling,
	}

	all      bool
	actions  bool
	packages bool
	security bool
	storage  bool

	billingReport utils.CSVReport
)

func init() {
	BillingCmd.PersistentFlags().BoolVar(&all, "all", true, "Get all billing data")
	BillingCmd.PersistentFlags().BoolVar(&actions, "actions", false, "Get GitHub Actions billing")
	BillingCmd.PersistentFlags().BoolVar(&packages, "packages", false, "Get GitHub Packages billing")
	BillingCmd.PersistentFlags().BoolVar(&security, "security", false, "Get GitHub Advanced Security active committers")
	BillingCmd.PersistentFlags().BoolVar(&storage, "storage", false, "Get shared storage billing")

	BillingCmd.MarkFlagsMutuallyExclusive("all", "actions")
	BillingCmd.MarkFlagsMutuallyExclusive("all", "packages")
	BillingCmd.MarkFlagsMutuallyExclusive("all", "security")
	BillingCmd.MarkFlagsMutuallyExclusive("all", "storage")

	RootCmd.AddCommand(BillingCmd)
}

type (
	Billing struct {
		Organization string
		Actions      ActionsBilling
		Packages     PackagesBilling
		Security     SecurityBilling
		Storage      StorageBilling
	}

	ActionsBilling struct {
		TotalMinutesUsed     float64 `json:"total_minutes_used"`
		TotalPaidMinutesUsed float64 `json:"total_paid_minutes_used"`
		IncludedMinutes      float64 `json:"included_minutes"`
		MinutesUsedBreakdown struct {
			MacOS   float64 `json:"MACOS"`
			Ubuntu  float64 `json:"UBUNTU"`
			Windows float64 `json:"WINDOWS"`
		} `json:"minutes_used_breakdown"`
	}

	PackagesBilling struct {
		TotalGigabytesBandwidthUsed     float64 `json:"total_gigabytes_bandwidth_used"`
		TotalPaidGigabytesBandwidthUsed float64 `json:"total_paid_gigabytes_bandwidth_used"`
		IncludedGigabytesBandwidth      float64 `json:"included_gigabytes_bandwidth"`
	}

	SecurityBilling struct {
		TotalAdvancedSecurityCommitters int                         `json:"total_advanced_security_committers"`
		TotalCount                      int                         `json:"total_count"`
		Repositories                    []SecurityBillingRepository `json:"repositories"`
	}

	SecurityBillingRepository struct {
		Name                                string                                `json:"name"`
		AdvancedSecurityCommitters          int                                   `json:"advanced_security_committers"`
		AdvancedSecurityCommittersBreakdown []AdvancedSecurityCommittersBreakdown `json:"advanced_security_committers_breakdown"`
	}

	AdvancedSecurityCommittersBreakdown struct {
		UserLogin      string   `json:"user_login"`
		LastPushedDate PushDate `json:"last_pushed_date"`
	}

	PushDate struct {
		time.Time
	}

	StorageBilling struct {
		DaysLeftInBillingCycle       int `json:"days_left_in_billing_cycle"`
		EstimatedPaidStorageForMonth int `json:"estimated_paid_storage_for_month"`
		EstimatedStorageForMonth     int `json:"estimated_storage_for_month"`
	}

	BillingReportJSON struct {
		Organization               string  `json:"organization"`
		ActionMinutesUsed          float64 `json:"action_minutes_used"`
		GigabytesBandwidthUsed     float64 `json:"gigabytes_bandwidth_used"`
		AdvancedSecurityCommitters int     `json:"advanced_security_committers"`
		EstimatedStorageForMonth   int     `json:"estimated_storage_for_month"`
	}
)

func (c *PushDate) UnmarshalJSON(b []byte) error {
	value := strings.Trim(string(b), `"`) //get rid of "

	if value == "" || value == "null" {
		return nil
	}

	t, err := time.Parse("2006-01-02", value) //parse time

	if err != nil {
		return err
	}
	c.Time = t

	return nil
}

// GetBilling returns GitHub billing information
func GetBilling(cmd *cobra.Command, args []string) (err error) {
	if repo != "" {
		return fmt.Errorf("repository not supported for this report")
	}

	if user.Type == "User" {
		return fmt.Errorf("%s type not supported for this report", user.Type)
	}

	if actions || packages || security || storage {
		all = false
	}

	if all {
		actions = true
		packages = true
		security = true
		storage = true
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

	var billing []Billing

	for _, org := range organizations {
		var actionsBillingData ActionsBilling
		var packagesBillingData PackagesBilling
		var securityBillingData SecurityBilling
		var storageBillingData StorageBilling

		if actions {
			sp.Suffix = fmt.Sprintf(
				" fetching %s billing report %s",
				utils.Cyan(org.Login),
				utils.HiBlack("(actions data)"),
			)

			if err := restClient.Get(
				fmt.Sprintf(
					"orgs/%s/settings/billing/actions",
					org.Login,
				),
				&actionsBillingData,
			); err != nil {
				return err
			}

			// sleep for 1 second to avoid rate limiting
			time.Sleep(1 * time.Second)
		}

		if packages {
			sp.Suffix = fmt.Sprintf(
				" fetching %s billing report %s",
				utils.Cyan(org.Login),
				utils.HiBlack("(packages data)"),
			)

			if err := restClient.Get(
				fmt.Sprintf(
					"orgs/%s/settings/billing/packages",
					org.Login,
				),
				&packagesBillingData,
			); err != nil {
				return err
			}

			// sleep for 1 second to avoid rate limiting
			time.Sleep(1 * time.Second)
		}

		if security {
			sp.Suffix = fmt.Sprintf(
				" fetching %s billing report %s",
				utils.Cyan(org.Login),
				utils.HiBlack("(security data)"),
			)

			if err := restClient.Get(
				fmt.Sprintf(
					"orgs/%s/settings/billing/advanced-security",
					org.Login,
				),
				&securityBillingData,
			); err != nil {
				return err
			}

			// sleep for 1 second to avoid rate limiting
			time.Sleep(1 * time.Second)
		}

		if storage {
			sp.Suffix = fmt.Sprintf(
				" fetching %s billing report %s",
				utils.Cyan(org.Login),
				utils.HiBlack("(storage data)"),
			)

			if err := restClient.Get(
				fmt.Sprintf(
					"orgs/%s/settings/billing/shared-storage",
					org.Login,
				),
				&storageBillingData,
			); err != nil {
				return err
			}

			// sleep for 1 second to avoid rate limiting
			time.Sleep(1 * time.Second)
		}

		billing = append(billing, Billing{
			Organization: org.Login,
			Actions:      actionsBillingData,
			Packages:     packagesBillingData,
			Security:     securityBillingData,
			Storage:      storageBillingData,
		})
	}

	// sleep for 1 second to avoid rate limiting
	time.Sleep(1 * time.Second)

	sp.Stop()

	header := []string{
		"organization",
	}

	if actions {
		header = append(header, "action_minutes_used")
	}
	if packages {
		header = append(header, "gigabytes_bandwidth_used")
	}
	if security {
		header = append(header, "advanced_security_committers")
	}
	if storage {
		header = append(header, "estimated_storage_for_month")
	}

	var td = pterm.TableData{header}
	var res []BillingReportJSON

	var actionsSum float64
	var packagesSum float64
	var securitySum int
	var storageSum int

	// start CSV file
	if csvPath != "" {
		billingReport, err = utils.NewCSVReport(csvPath)

		if err != nil {
			return err
		}

		billingReport.SetHeader(header)
	}

	for _, b := range billing {
		var data = []string{
			b.Organization,
		}

		if actions {
			data = append(data, fmt.Sprintf("%.2f", b.Actions.TotalMinutesUsed))
		}
		if packages {
			data = append(data, fmt.Sprintf("%.2f", b.Packages.TotalGigabytesBandwidthUsed))
		}
		if security {
			data = append(data, fmt.Sprintf("%d", b.Security.TotalAdvancedSecurityCommitters))
		}
		if storage {
			data = append(data, fmt.Sprintf("%d", b.Storage.EstimatedStorageForMonth))
		}

		actionsSum += b.Actions.TotalMinutesUsed
		packagesSum += b.Packages.TotalGigabytesBandwidthUsed
		securitySum += b.Security.TotalAdvancedSecurityCommitters
		storageSum += b.Storage.EstimatedStorageForMonth

		td = append(td, data)

		if csvPath != "" {
			billingReport.AddData(data)
		}

		res = append(res, BillingReportJSON{
			Organization:               b.Organization,
			ActionMinutesUsed:          b.Actions.TotalMinutesUsed,
			GigabytesBandwidthUsed:     b.Packages.TotalGigabytesBandwidthUsed,
			AdvancedSecurityCommitters: b.Security.TotalAdvancedSecurityCommitters,
			EstimatedStorageForMonth:   b.Storage.EstimatedStorageForMonth,
		})
	}

	div := []string{""}
	sum := []string{""}

	if actions {
		div = append(div, "")
		sum = append(sum, utils.Bold(fmt.Sprintf("%.2f", actionsSum)))
	}
	if packages {
		div = append(div, "")
		sum = append(sum, utils.Bold(fmt.Sprintf("%.2f", packagesSum)))
	}
	if security {
		div = append(div, "")
		sum = append(sum, utils.Bold(securitySum))
	}
	if storage {
		div = append(div, "")
		sum = append(sum, utils.Bold(storageSum))
	}

	td = append(td, div)
	td = append(td, sum)

	if !silent {
		pterm.DefaultTable.WithHasHeader().WithHeaderRowSeparator("-").WithRightAlignment(true).WithData(td).Render()
	}

	if csvPath != "" {
		billingReport.Save()
	}

	if jsonPath != "" {
		utils.SaveJsonReport(jsonPath, res)
	}

	return nil
}
