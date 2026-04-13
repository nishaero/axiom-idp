# Testing

Comprehensive test suite for Axiom IDP.

## Running Tests

### All tests
```bash
make test
```

### Backend tests only
```bash
go test -v -race ./...
```

### Frontend tests only
```bash
cd web && npm test
```

### With coverage
```bash
make coverage
```

### Integration tests
```bash
go test -v -race -tags=integration ./tests/...
```

## Test Structure

```
tests/
├── fixtures/        # Test data
├── helpers/         # Test utilities
└── integration/     # Integration tests

internal/
├── auth/
│   ├── *_test.go    # Unit tests
│   └── fixtures/    # Auth test data
├── catalog/
│   └── *_test.go
├── mcp/
│   └── *_test.go
└── ...
```

## Coverage Requirements

- **Minimum**: 80% overall
- **Security code**: 100%
- **Public APIs**: 100%
- **Critical paths**: 100%

## Testing Best Practices

1. Use table-driven tests for multiple cases
2. Test happy path and error paths
3. Include concurrency tests where applicable
4. Mock external dependencies
5. Keep tests focused and isolated
6. Use descriptive test names

## Example Test

```go
func TestFeature(t *testing.T) {
    cases := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "result",
            wantErr:  false,
        },
        {
            name:     "invalid input",
            input:    "",
            expected: "",
            wantErr:  true,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := Feature(tc.input)
            if (err != nil) != tc.wantErr {
                t.Errorf("got error %v, want %v", err, tc.wantErr)
            }
            if result != tc.expected {
                t.Errorf("got %s, want %s", result, tc.expected)
            }
        })
    }
}
```

## Continuous Integration

Tests run automatically on:
- Every push to `main` and `develop`
- Every pull request
- Scheduled daily security scans

See `.github/workflows/ci.yml` for details.
