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
	_ "embed"
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
	BillingCmd = &cobra.Command{
		Use:   "billing",
		Short: "Report on GitHub billing",
		Long: heredoc.Docf(
			`Report on GitHub billing for enterprises, organizations, and users.
			Requires %[1]sread:enterprise%[1]s, %[1]sread:org%[1]s, and/or %[1]sread:user%[1]s scope.

			Note: This command uses the new unified billing API endpoint (/settings/billing/usage)
			introduced with GitHub's metered billing platform. The Advanced Security billing data
			continues to use its dedicated endpoint.`,
			"`",
		),
		RunE: GetBilling,
	}

	all          bool
	actions      bool
	packages     bool
	security     bool
	storage      bool
	showCosts    bool
	billingMonth string
	billingYear  string

	billingReport utils.CSVReport

	//go:embed templates/billing.md.tmpl
	mdBillingTemplate string
)

type (
	BillingAccount struct {
		Login       string
		AccountType string // "enterprise", "organization", "user"
	}

	Billing struct {
		Organization string
		Actions      ActionsBilling
		Packages     PackagesBilling
		Security     SecurityBilling
		Storage      StorageBilling
	}

	// New unified billing usage response structure
	BillingUsageResponse struct {
		TimePeriod   TimePeriod  `json:"timePeriod"`
		Organization string      `json:"organization,omitempty"`
		Enterprise   string      `json:"enterprise,omitempty"`
		UsageItems   []UsageItem `json:"usageItems"`
	}

	TimePeriod struct {
		Year  int `json:"year"`
		Month int `json:"month"`
	}

	UsageItem struct {
		Date             string  `json:"date,omitempty"`
		Product          string  `json:"product"`
		SKU              string  `json:"sku"`
		GrossQuantity    float64 `json:"grossQuantity"`
		DiscountQuantity float64 `json:"discountQuantity"`
		NetQuantity      float64 `json:"netQuantity"`
		UnitType         string  `json:"unitType"`
		PricePerUnit     float64 `json:"pricePerUnit"`
		GrossAmount      float64 `json:"grossAmount"`
		DiscountAmount   float64 `json:"discountAmount"`
		NetAmount        float64 `json:"netAmount"`
		OrganizationName string  `json:"organizationName,omitempty"`
		RepositoryName   string  `json:"repositoryName,omitempty"`
	}

	ActionsBilling struct {
		// Quantity fields (in minutes)
		TotalMinutesUsed     float64 `json:"total_minutes_used"`
		TotalPaidMinutesUsed float64 `json:"total_paid_minutes_used"`
		IncludedMinutes      float64 `json:"included_minutes"`
		MinutesUsedBreakdown struct {
			MacOS   float64 `json:"MACOS"`
			Ubuntu  float64 `json:"UBUNTU"`
			Windows float64 `json:"WINDOWS"`
		} `json:"minutes_used_breakdown"`
		// Cost fields (in dollars)
		GrossAmount    float64 `json:"gross_amount"`
		DiscountAmount float64 `json:"discount_amount"`
		NetAmount      float64 `json:"net_amount"`
	}

	PackagesBilling struct {
		// Quantity fields (in gigabytes)
		TotalGigabytesBandwidthUsed     float64 `json:"total_gigabytes_bandwidth_used"`
		TotalPaidGigabytesBandwidthUsed float64 `json:"total_paid_gigabytes_bandwidth_used"`
		IncludedGigabytesBandwidth      float64 `json:"included_gigabytes_bandwidth"`
		// Cost fields (in dollars)
		GrossAmount    float64 `json:"gross_amount"`
		DiscountAmount float64 `json:"discount_amount"`
		NetAmount      float64 `json:"net_amount"`
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
		DaysLeftInBillingCycle int `json:"days_left_in_billing_cycle"`
		// Quantity fields (in gigabyte-hours)
		EstimatedStorageForMonth float64 `json:"estimated_storage_for_month"`
		ActionsStorageGB         float64 `json:"actions_storage_gb"`
		PackagesStorageGB        float64 `json:"packages_storage_gb"`
		// Cost fields (in dollars)
		GrossAmount                   float64 `json:"gross_amount"`
		DiscountAmount                float64 `json:"discount_amount"`
		NetAmount                     float64 `json:"net_amount"`
		ActionsStorageGrossAmount     float64 `json:"actions_storage_gross_amount"`
		ActionsStorageDiscountAmount  float64 `json:"actions_storage_discount_amount"`
		ActionsStorageNetAmount       float64 `json:"actions_storage_net_amount"`
		PackagesStorageGrossAmount    float64 `json:"packages_storage_gross_amount"`
		PackagesStorageDiscountAmount float64 `json:"packages_storage_discount_amount"`
		PackagesStorageNetAmount      float64 `json:"packages_storage_net_amount"`
	}

	BillingReportJSON struct {
		Account                    string  `json:"account"`
		ActionMinutesUsed          float64 `json:"action_minutes_used,omitempty"`
		ActionNetCost              float64 `json:"action_net_cost,omitempty"`
		ActionDiscountAmount       float64 `json:"action_discount_amount,omitempty"`
		GigabytesBandwidthUsed     float64 `json:"gigabytes_bandwidth_used,omitempty"`
		PackagesNetCost            float64 `json:"packages_net_cost,omitempty"`
		PackagesDiscountAmount     float64 `json:"packages_discount_amount,omitempty"`
		AdvancedSecurityCommitters int     `json:"advanced_security_committers,omitempty"`
		EstimatedStorageForMonth   float64 `json:"estimated_storage_for_month,omitempty"`
		ActionsStorageGB           float64 `json:"actions_storage_gb,omitempty"`
		PackagesStorageGB          float64 `json:"packages_storage_gb,omitempty"`
		StorageNetCost             float64 `json:"storage_net_cost,omitempty"`
		StorageDiscountAmount      float64 `json:"storage_discount_amount,omitempty"`
	}
)

