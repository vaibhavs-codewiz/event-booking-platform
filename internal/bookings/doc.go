// Package bookings implements the booking domain.
//
// Phase 1: intentionally empty.
//
// Phase 2 scope:
//   - create a booking while preserving event capacity under concurrency
//   - enqueue a booking confirmation email task (Asynq)
//   - cancel a booking and return seats to the event
//   - list a customer's bookings and an event's bookings
package bookings
