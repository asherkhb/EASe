# Testing Report

## Framework

EASe uses the Go standard `testing` package with the `stretchr/testify` helpers for clearer assertions.

## Coverage

- **App discovery scanning**
  - Validates grouping behavior for nested directories.
  - Verifies default app names when `spec.yml` omits a `name`.

## Running tests

```
go test ./...
```
