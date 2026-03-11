# Release and Publish Docker Images

This guide explains how to publish NSW container images using the release workflow.

Workflow file: `.github/workflows/release.yml`

## Published Images

The workflow builds and publishes these images to GHCR:

- `ghcr.io/<owner>/<repo>/nsw-backend`
- `ghcr.io/<owner>/<repo>/nsw-oga-backend`
- `ghcr.io/<owner>/<repo>/nsw-trader-portal`
- `ghcr.io/<owner>/<repo>/nsw-oga-portal`

## Triggers

The release workflow supports:

1. Tag push with SemVer format: `v*.*.*` (example: `v1.2.0`)
2. Manual trigger (`workflow_dispatch`) with a required `version` input

## Tagging Strategy

For each image, tags are generated as follows:

- SemVer tags on tag-push releases:
  - `<major>.<minor>.<patch>`
  - `<major>.<minor>`
  - `<major>`
- SHA tag:
  - `sha-<full-commit-sha>`
- `latest`:
  - published only for SemVer tag pushes and manual dispatches from `main`
- Manual dispatch:
  - uses the provided `version` input as a raw tag

## Release Modes

The workflow has two modes:

1. Release mode
- Pushes images to GHCR
- Enabled for SemVer tag pushes
- Enabled for manual dispatch from `main`

2. Dry run mode
- Builds images only
- Does not push images
- Active for manual dispatches from branches other than `main`

## How to Run a Release

## Option 1: Tag Push (recommended)

From your local clone:

```bash
git checkout main
git pull origin main
git tag v1.2.0
git push origin v1.2.0
```

## Option 2: Manual Dispatch

1. Open Actions in GitHub
2. Select `Release - Build and Publish All Images`
3. Click `Run workflow`
4. Provide `version` (example: `v1.2.0`)

## Pull Examples

Replace `<owner>/<repo>` and `<version>` accordingly:

```bash
docker pull ghcr.io/<owner>/<repo>/nsw-backend:<version>
docker pull ghcr.io/<owner>/<repo>/nsw-oga-backend:<version>
docker pull ghcr.io/<owner>/<repo>/nsw-trader-portal:<version>
docker pull ghcr.io/<owner>/<repo>/nsw-oga-portal:<version>
```

## Security and Metadata

The release workflow includes:

- OCI image metadata labels (source, description, version, revision, created)
- Trivy image scanning for high and critical vulnerabilities
- SARIF upload to GitHub Security

## Notes

- Ensure package publishing permissions are available (`packages: write`).
- The workflow currently uses GHCR (`ghcr.io`).
- For PR-only image validation (without publishing), use `.github/workflows/docker-validation.yml`.
