# Local Build State

Note: When applying PRs from forks, cherry-pick the specific commits (`gh pr view <PR#> --json commits`) rather than merging the branch, which may include unrelated commits.

## Base
upstream/main: v0.49.3 (754192f4)

## Applied PRs

| PR | Commits | Description |
|----|---------|-------------|
| #1431 | 16f3f377 | fix(dolt): work around merge join panic in ready_issues view |
| #1436 | 61da49c9 | fix(dolt): remove FK constraint on depends_on_id for external refs |
| #1454 | 0c3ebae4 | feat(config): support .beads/config.local.yaml for local overrides |

## Build from Source

Building requires CGO for Dolt support:

```bash
# Debian/Ubuntu dependencies
sudo apt-get install -y gcc g++ libzstd-dev libicu-dev

# Build and install (ldflags set version info)
VERSION=$(git describe --tags --always --dirty | sed 's/^v//') && \
COMMIT=$(git rev-parse --short HEAD) && \
go install -ldflags "-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.Build=local" ./cmd/bd
```

Note: Installs to ~/go/bin. Ensure ~/go/bin is in PATH.
