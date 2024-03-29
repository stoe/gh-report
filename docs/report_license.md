## report license

Report on GitHub Enterprise licensing

### Synopsis

Report on GitHub Enterprise licensing, requires `read:enterprise` and `user:email` scope

```
report license [flags]
```

### Options

```
  -h, --help   help for license
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

* [report](report.md)	 - gh cli extension to generate reports

