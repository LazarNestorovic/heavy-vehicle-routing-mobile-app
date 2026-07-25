package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	DispatcherRequestPending  = "pending"
	DispatcherRequestApproved = "approved"
	DispatcherRequestRejected = "rejected"
)

var (
	// ErrAlreadyManaged means the target driver already has a dispatcher.
	ErrAlreadyManaged = errors.New("store: driver already has a dispatcher")
	// ErrRequestAlreadyPending means this dispatcher already has an open
	// request to this driver.
	ErrRequestAlreadyPending = errors.New("store: a pending request to this driver already exists")
)

type DispatcherRequest struct {
	ID           int64
	DispatcherID int64
	DriverID     int64
	Status       string
	CreatedAt    time.Time
	RespondedAt  *time.Time
}

type DispatcherRequestStore struct {
	db *sql.DB
}

func NewDispatcherRequestStore(db *sql.DB) *DispatcherRequestStore {
	return &DispatcherRequestStore{db: db}
}

// Create records a new pending request from dispatcherID to driverID. Rejects
// if the driver is already managed, or a pending request from this dispatcher
// to this driver already exists.
func (s *DispatcherRequestStore) Create(ctx context.Context, dispatcherID, driverID int64) (DispatcherRequest, error) {
	var alreadyManaged bool
	if err := s.db.QueryRowContext(ctx, `SELECT dispatcher_id IS NOT NULL FROM drivers WHERE id = $1`, driverID).Scan(&alreadyManaged); err != nil {
		if err == sql.ErrNoRows {
			return DispatcherRequest{}, ErrNotFound
		}
		return DispatcherRequest{}, fmt.Errorf("check driver managed: %w", err)
	}
	if alreadyManaged {
		return DispatcherRequest{}, ErrAlreadyManaged
	}

	var pendingExists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM dispatcher_requests WHERE dispatcher_id = $1 AND driver_id = $2 AND status = $3)`,
		dispatcherID, driverID, DispatcherRequestPending).Scan(&pendingExists)
	if err != nil {
		return DispatcherRequest{}, fmt.Errorf("check pending request: %w", err)
	}
	if pendingExists {
		return DispatcherRequest{}, ErrRequestAlreadyPending
	}

	req := DispatcherRequest{DispatcherID: dispatcherID, DriverID: driverID, Status: DispatcherRequestPending}
	row := s.db.QueryRowContext(ctx, `
		INSERT INTO dispatcher_requests (dispatcher_id, driver_id) VALUES ($1, $2)
		RETURNING id, created_at`, dispatcherID, driverID)
	if err := row.Scan(&req.ID, &req.CreatedAt); err != nil {
		return DispatcherRequest{}, fmt.Errorf("insert dispatcher request: %w", err)
	}
	return req, nil
}

// ListPendingForDriver returns driverID's pending incoming requests.
func (s *DispatcherRequestStore) ListPendingForDriver(ctx context.Context, driverID int64) ([]DispatcherRequest, error) {
	return s.list(ctx, `WHERE driver_id = $1 AND status = $2`, driverID, DispatcherRequestPending)
}

// ListByDispatcher returns every request dispatcherID has sent, any status.
func (s *DispatcherRequestStore) ListByDispatcher(ctx context.Context, dispatcherID int64) ([]DispatcherRequest, error) {
	return s.list(ctx, `WHERE dispatcher_id = $1`, dispatcherID)
}

func (s *DispatcherRequestStore) list(ctx context.Context, where string, args ...any) ([]DispatcherRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, dispatcher_id, driver_id, status, created_at, responded_at
		FROM dispatcher_requests `+where+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("list dispatcher requests: %w", err)
	}
	defer rows.Close()

	requests := []DispatcherRequest{}
	for rows.Next() {
		var req DispatcherRequest
		if err := rows.Scan(&req.ID, &req.DispatcherID, &req.DriverID, &req.Status, &req.CreatedAt, &req.RespondedAt); err != nil {
			return nil, fmt.Errorf("scan dispatcher request: %w", err)
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

// Respond approves or rejects requestID on behalf of driverID (verifying it's
// actually theirs to respond to), and on approval links the driver to the
// dispatcher. Runs in a transaction since it touches two tables.
func (s *DispatcherRequestStore) Respond(ctx context.Context, requestID, driverID int64, approve bool) (DispatcherRequest, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return DispatcherRequest{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	var req DispatcherRequest
	row := tx.QueryRowContext(ctx, `
		SELECT id, dispatcher_id, driver_id, status, created_at, responded_at
		FROM dispatcher_requests WHERE id = $1 FOR UPDATE`, requestID)
	if err := row.Scan(&req.ID, &req.DispatcherID, &req.DriverID, &req.Status, &req.CreatedAt, &req.RespondedAt); err != nil {
		if err == sql.ErrNoRows {
			return DispatcherRequest{}, ErrNotFound
		}
		return DispatcherRequest{}, fmt.Errorf("select dispatcher request: %w", err)
	}
	if req.DriverID != driverID {
		return DispatcherRequest{}, ErrNotFound
	}
	if req.Status != DispatcherRequestPending {
		return DispatcherRequest{}, fmt.Errorf("store: request is not pending")
	}

	newStatus := DispatcherRequestRejected
	if approve {
		newStatus = DispatcherRequestApproved
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE dispatcher_requests SET status = $2, responded_at = now() WHERE id = $1`, requestID, newStatus); err != nil {
		return DispatcherRequest{}, fmt.Errorf("update dispatcher request: %w", err)
	}
	req.Status = newStatus

	if approve {
		if _, err := tx.ExecContext(ctx, `
			UPDATE drivers SET dispatcher_id = $2 WHERE id = $1`, driverID, req.DispatcherID); err != nil {
			return DispatcherRequest{}, fmt.Errorf("set dispatcher: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return DispatcherRequest{}, fmt.Errorf("commit tx: %w", err)
	}
	return req, nil
}