// MarshalJSON implements custom JSON marshaling for BillingReportJSON
// to prevent scientific notation for small float values
func (b BillingReportJSON) MarshalJSON() ([]byte, error) {
	type Alias BillingReportJSON
	return []byte(fmt.Sprintf(`{"account":"%s","action_minutes_used":%.6f,"action_net_cost":%.6f,"action_discount_amount":%.6f,"gigabytes_bandwidth_used":%.6f,"packages_net_cost":%.6f,"packages_discount_amount":%.6f,"advanced_security_committers":%d,"estimated_storage_for_month":%.10f,"actions_storage_gb":%.10f,"packages_storage_gb":%.10f,"storage_net_cost":%.10f,"storage_discount_amount":%.10f}`,
		b.Account,
		b.ActionMinutesUsed,
		b.ActionNetCost,
		b.ActionDiscountAmount,
		b.GigabytesBandwidthUsed,
		b.PackagesNetCost,
		b.PackagesDiscountAmount,
		b.AdvancedSecurityCommitters,
		b.EstimatedStorageForMonth,
		b.ActionsStorageGB,
		b.PackagesStorageGB,
		b.StorageNetCost,
		b.StorageDiscountAmount,
	)), nil
}

func init() {
	BillingCmd.PersistentFlags().BoolVar(&all, "all", true, "Get all billing data")
	BillingCmd.PersistentFlags().BoolVar(&actions, "actions", false, "Get GitHub Actions billing")
	BillingCmd.PersistentFlags().BoolVar(&packages, "packages", false, "Get GitHub Packages billing")
	BillingCmd.PersistentFlags().BoolVar(&security, "security", false, "Get GitHub Advanced Security active committers")
	BillingCmd.PersistentFlags().BoolVar(&storage, "storage", false, "Get shared storage billing")
	BillingCmd.PersistentFlags().BoolVar(&showCosts, "show-costs", false, "Show cost information (net, gross, discount amounts)")
	BillingCmd.PersistentFlags().StringVar(&billingMonth, "month", "", "Billing month for storage data (MM, defaults to current month)")
	BillingCmd.PersistentFlags().StringVar(&billingYear, "year", "", "Billing year for storage data (YYYY, defaults to current year)")

	BillingCmd.MarkFlagsMutuallyExclusive("all", "actions")
	BillingCmd.MarkFlagsMutuallyExclusive("all", "packages")
	BillingCmd.MarkFlagsMutuallyExclusive("all", "security")
	BillingCmd.MarkFlagsMutuallyExclusive("all", "storage")

	RootCmd.AddCommand(BillingCmd)
}

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

