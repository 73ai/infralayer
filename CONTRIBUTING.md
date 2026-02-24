# Contributing to InfraLayer

Thank you for your interest in contributing to InfraLayer\! This document provides guidelines and instructions for contributing to the project.

## Development Setup

1. Clone the repository:
   ```
   git clone https://github.com/73ai/infralayer.git
   cd infralayer
   ```

2. Install in development mode:
   ```
   pip install -e .
   ```

3. Install development dependencies:
   ```
   pip install build twine pytest
   ```

## Versioning and Releases

The project includes a helper script for versioning:

```bash
# Bump the patch version (0.1.0 -> 0.1.1)
./bump_version.py

# Bump the minor version (0.1.0 -> 0.2.0)
./bump_version.py minor

# Bump the major version (0.1.0 -> 1.0.0)
./bump_version.py major

# Bump and create a git commit and tag
./bump_version.py --commit
```

## CI/CD

This project uses GitHub Actions for CI/CD:

1. **Tests Workflow**: Runs on every PR and push to master
   - Installs the package
   - Verifies it can be imported
   - Checks package structure

2. **Publish Workflow**: Automatically publishes to PyPI
   - Triggers on pushes to the master branch
   - Requires PyPI secrets to be configured in GitHub

### Setting up PyPI Publishing

1. Create an API token on PyPI:
   - Go to https://pypi.org/manage/account/token/
   - Create a token with "Entire account (all projects)" scope
2. Add the token as a GitHub secret:
   - Go to your GitHub repository → Settings → Secrets and variables → Actions
   - Create a new repository secret named `PYPI_API_TOKEN`
   - Paste your PyPI API token as the value

## Local Development

### Manual Publishing

The repository includes a `publish.sh` script to manually build and publish the package to PyPI:

```bash
./publish.sh
```

This script will:
1. Clean any previous builds
2. Build the distribution packages
3. Check the packages for errors
4. Prompt for confirmation before uploading to PyPI
5. Use API token authentication for PyPI

## Pull Request Process

1. Fork the repository and create a feature branch
2. Make your changes, ensuring they follow the project's code style
3. Add tests for any new functionality
4. Ensure all tests pass
5. Update documentation as needed
6. Submit a pull request to the `master` branch

## Code Style

This project follows standard Python code style conventions:
- Use 4 spaces for indentation
- Maximum line length of 88 characters (using Black formatting standards)
- Include docstrings for functions and classes

## License

By contributing to this project, you agree that your contributions will be licensed under the project's GNU GPL license.
