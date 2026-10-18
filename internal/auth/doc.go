// Package auth provides authentication and authorization primitives.
//
// Phase 1: intentionally empty. The package exists so the import path and
// directory are stable before the feature lands.
//
// Phase 2 scope:
//   - JWT access token signing and verification (HS256)
//   - bcrypt password hashing and verification
//   - role claims ("organizer", "customer")
//   - token refresh strategy
package auth