// Helper functions to aggregate usage data from new billing API
func aggregateActionsUsage(usageItems []UsageItem) ActionsBilling {
	var result ActionsBilling
	for _, item := range usageItems {
		if item.Product == "Actions" && item.UnitType == "minutes" {
			// Aggregate quantities (minutes)
			result.TotalMinutesUsed += item.GrossQuantity
			result.TotalPaidMinutesUsed += item.NetQuantity
			result.IncludedMinutes += item.DiscountQuantity

			// Aggregate costs (dollars)
			result.GrossAmount += item.GrossAmount
			result.DiscountAmount += item.DiscountAmount
			result.NetAmount += item.NetAmount

			// Map SKU to breakdown (by gross quantity)
			// Note: SKU string handling should be case-insensitive
			sku := strings.ToUpper(item.SKU)
			if strings.Contains(sku, "MACOS") || strings.Contains(sku, "MAC") {
				result.MinutesUsedBreakdown.MacOS += item.GrossQuantity
			} else if strings.Contains(sku, "WINDOWS") {
				result.MinutesUsedBreakdown.Windows += item.GrossQuantity
			} else if strings.Contains(sku, "UBUNTU") || strings.Contains(sku, "LINUX") {
				result.MinutesUsedBreakdown.Ubuntu += item.GrossQuantity
			}
		}
	}
	return result
}

func aggregatePackagesUsage(usageItems []UsageItem) PackagesBilling {
	var result PackagesBilling
	for _, item := range usageItems {
		// Note: Packages bandwidth is reported in the API, but storage uses different unitType
		if item.Product == "Packages" {
			if item.UnitType == "gigabytes" {
				// Aggregate bandwidth quantities (gigabytes)
				result.TotalGigabytesBandwidthUsed += item.GrossQuantity
				result.TotalPaidGigabytesBandwidthUsed += item.NetQuantity
				result.IncludedGigabytesBandwidth += item.DiscountQuantity
			}
			// Aggregate all Packages costs regardless of unitType
			result.GrossAmount += item.GrossAmount
			result.DiscountAmount += item.DiscountAmount
			result.NetAmount += item.NetAmount
		}
	}
	return result
}

func aggregateStorageUsage(usageItems []UsageItem) StorageBilling {
	var result StorageBilling

	// Separate Actions and Packages storage
	for _, item := range usageItems {
		if item.UnitType == "gigabyte-hours" {
			switch item.Product {
			case "Actions":
				result.ActionsStorageGB += item.GrossQuantity
				result.ActionsStorageGrossAmount += item.GrossAmount
				result.ActionsStorageDiscountAmount += item.DiscountAmount
				result.ActionsStorageNetAmount += item.NetAmount
			case "Packages":
				result.PackagesStorageGB += item.GrossQuantity
				result.PackagesStorageGrossAmount += item.GrossAmount
				result.PackagesStorageDiscountAmount += item.DiscountAmount
				result.PackagesStorageNetAmount += item.NetAmount
			case "Git LFS":
				// Git LFS storage is tracked separately but included in packages totals
				result.PackagesStorageGB += item.GrossQuantity
				result.PackagesStorageGrossAmount += item.GrossAmount
				result.PackagesStorageDiscountAmount += item.DiscountAmount
				result.PackagesStorageNetAmount += item.NetAmount
			case "Shared Storage":
				if strings.Contains(strings.ToLower(item.SKU), "actions") {
					result.ActionsStorageGB += item.GrossQuantity
					result.ActionsStorageGrossAmount += item.GrossAmount
					result.ActionsStorageDiscountAmount += item.DiscountAmount
					result.ActionsStorageNetAmount += item.NetAmount
				} else {
					// Default to packages storage if not explicitly actions
					result.PackagesStorageGB += item.GrossQuantity
					result.PackagesStorageGrossAmount += item.GrossAmount
					result.PackagesStorageDiscountAmount += item.DiscountAmount
					result.PackagesStorageNetAmount += item.NetAmount
				}
			}
		}
	}

	// Total storage quantities (gigabyte-hours)
	result.EstimatedStorageForMonth = result.ActionsStorageGB + result.PackagesStorageGB

	// Total storage costs (dollars)
	result.GrossAmount = result.ActionsStorageGrossAmount + result.PackagesStorageGrossAmount
	result.DiscountAmount = result.ActionsStorageDiscountAmount + result.PackagesStorageDiscountAmount
	result.NetAmount = result.ActionsStorageNetAmount + result.PackagesStorageNetAmount

	return result
}

