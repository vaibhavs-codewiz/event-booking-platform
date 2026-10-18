// Package events implements the event management domain.
//
// Phase 1: intentionally empty.
//
// Phase 2 scope:
//   - event CRUD restricted to users with the "organizer" role
//   - capacity management (see concurrency strategy in README)
//   - atomic, overbooking-safe seat reservation via a conditional UPDATE
//   - event update tracking (to fan out notifications)
package events
