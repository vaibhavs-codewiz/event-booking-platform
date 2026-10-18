// Package middleware provides HTTP middleware for the Gin router.
//
// Phase 1: intentionally empty.
//
// Phase 2 scope:
//   - JWT authentication middleware (verifies token, loads identity + role)
//   - role-based authorization middleware (RequireRole)
//   - request id / correlation id propagation
//   - structured request logging
package middleware