// buildBillingEndpoint constructs the appropriate billing endpoint based on account type
// Account types are determined by the GitHub API response in cmd.go:
// - "user": for User accounts (uses /users/{login}/settings/billing/{path})
// - "enterprise": for Enterprise accounts (uses /enterprises/{login}/settings/billing/{path})
// - "organization": for Organization accounts (uses /orgs/{login}/settings/billing/{path})
func buildBillingEndpoint(accountType, login, path string) string {
	switch accountType {
	case "user":
		return fmt.Sprintf("users/%s/settings/billing/%s", login, path)
	case "enterprise":
		return fmt.Sprintf("enterprises/%s/settings/billing/%s", login, path)
	default: // "organization" or fallback
		return fmt.Sprintf("orgs/%s/settings/billing/%s", login, path)
	}
}

// buildBillingQueryParams constructs query parameters for billing API calls.
// According to GitHub's documentation, month and year parameters are applicable for
// organization and enterprise accounts. User-level accounts do not support these parameters.
// If not provided, defaults to current month and year.
// Month should be in MM format (e.g., "01", "12"). If month is provided without year,
// year defaults to current year.
func buildBillingQueryParams(accountType string) string {
	// User accounts don't support month/year filtering on usage endpoints
	if accountType == "user" {
		return ""
	}

	now := time.Now()

	// Get year first (needed to build month parameter)
	year := billingYear
	if year == "" {
		year = now.Format("2006")
	}

	// Get month and ensure it's zero-padded to MM format
	month := billingMonth
	if month == "" {
		month = now.Format("01") // MM format
		// If year is not provided, use current year
		if year == "" {
			year = now.Format("2006")
		}
	} else if len(month) == 1 {
		month = fmt.Sprintf("0%s", month) // Zero-pad single digit months
	}

	// If year is still empty (e.g. month was provided but year wasn't), default to current year
	if year == "" {
		year = now.Format("2006")
	}

	return fmt.Sprintf("?month=%s&year=%s", month, year)
}

