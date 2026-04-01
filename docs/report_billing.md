## report billing

Report on GitHub billing

### Synopsis

Report on GitHub billing for enterprises, organizations, and users.
Requires `read:enterprise`, `read:org`, and/or `read:user` scope.

Note: This command uses the new unified billing API endpoint (/settings/billing/usage)
introduced with GitHub's metered billing platform. The Advanced Security billing data
continues to use its dedicated endpoint.

```
report billing [flags]
```

### Output Fields

The billing report provides usage quantities and optional cost information for each account.

#### Actions (`--actions`)

- **action_minutes_used**: Total compute minutes used (gross quantity)
- **action_net_cost**: Net cost in USD after discounts (with `--show-costs`)
- **action_discount_amount**: Discount amount in USD (with `--show-costs`)

Minutes are broken down by runner type (Linux/Ubuntu, macOS, Windows) and aggregated from all SKUs.

#### Packages (`--packages`)

- **gigabytes_bandwidth_used**: Total package bandwidth in GB (gross quantity)
- **packages_net_cost**: Net cost in USD after discounts (with `--show-costs`)
- **packages_discount_amount**: Discount amount in USD (with `--show-costs`)

#### Storage (`--storage`)

- **estimated_storage_for_month**: Total storage in gigabyte-hours (Actions + Packages)
- **actions_storage_gb**: Actions-specific storage in gigabyte-hours (with `--show-costs`)
- **packages_storage_gb**: Packages-specific storage in gigabyte-hours (with `--show-costs`)
- **storage_net_cost**: Net storage cost in USD after discounts (with `--show-costs`)

Storage includes Actions artifacts/caches, Packages storage, and Git LFS (tracked under packages).

#### Advanced Security (`--security`)

- **advanced_security_committers**: Total number of active committers using GitHub Advanced Security

### Notes

- Cost fields represent monetary amounts in USD
- Quantity fields represent actual usage (minutes, gigabytes, gigabyte-hours)
- JSON output uses fixed decimal precision to avoid scientific notation for very small values
- Storage values under 1 GB-hour are displayed with high precision (10 decimal places)

### Options

```
      --actions        Get GitHub Actions billing
      --all            Get all billing data (default true)
  -h, --help           help for billing
      --month string   Billing month for storage data (MM, defaults to current month)
      --packages       Get GitHub Packages billing
      --security       Get GitHub Advanced Security active committers
      --show-costs     Show cost information (net, gross, discount amounts)
      --storage        Get shared storage billing
      --year string    Billing year for storage data (YYYY, defaults to current year)
```

### Options inherited from parent commands

```
      --csv string                   Path to CSV file, to save report to file
  -e, --enterprise read:enterprise   GitHub Enterprise Cloud account (requires read:enterprise scope)
      --hostname string              GitHub Enterprise Server hostname (default "github.com")
      --json string                  Path to JSON file, to save report to file
      --md string                    Path to MD file, to save report to file
      --no-cache                     Do not cache results for one hour (default "false")
  -o, --owner read:org               GitHub account organization (requires read:org scope) or user account (requires `n/a` scope)
  -r, --repo repo                    GitHub repository (owner/repo), requires repo scope
      --silent                       Do not print any output (default: "false")
  -t, --token string                 GitHub Personal Access Token (default "gh auth token")
```

### SEE ALSO

- [report](report.md) - gh cli extension to generate reports
