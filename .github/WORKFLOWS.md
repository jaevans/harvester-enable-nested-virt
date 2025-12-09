# GitHub Actions Workflow

## Overview

This repository uses GitHub Actions to automatically test pull requests and ensure code quality.

## Workflows

### Test Workflow (`.github/workflows/test.yml`)

The test workflow runs automatically on:
- Pull requests to the `main` branch

#### Jobs

1. **Test Job**: Runs the test suite using `make test`
   - Sets up Go environment
   - Downloads dependencies
   - Runs all tests with race detection and coverage
   - Uploads coverage reports as artifacts

2. **Build Job**: Verifies the code can be built
   - Sets up Go environment
   - Downloads dependencies
   - Builds the webhook binary using `make build`
   - Verifies the binary was created successfully

3. **Lint Job**: Checks code quality
   - Runs `golangci-lint` for comprehensive linting
   - Verifies `go.mod` and `go.sum` are tidy

## Branch Protection

To ensure that only tested code is merged, configure the following branch protection rules for the `main` branch:

### Recommended Settings

1. Go to **Settings** → **Branches** → **Branch protection rules**
2. Click **Add rule** or edit existing rule
3. Set **Branch name pattern** to `main`
4. Enable the following settings:

#### Required Status Checks
- ✅ **Require status checks to pass before merging**
- ✅ **Require branches to be up to date before merging**
- Select the following status checks as required:
  - `Run Tests`
  - `Build Binary`
  - `Lint Code`

#### Additional Recommended Settings
- ✅ **Require a pull request before merging**
  - Set **Required number of approvals before merging** to at least 1 (if working with a team)
- ✅ **Require conversation resolution before merging**
- ✅ **Do not allow bypassing the above settings**

### Manual Setup Instructions

If you need to set up branch protection via GitHub UI:

1. Navigate to your repository on GitHub
2. Click on **Settings**
3. In the left sidebar, click on **Branches**
4. Under "Branch protection rules", click **Add rule**
5. In "Branch name pattern", enter `main`
6. Check **Require status checks to pass before merging**
7. Search for and select these status checks:
   - `Run Tests`
   - `Build Binary`
   - `Lint Code`
8. Click **Create** or **Save changes**

## Local Testing

Before pushing your code, you can run the same checks locally:

```bash
# Run tests
make test

# Build binary
make build

# Run linter
make lint

# Format code
make fmt

# Tidy dependencies
make tidy
```

## Troubleshooting

### Test Failures

If the test job fails:
1. Check the workflow logs in the **Actions** tab
2. Run `make test` locally to reproduce the issue
3. Fix the failing tests
4. Push your changes to trigger the workflow again

### Build Failures

If the build job fails:
1. Check the workflow logs for compilation errors
2. Run `make build` locally to reproduce the issue
3. Fix the build errors
4. Push your changes to trigger the workflow again

### Lint Failures

If the lint job fails:
1. Run `make lint` locally to see the issues
2. Fix the reported issues
3. Ensure `go mod tidy` doesn't change anything
4. Push your changes to trigger the workflow again

## Coverage Reports

Test coverage reports are automatically generated and uploaded as artifacts for each workflow run. You can:

1. Go to the **Actions** tab
2. Click on a workflow run
3. Scroll down to **Artifacts**
4. Download the `coverage-report` artifact
5. Use `go tool cover -html=coverage.out` to view the coverage report locally
