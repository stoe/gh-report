package cmd

import (
	"fmt"
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

	t.Run("non-storage billing returns empty string", func(t *testing.T) {
		billingMonth = ""
		billingYear = ""
		result := buildBillingQueryParams(false)
		if result != "" {
			t.Errorf("Expected empty string for non-storage, got %s", result)
		}
	})

	t.Run("storage billing with no params defaults to current month and year", func(t *testing.T) {
		billingMonth = ""
		billingYear = ""
		result := buildBillingQueryParams(true)

		now := time.Now()
		expectedMonth := now.Format("2006-01")
		expectedYear := now.Format("2006")

		if !strings.Contains(result, "month="+expectedMonth) {
			t.Errorf("Expected result to contain month=%s, got %s", expectedMonth, result)
		}
		if !strings.Contains(result, "year="+expectedYear) {
			t.Errorf("Expected result to contain year=%s, got %s", expectedYear, result)
		}
	})

	t.Run("storage billing with custom month and year", func(t *testing.T) {
		billingMonth = "12"
		billingYear = "2025"
		result := buildBillingQueryParams(true)

		if !strings.Contains(result, "month=2025-12") {
			t.Errorf("Expected result to contain month=2025-12, got %s", result)
		}
		if !strings.Contains(result, "year=2025") {
			t.Errorf("Expected result to contain year=2025, got %s", result)
		}
	})

	t.Run("storage billing with only custom month defaults to current year", func(t *testing.T) {
		billingMonth = "06"
		billingYear = ""
		result := buildBillingQueryParams(true)

		now := time.Now()
		expectedYear := now.Format("2006")
		expectedMonth := expectedYear + "-06"

		if !strings.Contains(result, "month="+expectedMonth) {
			t.Errorf("Expected result to contain month=%s, got %s", expectedMonth, result)
		}
		if !strings.Contains(result, "year="+expectedYear) {
			t.Errorf("Expected result to contain year=%s, got %s", expectedYear, result)
		}
	})

	t.Run("storage billing with only custom year", func(t *testing.T) {
		billingMonth = ""
		billingYear = "2024"
		result := buildBillingQueryParams(true)

		now := time.Now()
		expectedMonth := now.Format("01") // Current month in MM format
		expectedMonthParam := fmt.Sprintf("2024-%s", expectedMonth)

		if !strings.Contains(result, "month="+expectedMonthParam) {
			t.Errorf("Expected result to contain month=%s, got %s", expectedMonthParam, result)
		}
		if !strings.Contains(result, "year=2024") {
			t.Errorf("Expected result to contain year=2024, got %s", result)
		}
	})
}

func Test_aggregateActionsUsage(t *testing.T) {
	usageItems := []UsageItem{
		{Product: "Actions", UnitType: "minutes", SKU: "Actions Linux", Quantity: 100},
		{Product: "Actions", UnitType: "minutes", SKU: "Actions macOS", Quantity: 50},
		{Product: "Actions", UnitType: "minutes", SKU: "Actions Windows", Quantity: 75},
		{Product: "Packages", UnitType: "gigabytes", SKU: "Packages data transfer", Quantity: 10},
	}

	result := aggregateActionsUsage(usageItems)

	if result.TotalMinutesUsed != 225 {
		t.Errorf("Expected TotalMinutesUsed to be 225, got %.2f", result.TotalMinutesUsed)
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
		{Product: "Packages", UnitType: "gigabytes", SKU: "Packages data transfer", Quantity: 100},
		{Product: "Packages", UnitType: "gigabytes", SKU: "Packages data transfer", Quantity: 50},
		{Product: "Actions", UnitType: "minutes", SKU: "Actions Linux", Quantity: 200},
	}

	result := aggregatePackagesUsage(usageItems)

	if result.TotalGigabytesBandwidthUsed != 150 {
		t.Errorf("Expected TotalGigabytesBandwidthUsed to be 150, got %.2f", result.TotalGigabytesBandwidthUsed)
	}
}

func Test_aggregateStorageUsage(t *testing.T) {
	usageItems := []UsageItem{
		{Product: "Actions", UnitType: "GigabyteHours", SKU: "Actions storage", Quantity: 7300},
		{Product: "Packages", UnitType: "GigabyteHours", SKU: "Packages storage", Quantity: 3650},
		{Product: "Actions", UnitType: "minutes", SKU: "Actions Linux", Quantity: 100},
	}

	result := aggregateStorageUsage(usageItems)

	// Total GigabyteHours: 7300 (Actions) + 3650 (Packages) = 10950
	if result.EstimatedStorageForMonth != 10950 {
		t.Errorf("Expected EstimatedStorageForMonth to be 10950, got %.2f", result.EstimatedStorageForMonth)
	}

	if result.ActionsStorageGB != 7300 {
		t.Errorf("Expected ActionsStorageGB to be 7300, got %.2f", result.ActionsStorageGB)
	}

	if result.PackagesStorageGB != 3650 {
		t.Errorf("Expected PackagesStorageGB to be 3650, got %.2f", result.PackagesStorageGB)
	}
}

func Test_aggregateStorageUsage_Empty(t *testing.T) {
	usageItems := []UsageItem{
		{Product: "Actions", UnitType: "minutes", SKU: "Actions Linux", Quantity: 100},
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
