## report

gh cli extension to generate reports

### Synopsis

gh cli extension to generate enterprise/organization/user/repository reports

### Options

```
      --csv string                   Path to CSV file, to save report to file
  -e, --enterprise read:enterprise   GitHub Enterprise Cloud account (requires read:enterprise scope)
  -h, --help                         help for report
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

* [report actions](report_actions.md)	 - Report on GitHub Actions
* [report billing](report_billing.md)	 - Report on GitHub billing
* [report license](report_license.md)	 - Report on GitHub Enterprise licensing
* [report repo](report_repo.md)	 - Report on GitHub repositories
* [report verified-emails](report_verified-emails.md)	 - List enterprise/organization members' verified emails

