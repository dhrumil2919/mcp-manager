# GitHub Workflows Documentation

This document explains the automated workflows set up for the MCP Manager project.

## Overview

The project uses GitHub Actions for continuous integration, automated releases, and dependency management. Here's what each workflow does:

## Workflows

### 1. CI Workflow (`.github/workflows/ci.yml`)

**Triggers:** Push to `main`/`develop`, Pull Requests to `main`

**Purpose:** Continuous Integration - ensures code quality and functionality

**Jobs:**
- **Test**: Runs unit tests with coverage reporting
- **Lint**: Code quality checks with golangci-lint
- **Build**: Compiles the binary and verifies it works
- **Security**: Security scanning with Gosec and Trivy

**Features:**
- Go module caching for faster builds
- Coverage reporting to Codecov
- Security vulnerability scanning
- Build artifact upload

### 2. Auto Tag Workflow (`.github/workflows/auto-tag.yml`)

**Triggers:** 
- Push to `main` (automatic based on conventional commits)
- Manual trigger with version type selection

**Purpose:** Automatically creates version tags based on commit messages

**Features:**
- **Conventional Commits**: Analyzes commit messages to determine version bump
  - `feat:` → minor version bump
  - `fix:` → patch version bump  
  - `!` (breaking change) → major version bump
- **Manual Override**: Allows manual version bumping via workflow dispatch
- **Prerelease Support**: Can create prerelease versions
- **Validation**: Runs tests and builds before creating tags

**Conventional Commit Examples:**
```bash
git commit -m "feat: add new deployment option"     # → minor bump
git commit -m "fix: resolve Docker connection issue" # → patch bump
git commit -m "feat!: change CLI interface"         # → major bump
```

### 3. Release Workflow (`.github/workflows/release.yml`)

**Triggers:** Push of version tags (e.g., `v1.0.0`)

**Purpose:** Builds and publishes releases when tags are created

**Jobs:**
- **Test**: Final validation before release
- **Build and Release**: Cross-platform compilation and GitHub release creation
- **Docker Release**: Multi-architecture Docker image building and publishing

**Features:**
- Cross-platform binaries (Linux, macOS, Windows - AMD64/ARM64)
- Automatic changelog generation
- GitHub release creation with assets
- Docker image publishing to Docker Hub and GitHub Container Registry
- Release archive creation

### 4. Dependabot Auto-merge (`.github/workflows/dependabot-auto-merge.yml`)

**Triggers:** Dependabot pull requests

**Purpose:** Automatically handles dependency updates

**Features:**
- **Auto-merge**: Patch updates are automatically merged after tests pass
- **Auto-approve**: Minor updates are auto-approved for manual review
- **Manual Review**: Major updates require manual review with warning comment

### 5. Dependabot Configuration (`.github/dependabot.yml`)

**Purpose:** Configures automatic dependency updates

**Monitors:**
- Go modules (weekly)
- GitHub Actions (weekly)
- Docker base images (weekly)

## Usage Guide

### Automatic Releases

1. **Make changes** and commit using conventional commit format:
   ```bash
   git commit -m "feat: add health check timeout option"
   git commit -m "fix: resolve port binding issue"
   git commit -m "docs: update installation guide"
   ```

2. **Push to main**:
   ```bash
   git push origin main
   ```

3. **Auto-tag workflow** analyzes commits and creates appropriate version tag

4. **Release workflow** automatically triggers and:
   - Builds cross-platform binaries
   - Creates GitHub release with changelog
   - Publishes Docker images

### Manual Releases

1. **Trigger auto-tag workflow** manually:
   - Go to Actions → Auto Tag → Run workflow
   - Select version bump type (patch/minor/major)
   - Optionally create prerelease

2. **Release workflow** automatically triggers after tag creation

### Development Workflow

1. **Create feature branch**:
   ```bash
   git checkout -b feature/new-feature
   ```

2. **Make changes** and commit:
   ```bash
   git commit -m "feat: implement new feature"
   ```

3. **Push and create PR**:
   ```bash
   git push origin feature/new-feature
   ```

4. **CI workflow** runs automatically on PR:
   - Tests, linting, building, security scanning
   - Must pass before merge

5. **Merge to main** triggers auto-tag workflow

## Required Secrets

Configure these secrets in your GitHub repository settings:

### For Docker Publishing
- `DOCKER_USERNAME`: Docker Hub username
- `DOCKER_PASSWORD`: Docker Hub password or access token

### For Enhanced Features (Optional)
- `CODECOV_TOKEN`: For coverage reporting
- Personal Access Token with appropriate permissions for auto-tagging

## Workflow Status

Check workflow status:
- **CI Badge**: Shows build status for main branch
- **Release Badge**: Shows latest release build status
- **Actions Tab**: Detailed workflow run information

## Customization

### Modify Auto-tagging Rules

Edit `.github/workflows/auto-tag.yml`:
- Change conventional commit patterns
- Modify version bump logic
- Add custom validation steps

### Customize Release Assets

Edit `.github/workflows/release.yml`:
- Add/remove target platforms
- Modify release archive format
- Change Docker image repositories

### Adjust CI Pipeline

Edit `.github/workflows/ci.yml`:
- Add/remove testing steps
- Modify security scanning tools
- Change Go version or dependencies

## Best Practices

### Commit Messages
Use conventional commits for automatic versioning:
```bash
feat: add new feature
fix: bug fix
docs: documentation update
style: formatting changes
refactor: code refactoring
test: add tests
chore: maintenance tasks
```

### Branch Protection
Recommended branch protection rules for `main`:
- Require PR reviews
- Require status checks (CI workflow)
- Require up-to-date branches
- Restrict pushes to main

### Release Management
- Use semantic versioning (semver)
- Tag releases consistently (v1.0.0 format)
- Write meaningful release notes
- Test releases thoroughly

## Troubleshooting

### Common Issues

1. **Auto-tag not triggering**:
   - Check commit message format
   - Verify workflow permissions
   - Check if there are commits since last tag

2. **Release build failing**:
   - Check Go version compatibility
   - Verify all dependencies are available
   - Check cross-compilation issues

3. **Docker publish failing**:
   - Verify Docker Hub credentials
   - Check repository permissions
   - Verify Dockerfile syntax

### Debug Steps

1. **Check workflow logs** in Actions tab
2. **Verify secrets** are configured correctly
3. **Test locally** with same commands as workflow
4. **Check permissions** for GitHub token

## Monitoring

Monitor your workflows:
- Set up notifications for workflow failures
- Review dependency update PRs regularly
- Monitor security scan results
- Check release download metrics

This automated setup ensures consistent, reliable releases while maintaining code quality and security standards.
