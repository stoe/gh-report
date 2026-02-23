package cmd

import (
	"strings"
	"testing"
	"time"
)

func Test_Billing(t *testing.T) {
	t.Skip()
}

func Test_buildBillingEndpoint(t *testing.T) {
	tests := []struct {
		name        string
		accountType string
		login       string
		path        string
		want        string
	}{
		{
			name:        "organization usage endpoint",
			accountType: "organization",
			login:       "myorg",
			path:        "usage",
			want:        "orgs/myorg/settings/billing/usage",
		},
		{
			name:        "user usage endpoint",
			accountType: "user",
			login:       "myuser",
			path:        "usage",
			want:        "users/myuser/settings/billing/usage",
		},
		{
			name:        "enterprise usage endpoint",
			accountType: "enterprise",
			login:       "myenterprise",
			path:        "usage",
			want:        "enterprises/myenterprise/settings/billing/usage",
		},
		{
			name:        "organization advanced-security endpoint",
			accountType: "organization",
			login:       "myorg",
			path:        "advanced-security",
			want:        "orgs/myorg/settings/billing/advanced-security",
		},
		{
			name:        "user advanced-security endpoint",
			accountType: "user",
			login:       "myuser",
			path:        "advanced-security",
			want:        "users/myuser/settings/billing/advanced-security",
		},
		{
			name:        "fallback to organization for unknown type",
			accountType: "unknown",
			login:       "test",
			path:        "usage",
			want:        "orgs/test/settings/billing/usage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildBillingEndpoint(tt.accountType, tt.login, tt.path)
			if got != tt.want {
				t.Errorf("buildBillingEndpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildBillingQueryParams(t *testing.T) {
	// Save original values
	originalMonth := billingMonth
	originalYear := billingYear
	defer func() {
		billingMonth = originalMonth
		billingYear = originalYear
	}()

	t.Run("user account returns empty string", func(t *testing.T) {
		billingMonth = ""
		billingYear = ""
		result := buildBillingQueryParams("user")
		if result != "" {
			t.Errorf("Expected empty string for user account, got %s", result)
		}
	})

	t.Run("organization account with no params defaults to current month and year", func(t *testing.T) {
		billingMonth = ""
		billingYear = ""
		result := buildBillingQueryParams("organization")

		now := time.Now()
		expectedMonth := now.Format("01") // MM format
		expectedYear := now.Format("2006")

		if !strings.Contains(result, "month="+expectedMonth) {
			t.Errorf("Expected result to contain month=%s, got %s", expectedMonth, result)
		}
		if !strings.Contains(result, "year="+expectedYear) {
			t.Errorf("Expected result to contain year=%s, got %s", expectedYear, result)
		}
	})

	t.Run("enterprise account with no params defaults to current month and year", func(t *testing.T) {
		billingMonth = ""
		billingYear = ""
		result := buildBillingQueryParams("enterprise")

		now := time.Now()
		expectedMonth := now.Format("01") // MM format
		expectedYear := now.Format("2006")

		if !strings.Contains(result, "month="+expectedMonth) {
			t.Errorf("Expected result to contain month=%s, got %s", expectedMonth, result)
		}
		if !strings.Contains(result, "year="+expectedYear) {
			t.Errorf("Expected result to contain year=%s, got %s", expectedYear, result)
		}
	})

	t.Run("organization account with custom month and year", func(t *testing.T) {
		billingMonth = "12"
		billingYear = "2025"
		result := buildBillingQueryParams("organization")

		if !strings.Contains(result, "month=12") {
			t.Errorf("Expected result to contain month=12, got %s", result)
		}
		if !strings.Contains(result, "year=2025") {
			t.Errorf("Expected result to contain year=2025, got %s", result)
		}
	})

	t.Run("organization account with single-digit month gets zero-padded", func(t *testing.T) {
		billingMonth = "6"
		billingYear = "2025"
		result := buildBillingQueryParams("organization")

		if !strings.Contains(result, "month=06") {
			t.Errorf("Expected result to contain month=06, got %s", result)
		}
		if !strings.Contains(result, "year=2025") {
			t.Errorf("Expected result to contain year=2025, got %s", result)
		}
	})

	t.Run("organization account with only custom month defaults to current year", func(t *testing.T) {
		billingMonth = "06"
		billingYear = ""
		result := buildBillingQueryParams("organization")

		now := time.Now()
		expectedYear := now.Format("2006")

		if !strings.Contains(result, "month=06") {
			t.Errorf("Expected result to contain month=06, got %s", result)
		}
		if !strings.Contains(result, "year="+expectedYear) {
			t.Errorf("Expected result to contain year=%s, got %s", expectedYear, result)
		}
	})

	t.Run("organization account with only custom year defaults to current month", func(t *testing.T) {
		billingMonth = ""
		billingYear = "2024"
		result := buildBillingQueryParams("organization")

		now := time.Now()
		expectedMonth := now.Format("01") // Current month in MM format

		if !strings.Contains(result, "month="+expectedMonth) {
			t.Errorf("Expected result to contain month=%s, got %s", expectedMonth, result)
		}
		if !strings.Contains(result, "year=2024") {
			t.Errorf("Expected result to contain year=2024, got %s", result)
		}
	})
}

func Test_aggregateActionsUsage(t *testing.T) {
	usageItems := []UsageItem{
		{Product: "Actions", UnitType: "minutes", SKU: "Actions Linux", GrossQuantity: 100, NetQuantity: 95, DiscountQuantity: 5, GrossAmount: 0.60, DiscountAmount: 0.03, NetAmount: 0.57},
		{Product: "Actions", UnitType: "minutes", SKU: "Actions macOS", GrossQuantity: 50, NetQuantity: 48, DiscountQuantity: 2, GrossAmount: 3.10, DiscountAmount: 0.12, NetAmount: 2.98},
		{Product: "Actions", UnitType: "minutes", SKU: "Actions Windows", GrossQuantity: 75, NetQuantity: 70, DiscountQuantity: 5, GrossAmount: 0.75, DiscountAmount: 0.05, NetAmount: 0.70},
		{Product: "Packages", UnitType: "gigabytes", SKU: "Packages data transfer", GrossQuantity: 10, NetQuantity: 10, DiscountQuantity: 0, GrossAmount: 0.875, DiscountAmount: 0, NetAmount: 0.875},
	}

	result := aggregateActionsUsage(usageItems)

	if result.TotalMinutesUsed != 225 {
		t.Errorf("Expected TotalMinutesUsed to be 225, got %.2f", result.TotalMinutesUsed)
	}
	if result.TotalPaidMinutesUsed != 213 {
		t.Errorf("Expected TotalPaidMinutesUsed to be 213, got %.2f", result.TotalPaidMinutesUsed)
	}
	if result.IncludedMinutes != 12 {
		t.Errorf("Expected IncludedMinutes to be 12, got %.2f", result.IncludedMinutes)
	}
	if result.NetAmount != 4.25 {
		t.Errorf("Expected NetAmount to be 4.25, got %.2f", result.NetAmount)
	}
	if result.MinutesUsedBreakdown.Ubuntu != 100 {
		t.Errorf("Expected Ubuntu minutes to be 100, got %.2f", result.MinutesUsedBreakdown.Ubuntu)
	}
	if result.MinutesUsedBreakdown.MacOS != 50 {
		t.Errorf("Expected MacOS minutes to be 50, got %.2f", result.MinutesUsedBreakdown.MacOS)
	}
	if result.MinutesUsedBreakdown.Windows != 75 {
		t.Errorf("Expected Windows minutes to be 75, got %.2f", result.MinutesUsedBreakdown.Windows)
	}
}

func Test_aggregatePackagesUsage(t *testing.T) {
	usageItems := []UsageItem{
		{Product: "Packages", UnitType: "gigabytes", SKU: "Packages data transfer", GrossQuantity: 100, NetQuantity: 95, DiscountQuantity: 5, GrossAmount: 8.75, DiscountAmount: 0.44, NetAmount: 8.31},
		{Product: "Packages", UnitType: "gigabytes", SKU: "Packages data transfer", GrossQuantity: 50, NetQuantity: 50, DiscountQuantity: 0, GrossAmount: 4.38, DiscountAmount: 0, NetAmount: 4.38},
		{Product: "Actions", UnitType: "minutes", SKU: "Actions Linux", GrossQuantity: 200, NetQuantity: 190, DiscountQuantity: 10, GrossAmount: 1.20, DiscountAmount: 0.06, NetAmount: 1.14},
	}

	result := aggregatePackagesUsage(usageItems)

	if result.TotalGigabytesBandwidthUsed != 150 {
		t.Errorf("Expected TotalGigabytesBandwidthUsed to be 150, got %.2f", result.TotalGigabytesBandwidthUsed)
	}
	if result.TotalPaidGigabytesBandwidthUsed != 145 {
		t.Errorf("Expected TotalPaidGigabytesBandwidthUsed to be 145, got %.2f", result.TotalPaidGigabytesBandwidthUsed)
	}
	if result.IncludedGigabytesBandwidth != 5 {
		t.Errorf("Expected IncludedGigabytesBandwidth to be 5, got %.2f", result.IncludedGigabytesBandwidth)
	}
	// Use tolerance-based comparison for floating point
	tolerance := 0.001
	if result.NetAmount < 12.69-tolerance || result.NetAmount > 12.69+tolerance {
		t.Errorf("Expected NetAmount to be approximately 12.69, got %.10f", result.NetAmount)
	}
}

func Test_aggregateStorageUsage(t *testing.T) {
	usageItems := []UsageItem{
		{Product: "Actions", UnitType: "gigabyte-hours", SKU: "Actions storage", GrossQuantity: 7300, NetQuantity: 7200, DiscountQuantity: 100, GrossAmount: 2.45, DiscountAmount: 0.03, NetAmount: 2.42},
		{Product: "Packages", UnitType: "gigabyte-hours", SKU: "Packages storage", GrossQuantity: 3650, NetQuantity: 3650, DiscountQuantity: 0, GrossAmount: 1.23, DiscountAmount: 0, NetAmount: 1.23},
		{Product: "Actions", UnitType: "minutes", SKU: "Actions Linux", GrossQuantity: 100, NetQuantity: 95, DiscountQuantity: 5, GrossAmount: 0.60, DiscountAmount: 0.03, NetAmount: 0.57},
	}

	result := aggregateStorageUsage(usageItems)

	// Total gigabyte-hours: 7300 (Actions) + 3650 (Packages) = 10950
	if result.EstimatedStorageForMonth != 10950 {
		t.Errorf("Expected EstimatedStorageForMonth to be 10950, got %.2f", result.EstimatedStorageForMonth)
	}

	if result.ActionsStorageGB != 7300 {
		t.Errorf("Expected ActionsStorageGB to be 7300, got %.2f", result.ActionsStorageGB)
	}

	if result.PackagesStorageGB != 3650 {
		t.Errorf("Expected PackagesStorageGB to be 3650, got %.2f", result.PackagesStorageGB)
	}

	if result.NetAmount != 3.65 {
		t.Errorf("Expected NetAmount to be 3.65, got %.2f", result.NetAmount)
	}
}

func Test_aggregateStorageUsage_Empty(t *testing.T) {
	usageItems := []UsageItem{
		{Product: "Actions", UnitType: "minutes", SKU: "Actions Linux", GrossQuantity: 100, NetQuantity: 95, DiscountQuantity: 5, GrossAmount: 0.60, DiscountAmount: 0.03, NetAmount: 0.57},
	}

	result := aggregateStorageUsage(usageItems)

	if result.EstimatedStorageForMonth != 0 {
		t.Errorf("Expected EstimatedStorageForMonth to be 0, got %.2f", result.EstimatedStorageForMonth)
	}

	if result.ActionsStorageGB != 0 {
		t.Errorf("Expected ActionsStorageGB to be 0, got %.2f", result.ActionsStorageGB)
	}

	if result.PackagesStorageGB != 0 {
		t.Errorf("Expected PackagesStorageGB to be 0, got %.2f", result.PackagesStorageGB)
	}
}
