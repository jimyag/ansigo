# AnsiGo Test Suite

This directory contains comprehensive tests for the AnsiGo project, including unit tests, integration tests, and test automation scripts.

## Directory Structure

```
tests/
├── README.md                    # This file
├── inventory/                   # Test inventory files
│   └── test_hosts              # Test inventory configuration
├── playbooks/                   # Test playbook files
│   ├── test-jinja2-working.yml # Jinja2 working features test
│   ├── test-jinja2-filters.yml # Jinja2 filters test
│   ├── test-jinja2-loops.yml   # Jinja2 loops test
│   └── test-jinja2-advanced.yml# Jinja2 advanced features test
└── scripts/                     # Test automation scripts
    ├── run-all-tests.sh        # Run all tests (unit + integration)
    ├── run-integration-tests.sh# Run integration tests only
    └── setup-test-env.sh       # Setup test environment (Docker)
```

## Running Tests

### Quick Start

Run all tests (unit + integration):
```bash
./tests/scripts/run-all-tests.sh
```

### Unit Tests Only

Run unit tests for all packages:
```bash
go test ./pkg/... -v
```

Run tests with coverage:
```bash
go test ./pkg/... -v -cover
```

### Integration Tests Only

Run integration tests:
```bash
./tests/scripts/run-integration-tests.sh
```

## Test Coverage

Current test coverage:
- `pkg/inventory`: 54.1%
- `pkg/playbook`: 17.3%
- `pkg/module`: 5.8%

See unit test files for details.
