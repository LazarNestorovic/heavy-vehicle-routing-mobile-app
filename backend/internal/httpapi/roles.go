package httpapi

import (
	"context"

	"heavy-vehicle-routing/backend/internal/store"
)

// tripAccessible reports whether callerID may view/act on trip: either the
// assigned driver, or - for a dispatcher-assigned trip - the dispatcher who
// assigned it (so they can watch its live position, events, etc).
func tripAccessible(trip store.Trip, callerID int64) bool {
	if trip.DriverID == callerID {
		return true
	}
	return trip.AssignedByID != nil && *trip.AssignedByID == callerID
}

// loadAccount loads the calling account (role + dispatcher_id). The JWT only
// carries driver_id/username (see internal/auth), so role-aware handlers need
// this extra lookup - a deliberate per-request DB round trip, consistent with
// this project's bias toward simplicity over caching role in the token.
func (s *Server) loadAccount(ctx context.Context, id int64) (store.Driver, error) {
	return s.Drivers.Get(ctx, id)
}

// vehicleAccessible reports whether account can use/view v: their own
// personal vehicle, their own fleet (as a dispatcher), or - for a managed
// driver - their dispatcher's fleet.
func vehicleAccessible(v store.Vehicle, account store.Driver) bool {
	if v.DriverID != nil && *v.DriverID == account.ID {
		return true
	}
	if v.DispatcherID != nil && account.Role == store.RoleDispatcher && *v.DispatcherID == account.ID {
		return true
	}
	if v.DispatcherID != nil && account.DispatcherID != nil && *v.DispatcherID == *account.DispatcherID {
		return true
	}
	return false
}

// vehicleMutable is vehicleAccessible's stricter counterpart, for actions
// that change the vehicle itself (edit dimensions, delete) rather than just
// viewing it or reporting its live fuel/service status. A managed driver can
// legitimately VIEW and update the STATUS of their dispatcher's fleet truck
// while driving it (see vehicleAccessible, used by handleGetVehicle/
// handleUpdateVehicleStatus) - but editing its physical profile or deleting
// it outright should stay with whoever actually owns it: themselves (a
// personal vehicle) or the dispatcher (their own fleet).
func vehicleMutable(v store.Vehicle, account store.Driver) bool {
	if v.DriverID != nil && *v.DriverID == account.ID {
		return true
	}
	if v.DispatcherID != nil && account.Role == store.RoleDispatcher && *v.DispatcherID == account.ID {
		return true
	}
	return false
}
