# gh-report

[![build](https://github.com/stoe/gh-report/actions/workflows/build.yml/badge.svg)](https://github.com/stoe/gh-report/actions/workflows/build.yml) [![codeql](https://github.com/stoe/gh-report/actions/workflows/codeql.yml/badge.svg)](https://github.com/stoe/gh-report/actions/workflows/codeql.yml) [![release](https://github.com/stoe/gh-report/actions/workflows/release.yml/badge.svg)](https://github.com/stoe/gh-report/actions/workflows/release.yml)

> gh cli extension to generate account/organization/enterprise reports

## Install

```bash
$ gh extension install stoe/gh-report
```

## Usage

```txt
$ gh report [command] [flags]
```

```txt
gh cli extension to generate organization/user/repository reports

Usage:
  gh report [command]

Available Commands:
  actions     Report on GitHub Actions [permissions|uses]
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  repo        Report on GitHub repositories

Flags:
      --csv string        Path to CSV file
  -h, --help              help for gh report
      --hostname string   GitHub Enterprise Server hostname
  -o, --owner string      GitHub account (organization or user account)
  -r, --repo string       GitHub repository (owner/repo)
  -v, --version           version for gh report

Use "gh report [command] --help" for more information about a command.
```

## License

- [MIT](./license) (c) [Stefan Stölzle](https://github.com/stoe)
- [Code of Conduct](./.github/code_of_conduct.md)