// GetBilling returns GitHub billing information
// Note: The global 'user' variable is populated by cmd.go's run() function
// and contains the owner's Login and Type ("User" or "Organization")
func GetBilling(cmd *cobra.Command, args []string) (err error) {
	if repo != "" {
		return fmt.Errorf("repository not supported for this report")
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

	var accounts []BillingAccount

	// Add organizations from enterprise
	for _, org := range organizations {
		accounts = append(accounts, BillingAccount{
			Login:       org.Login,
			AccountType: "organization",
		})
	}

	// Add owner (could be org or user)
	if owner != "" {
		// Use the Type from the API response ("User" or "Organization")
		// The 'user' global variable is populated by cmd.go's run() function
		accountType := "organization"
		if user.Type == "User" {
			accountType = "user"
		}
		accounts = append(accounts, BillingAccount{
			Login:       owner,
			AccountType: accountType,
		})
	}

	var billing []Billing
	securitySkipped := false

	for _, account := range accounts {
		var actionsBillingData ActionsBilling
		var packagesBillingData PackagesBilling
		var securityBillingData SecurityBilling
		var storageBillingData StorageBilling

		// Fetch unified billing usage data if actions, packages, or storage is requested
		if actions || packages || storage {
			sp.Suffix = fmt.Sprintf(
				" fetching %s billing report %s",
				utils.Cyan(account.Login),
				utils.HiBlack("(usage data)"),
			)

			var usageResponse BillingUsageResponse
			// Use the summary endpoint: /settings/billing/usage/summary
			// Query parameters are only supported for org/enterprise accounts
			endpoint := buildBillingEndpoint(account.AccountType, account.Login, "usage/summary") + buildBillingQueryParams(account.AccountType)
			if err := restClient.Get(
				endpoint,
				&usageResponse,
			); err != nil {
				if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "404") {
					sp.Suffix = fmt.Sprintf(
						" fetching %s billing report %s",
						utils.Cyan(account.Login),
						utils.Orange("(usage data not accessible, skipping)"),
					)
					continue
				}
				if strings.Contains(err.Error(), "500") || strings.Contains(err.Error(), "502") || strings.Contains(err.Error(), "503") {
					sp.Suffix = fmt.Sprintf(
						" fetching %s billing report %s",
						utils.Cyan(account.Login),
						utils.Orange("(server error, skipping)"),
					)
					continue
				}
				return err
			}

			// Aggregate the usage data
			if actions {
				actionsBillingData = aggregateActionsUsage(usageResponse.UsageItems)
			}
			if packages {
				packagesBillingData = aggregatePackagesUsage(usageResponse.UsageItems)
			}
			if storage {
				storageBillingData = aggregateStorageUsage(usageResponse.UsageItems)
			}

			// sleep for 1 second to avoid rate limiting
			time.Sleep(1 * time.Second)
		}

		if security {
			sp.Suffix = fmt.Sprintf(
				" fetching %s billing report %s",
				utils.Cyan(account.Login),
				utils.HiBlack("(security data)"),
			)

			// Advanced Security endpoint - no product parameter required
			securityEndpoint := buildBillingEndpoint(account.AccountType, account.Login, "advanced-security")
			if err := restClient.Get(
				securityEndpoint,
				&securityBillingData,
			); err != nil {
				// silently ignore 403 and 422 errors (not enabled or not accessible)
				// Don't skip the entire account - just mark security as unavailable for this account
				if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "422") {
					sp.Suffix = fmt.Sprintf(
						" fetching %s billing report %s",
						utils.Cyan(account.Login),
						utils.Orange("(security not enabled, skipping)"),
					)

					securitySkipped = true
				} else if strings.Contains(err.Error(), "500") || strings.Contains(err.Error(), "502") || strings.Contains(err.Error(), "503") {
					sp.Suffix = fmt.Sprintf(
						" fetching %s billing report %s",
						utils.Cyan(account.Login),
						utils.Orange("(server error, skipping)"),
					)
					securitySkipped = true
				} else {
					return err
				}
			} else {
				// Advanced Security billing still uses separate endpoint
				// sleep for 1 second to avoid rate limiting
				time.Sleep(1 * time.Second)
			}
		}

		billing = append(billing, Billing{
			Organization: account.Login,
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
		"account",
	}

	if actions {
		header = append(header, "action_minutes_used")
		if showCosts {
			header = append(header, "action_net_cost", "action_discount_amount")
		}
	}
	if packages {
		header = append(header, "gigabytes_bandwidth_used")
		if showCosts {
			header = append(header, "packages_net_cost", "packages_discount_amount")
		}
	}
	if security && !securitySkipped {
		header = append(header, "advanced_security_committers")
	}
	if storage {
		header = append(header, "estimated_storage_for_month")
		if showCosts {
			header = append(header, "actions_storage_gb", "packages_storage_gb", "storage_net_cost")
		}
	}

	var td = pterm.TableData{header}
	var res []BillingReportJSON

	var actionsSum float64
	var actionsNetCostSum float64
	var actionsDiscountSum float64
	var packagesSum float64
	var packagesNetCostSum float64
	var packagesDiscountSum float64
	var securitySum int
	var storageSum float64
	var actionsStorageSum float64
	var packagesStorageSum float64
	var storageNetCostSum float64

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
			if showCosts {
				data = append(data, fmt.Sprintf("%.2f", b.Actions.NetAmount))
				data = append(data, fmt.Sprintf("%.2f", b.Actions.DiscountAmount))
			}
		}
		if packages {
			data = append(data, fmt.Sprintf("%.2f", b.Packages.TotalGigabytesBandwidthUsed))
			if showCosts {
				data = append(data, fmt.Sprintf("%.2f", b.Packages.NetAmount))
				data = append(data, fmt.Sprintf("%.2f", b.Packages.DiscountAmount))
			}
		}
		if security && !securitySkipped {
			data = append(data, fmt.Sprintf("%d", b.Security.TotalAdvancedSecurityCommitters))
		}
		if storage {
			data = append(data, fmt.Sprintf("%.2f", b.Storage.EstimatedStorageForMonth))
			if showCosts {
				data = append(data, fmt.Sprintf("%.2f", b.Storage.ActionsStorageGB))
				data = append(data, fmt.Sprintf("%.2f", b.Storage.PackagesStorageGB))
				data = append(data, fmt.Sprintf("%.2f", b.Storage.NetAmount))
			}
		}

		actionsSum += b.Actions.TotalMinutesUsed
		actionsNetCostSum += b.Actions.NetAmount
		actionsDiscountSum += b.Actions.DiscountAmount
		packagesSum += b.Packages.TotalGigabytesBandwidthUsed
		packagesNetCostSum += b.Packages.NetAmount
		packagesDiscountSum += b.Packages.DiscountAmount
		if security && !securitySkipped {
			securitySum += b.Security.TotalAdvancedSecurityCommitters
		}
		storageSum += b.Storage.EstimatedStorageForMonth
		actionsStorageSum += b.Storage.ActionsStorageGB
		packagesStorageSum += b.Storage.PackagesStorageGB
		storageNetCostSum += b.Storage.NetAmount

		td = append(td, data)

		if csvPath != "" {
			billingReport.AddData(data)
		}

		res = append(res, BillingReportJSON{
			Account:                    b.Organization,
			ActionMinutesUsed:          b.Actions.TotalMinutesUsed,
			ActionNetCost:              b.Actions.NetAmount,
			ActionDiscountAmount:       b.Actions.DiscountAmount,
			GigabytesBandwidthUsed:     b.Packages.TotalGigabytesBandwidthUsed,
			PackagesNetCost:            b.Packages.NetAmount,
			PackagesDiscountAmount:     b.Packages.DiscountAmount,
			AdvancedSecurityCommitters: b.Security.TotalAdvancedSecurityCommitters,
			EstimatedStorageForMonth:   b.Storage.EstimatedStorageForMonth,
			ActionsStorageGB:           b.Storage.ActionsStorageGB,
			PackagesStorageGB:          b.Storage.PackagesStorageGB,
			StorageNetCost:             b.Storage.NetAmount,
			StorageDiscountAmount:      b.Storage.DiscountAmount,
		})
	}

	div := []string{""}
	sum := []string{""}

	if actions {
		div = append(div, "")
		sum = append(sum, fmt.Sprintf("%.2f", actionsSum))
		if showCosts {
			div = append(div, "", "")
			sum = append(sum, fmt.Sprintf("%.2f", actionsNetCostSum))
			sum = append(sum, fmt.Sprintf("%.2f", actionsDiscountSum))
		}
	}
	if packages {
		div = append(div, "")
		sum = append(sum, fmt.Sprintf("%.2f", packagesSum))
		if showCosts {
			div = append(div, "", "")
			sum = append(sum, fmt.Sprintf("%.2f", packagesNetCostSum))
			sum = append(sum, fmt.Sprintf("%.2f", packagesDiscountSum))
		}
	}
	if security && !securitySkipped {
		div = append(div, "")
		sum = append(sum, fmt.Sprintf("%d", securitySum))
	}
	if storage {
		div = append(div, "")
		sum = append(sum, fmt.Sprintf("%.2f", storageSum))
		if showCosts {
			div = append(div, "", "", "")
			sum = append(sum, fmt.Sprintf("%.2f", actionsStorageSum))
			sum = append(sum, fmt.Sprintf("%.2f", packagesStorageSum))
			sum = append(sum, fmt.Sprintf("%.2f", storageNetCostSum))
		}
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
		err = utils.SaveJsonReport(jsonPath, res)
	}

	if mdPath != "" {
		err = utils.SaveMDReport(mdPath, mdBillingTemplate, struct {
			Data                  []BillingReportJSON
			IsActions             bool
			IsPackages            bool
			IsSecurity            bool
			IsStorage             bool
			ShowCosts             bool
			TotalActions          float64
			TotalActionsNetCost   float64
			TotalActionsDiscount  float64
			TotalPackages         float64
			TotalPackagesNetCost  float64
			TotalPackagesDiscount float64
			TotalSecurity         int
			TotalStorage          float64
			TotalActionsStorage   float64
			TotalPackagesStorage  float64
			TotalStorageNetCost   float64
		}{
			Data:                  res,
			IsActions:             actions,
			IsPackages:            packages,
			IsSecurity:            security,
			IsStorage:             storage,
			ShowCosts:             showCosts,
			TotalActions:          actionsSum,
			TotalActionsNetCost:   actionsNetCostSum,
			TotalActionsDiscount:  actionsDiscountSum,
			TotalPackages:         packagesSum,
			TotalPackagesNetCost:  packagesNetCostSum,
			TotalPackagesDiscount: packagesDiscountSum,
			TotalSecurity:         securitySum,
			TotalStorage:          storageSum,
			TotalActionsStorage:   actionsStorageSum,
			TotalPackagesStorage:  packagesStorageSum,
			TotalStorageNetCost:   storageNetCostSum,
		})
	}

	return err
}
