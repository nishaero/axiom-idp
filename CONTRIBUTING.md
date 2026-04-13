# Contributing to Axiom IDP

We welcome contributions! This document provides guidelines for contributing to the project.

## Code of Conduct

We are committed to providing a welcoming and inspiring community. Please be respectful and constructive.

## Getting Started

### Prerequisites
- Go 1.22+
- Node.js 18+
- Git
- Make

### Setup Development Environment

```bash
# Clone the repository
git clone https://github.com/axiom-idp/axiom-idp.git
cd axiom-idp

# Create feature branch
git checkout -b feature/my-feature

# Install dependencies
go mod download
cd web && npm install

# Run tests
make test

# Start development
make dev
```

## Development Workflow

### 1. Create an Issue

Before starting work on a feature, create an issue describing:
- What problem does it solve?
- Proposed solution
- Alternative approaches
- Expected behavior

### 2. Fork and Branch

```bash
# Create descriptive branch names
git checkout -b fix/issue-123-description
git checkout -b feature/issue-456-description
git checkout -b docs/improve-readme
```

### 3. Code Changes

Follow these guidelines:

#### Go Code
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Use `gofmt` and `goimports` for formatting
- Run `golangci-lint run` before committing
- Add comments for exported functions
- Handle errors explicitly

```go
// Good
func (s *Server) Start(ctx context.Context) error {
	if err := s.validateConfig(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return s.listen(ctx)
}

// Every exported function should have a comment
// Start initializes and starts the server.
```

#### TypeScript/React
- Use Prettier for formatting
- Follow [React best practices](https://react.dev)
- Use TypeScript strict mode
- Avoid `any` types
- Add JSDoc comments for complex functions

```typescript
/**
 * AIAssistant component for querying Axiom IDP
 * @param props Component props
 * @returns React component
 */
export const AIAssistant: React.FC<AIAssistantProps> = (props) => {
  // Implementation
};
```

#### Commit Messages

Use conventional commits:

```
feat: add MCP server hot reload
fix: handle null context in AI router
docs: improve API documentation
test: add integration tests for catalog
refactor: simplify auth middleware
chore: update dependencies
```

### 4. Testing

Write tests for all changes:

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/mcp/...

# Run with coverage
make coverage

# Run frontend tests
cd web && npm test
```

**Coverage Requirements**:
- Minimum 80% overall coverage
- 100% for security-sensitive code
- 100% for public APIs

### 5. Commit and Push

```bash
# Commit with descriptive message
git commit -m "feat: add user preferences API endpoint"

# Push to your fork
git push origin feature/my-feature
```

## Pull Request Process

### 1. Open PR

- Use descriptive title (conventional commit style)
- Reference related issues: "Closes #123"
- Describe changes and testing
- Include screenshots for UI changes

### 2. PR Template

```markdown
## Description
Brief description of changes

## Type
- [ ] Feature
- [ ] Bug Fix
- [ ] Documentation
- [ ] Performance
- [ ] Security

## Related Issues
Closes #123

## Testing
- [ ] Unit tests added
- [ ] Integration tests added
- [ ] Manual testing completed
- [ ] No breaking changes

## Checklist
- [ ] Code follows style guidelines
- [ ] Tests pass locally
- [ ] No hardcoded credentials
- [ ] Documentation updated
- [ ] Commit messages follow convention
```

### 3. Code Review

- All PRs require review from CODEOWNERS
- Address review comments promptly
- Re-request review after making changes
- PRs must pass all CI checks before merge

### 4. Merge

Once approved and all CI passes:
- Maintainer will squash and merge
- Automatic deployment to staging
- Production deployment on release

## Areas for Contribution

### High Priority
- [ ] MCP server implementations
- [ ] Performance optimizations
- [ ] Security improvements
- [ ] Documentation
- [ ] Test coverage

### Backend (Go)
- MCP registry enhancements
- Catalog indexing improvements
- AI router optimization
- Auth provider implementations
- API endpoints

### Frontend (React)
- Component library expansion
- UI/UX improvements
- Accessibility enhancements
- Performance optimization
- Mobile responsiveness

### Infrastructure
- Kubernetes manifests
- Docker configurations
- CI/CD workflows
- Monitoring and observability

## Style Guide

### Go
```bash
# Format code
gofmt -s -w .

# Run linter
golangci-lint run

# Get imports in order
goimports -w .
```

### TypeScript/React
```bash
# Format code
npx prettier --write .

# Run linter
npm run lint

# Type check
npm run type-check
```

## Testing Standards

### Unit Tests
```go
func TestServerStart(t *testing.T) {
	server := NewServer(&Config{Port: 8080})
	
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer server.Stop()
	
	// Assertions
}
```

### Integration Tests
```go
func TestCatalogIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	
	// Test end-to-end flows
}
```

## Documentation

All PRs should update relevant documentation:

- Code comments for complex logic
- README if adding features
- docs/ for architecture changes
- API docs for new endpoints
- Changelog entry (if user-facing)

## Security Guidelines

Never commit:
- Passwords or API keys
- Private credentials
- Internal URLs or IPs
- Sensitive configuration

Use environment variables and `.gitignore`:
```bash
# .env files
.env
.env.local
*.key
*.pem

# IDE
.idea/
.vscode/
*.swp
```

## Performance Considerations

When contributing:
- Profile code for bottlenecks
- Use benchmarks for critical paths
- Consider memory usage
- Optimize database queries
- Cache appropriately

```go
// Add benchmarks for performance-critical code
func BenchmarkCatalogQuery(b *testing.B) {
	catalog := setupTestCatalog()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		catalog.Search("service")
	}
}
```

## Questions?

- 📖 Check [docs](./docs)
- 💬 GitHub Discussions
- 📧 Email: dev@axiom-idp.dev
- 🐛 GitHub Issues

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.

---

Thank you for contributing to Axiom IDP! 🚀
