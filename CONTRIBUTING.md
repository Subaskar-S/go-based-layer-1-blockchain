# Contributing to Layer-1 Blockchain

Thank you for your interest in contributing to the Layer-1 Blockchain project!

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/layer1-blockchain.git
   cd layer1-blockchain
   ```
3. **Set up development environment**:
   ```bash
   make deps
   make dev-setup
   ```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

Use prefixes:
- `feature/` for new features
- `fix/` for bug fixes
- `docs/` for documentation
- `refactor/` for code refactoring
- `test/` for adding tests

### 2. Make Changes

- Write clean, idiomatic Go code
- Follow the existing code style
- Add tests for new functionality
- Update documentation as needed

### 3. Run Tests

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/consensus/...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 4. Run Linter

```bash
make lint
```

Fix any linting issues before submitting.

### 5. Format Code

```bash
make fmt
```

### 6. Commit Changes

Write clear, descriptive commit messages:

```
feat: add BLS signature aggregation

- Implement BLS signature scheme
- Add aggregation for validator signatures
- Update consensus to use aggregated signatures

Closes #123
```

Commit message format:
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation changes
- `test:` adding tests
- `refactor:` code refactoring
- `perf:` performance improvements
- `chore:` maintenance tasks

### 7. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Code Style Guidelines

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Keep functions small and focused
- Write descriptive variable names
- Add comments for exported functions
- Use interfaces for abstraction

Example:

```go
// ValidateBlock validates a block according to consensus rules.
// It checks the block header, transactions, and signatures.
// Returns an error if validation fails.
func (bc *Blockchain) ValidateBlock(block *types.Block) error {
    if block == nil {
        return errors.New("block is nil")
    }
    
    // Validate header
    if err := bc.validateHeader(block.Header); err != nil {
        return fmt.Errorf("invalid header: %w", err)
    }
    
    // Validate transactions
    for _, tx := range block.Transactions {
        if err := tx.Validate(); err != nil {
            return fmt.Errorf("invalid transaction: %w", err)
        }
    }
    
    return nil
}
```

### Testing

- Write table-driven tests
- Test edge cases and error conditions
- Use meaningful test names
- Mock external dependencies

Example:

```go
func TestValidateBlock(t *testing.T) {
    tests := []struct {
        name    string
        block   *types.Block
        wantErr bool
    }{
        {
            name:    "valid block",
            block:   createValidBlock(),
            wantErr: false,
        },
        {
            name:    "nil block",
            block:   nil,
            wantErr: true,
        },
        {
            name:    "invalid header",
            block:   createBlockWithInvalidHeader(),
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := bc.ValidateBlock(tt.block)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateBlock() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Documentation

- Add godoc comments for all exported types and functions
- Update README.md for user-facing changes
- Update architecture.md for design changes
- Add examples for new features

## Pull Request Guidelines

### Before Submitting

- [ ] Tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Code is formatted (`make fmt`)
- [ ] Documentation is updated
- [ ] Commit messages are clear
- [ ] Branch is up to date with main

### PR Description

Include:
- **What**: What does this PR do?
- **Why**: Why is this change needed?
- **How**: How does it work?
- **Testing**: How was it tested?
- **Breaking Changes**: Any breaking changes?

Example:

```markdown
## What
Adds BLS signature aggregation for validator signatures.

## Why
Reduces block size and improves verification performance by aggregating
validator signatures instead of storing individual signatures.

## How
- Implements BLS12-381 signature scheme
- Adds aggregation logic in consensus engine
- Updates block structure to store aggregated signature

## Testing
- Added unit tests for BLS operations
- Added integration test for multi-validator signing
- Benchmarked aggregation performance

## Breaking Changes
- Block structure changed (added AggregatedSignature field)
- Requires database migration for existing chains
```

### Review Process

1. Automated checks must pass (tests, linting)
2. At least one maintainer approval required
3. Address review comments
4. Squash commits if requested
5. Maintainer will merge when ready

## Reporting Issues

### Bug Reports

Include:
- **Description**: Clear description of the bug
- **Steps to Reproduce**: How to reproduce the issue
- **Expected Behavior**: What should happen
- **Actual Behavior**: What actually happens
- **Environment**: OS, Go version, etc.
- **Logs**: Relevant log output

### Feature Requests

Include:
- **Use Case**: Why is this feature needed?
- **Proposed Solution**: How should it work?
- **Alternatives**: Other approaches considered
- **Additional Context**: Any other relevant information

## Community

- Be respectful and inclusive
- Help others learn and grow
- Give constructive feedback
- Celebrate contributions

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Questions?

- Open an issue for questions
- Join our Discord/Slack (if available)
- Email maintainers (if provided)

Thank you for contributing! 🎉

