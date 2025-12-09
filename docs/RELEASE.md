# Release Process

This document describes how to create releases for the Harvester Nested Virtualization Webhook.

## Automated Release Process

When a semantic version tag is pushed to the repository, GitHub Actions automatically:

1. Builds binaries for linux/amd64 and linux/arm64
2. Creates multi-arch Docker images
3. Pushes images to GitHub Container Registry (ghcr.io)
4. Creates a GitHub release with:
   - Binary artifacts and checksums
   - Formatted changelog
   - Container image information

## Creating a Release

### 1. Ensure All Changes Are Merged

Make sure all intended changes are merged to the main branch and tests are passing.

### 2. Create and Push a Tag

Tags must follow semantic versioning (e.g., v1.0.0, v1.2.3, v2.0.0-beta.1):

```bash
# Create a tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag to trigger the release workflow
git push origin v1.0.0
```

### 3. Monitor the Release Workflow

The release workflow will automatically start. You can monitor it at:
- GitHub Actions: `https://github.com/jaevans/harvester-enable-nested-virt/actions`

The workflow takes approximately 5-10 minutes to complete.

### 4. Verify the Release

Once complete, verify:

1. **GitHub Release**: Check https://github.com/jaevans/harvester-enable-nested-virt/releases
   - Release should include binary artifacts and checksums
   - Changelog should be properly formatted

2. **Container Images**: Verify images are available
   ```bash
   # Pull the multi-arch image
   docker pull ghcr.io/jaevans/harvester-nested-virt-webhook:v1.0.0
   
   # Verify it works
   docker run --rm ghcr.io/jaevans/harvester-nested-virt-webhook:v1.0.0 --version
   ```

## Container Images

Each release produces the following container images:

- `ghcr.io/jaevans/harvester-nested-virt-webhook:<version>` - Multi-arch manifest
- `ghcr.io/jaevans/harvester-nested-virt-webhook:<version>-amd64` - AMD64 specific
- `ghcr.io/jaevans/harvester-nested-virt-webhook:<version>-arm64` - ARM64 specific
- `ghcr.io/jaevans/harvester-nested-virt-webhook:latest` - Latest release (multi-arch)

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version (v2.0.0): Breaking changes
- **MINOR** version (v1.1.0): New features, backwards compatible
- **PATCH** version (v1.0.1): Bug fixes, backwards compatible

### Pre-release Versions

For pre-release versions, append a pre-release identifier:

- `v1.0.0-alpha.1` - Alpha release
- `v1.0.0-beta.1` - Beta release
- `v1.0.0-rc.1` - Release candidate

Pre-releases are automatically marked as such in GitHub releases.

## Troubleshooting

### Release Workflow Fails

1. Check the workflow logs in GitHub Actions
2. Common issues:
   - Invalid tag format (must be v*.*.*)
   - Build failures (run `make test` and `make build` locally first)
   - Docker registry authentication issues
   - Network issues during Docker push

### Deleting a Release

If you need to delete a release:

```bash
# Delete the GitHub release (via web UI)
# Then delete the tag
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0
```

Note: Container images pushed to GHCR must be deleted separately via the GitHub Packages UI.

## GoReleaser Configuration

The release process is managed by GoReleaser. Configuration is in `.goreleaser.yaml`.

To test the release locally without publishing:

```bash
# Install goreleaser (if not already installed)
# See: https://goreleaser.com/install/

# Build a snapshot (no tag required)
goreleaser build --snapshot --clean

# Full release dry-run
goreleaser release --snapshot --clean
```

## Forking

If you fork this repository and want to publish to your own registry:

1. Update `.goreleaser.yaml`:
   - Change `jaevans` to your GitHub username/org in all `image_templates`
   - Update the `owner` in the `release` section

2. Ensure GitHub Actions has the necessary permissions:
   - Repository Settings → Actions → General → Workflow permissions
   - Enable "Read and write permissions"
