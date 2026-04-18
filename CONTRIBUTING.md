# Contributing to taito CLI

First, thank you for your interest in contributing! taito is an open-source project, and we welcome contributions—whether they be bug reports, feature requests, documentation improvements, or code pull requests.

## Getting Started

taito is written in [Go](https://golang.org/). To contribute code, you'll need a basic Go development environment.

### Prerequisites
* Go 1.25 or higher
* `git`

### Setting up your environment

1. **Fork the repository** on GitHub.
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/<your-username>/taito.git
   cd taito
   ```
3. **Install dependencies**:
   ```bash
   go mod download
   ```

## Development Workflow

### Building the CLI
You can compile the CLI into a binary for testing your changes locally:
```bash
go build -o taito main.go
```

### Running Tests
We expect all tests to pass before merging a PR. To run the test suite:
```bash
go test ./...
```
*If you are adding new features, please write corresponding tests.*

If you use [`just`](https://github.com/casey/just), you can also run the common development tasks through the project `justfile`:
```bash
just test
just complexity
just complexity-over 15
just review
```

### Code Style
We follow standard Go conventions. Please format your code before committing:
```bash
go fmt ./...
```
You can also use tools like `golangci-lint` to ensure your code is clean and idiomatic.
For maintainability reviews, `just complexity` and `just complexity-over 15` use `gocognit` to highlight the most cognitively complex functions first.

## Submitting Pull Requests

1. **Create a branch** for your feature or bug fix:
   ```bash
   git checkout -b feature/my-new-feature
   ```
   or
   ```bash
   git checkout -b fix/issue-123
   ```
2. **Commit your changes**: Keep your commit messages clear and concise.
3. **Push to your fork**:
   ```bash
   git push origin feature/my-new-feature
   ```
4. **Open a Pull Request**: Navigate to the original taito repository on GitHub and open a Pull Request. Provide a clear description of the problem you're solving and how your changes address it.

## Issues and Feature Requests

If you don't want to write code but have ideas or found a bug, please [open an issue](https://github.com/taito-project/taito-cli/issues) (or the corresponding link for your repository).
* **Bug Reports**: Include steps to reproduce, the expected behavior, and what actually happened. Mention your OS and taito version.
* **Feature Requests**: Describe the feature, why you need it, and how it should work.

Thank you for helping make taito better!
