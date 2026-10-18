-- ===========================================================================
-- 001_init.up.sql - Initial schema for the Event Booking Platform
-- ===========================================================================
-- Phase 1: this schema is planned/documented. It is NOT applied automatically
-- by the application yet. Apply it manually for local development, e.g.:
--     psql "$DATABASE_URL" -f migrations/001_init.up.sql
-- or with the migration tool chosen in Phase 2 (golang-migrate).
-- ===========================================================================

BEGIN;

-- users: both organizers and customers. role drives authorization.
CREATE TABLE users (
    id            uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    email         text         NOT NULL,
    password_hash text         NOT NULL,
    role          text         NOT NULL CHECK (role IN ('organizer', 'customer')),
    name          text,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX users_email_uidx ON users (lower(email));

-- events: owned by an organizer. capacity + tickets_booked track availability.
CREATE TABLE events (
    id             uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id   uuid         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title          text         NOT NULL,
    description    text,
    venue          text,
    start_time     timestamptz  NOT NULL,
    end_time       timestamptz,
    capacity       integer      NOT NULL CHECK (capacity > 0),
    tickets_booked integer      NOT NULL DEFAULT 0 CHECK (tickets_booked >= 0),
    status         text         NOT NULL DEFAULT 'active'
                                 CHECK (status IN ('active', 'cancelled', 'completed')),
    created_at     timestamptz  NOT NULL DEFAULT now(),
    updated_at     timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX events_organizer_idx   ON events (organizer_id);
CREATE INDEX events_start_time_idx  ON events (start_time);
-- Defense-in-depth: the atomic conditional UPDATE (see README) guarantees
-- tickets_booked never exceeds capacity at write time. This CHECK guards
-- against any path that bypasses that UPDATE.
ALTER TABLE events ADD CONSTRAINT events_capacity_gte_booked
    CHECK (tickets_booked <= capacity);

-- bookings: a customer's reservation of N seats for an event.
CREATE TABLE bookings (
    id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id    uuid         NOT NULL REFERENCES events(id) ON DELETE RESTRICT,
    customer_id uuid         NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    seats       integer      NOT NULL CHECK (seats > 0),
    status      text         NOT NULL DEFAULT 'confirmed'
                              CHECK (status IN ('confirmed', 'cancelled')),
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX bookings_event_idx    ON bookings (event_id);
CREATE INDEX bookings_customer_idx  ON bookings (customer_id);
-- One active booking per customer per event. Cancelling re-allows a new one.
CREATE UNIQUE INDEX bookings_event_customer_active_uidx
    ON bookings (event_id, customer_id) WHERE status = 'confirmed';

COMMIT;
