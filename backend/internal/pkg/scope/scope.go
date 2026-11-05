// Package scope defines the tenant/outlet boundary attached to every request.
package scope

import "github.com/google/uuid"

// Tenant is the authorization boundary for a request. AllOutletAccess is true
// for owners and admins, who may use every outlet in their tenant.
type Tenant struct {
	TenantID        uuid.UUID
	ActiveOutletID  *uuid.UUID
	OutletIDs       []uuid.UUID
	Role            string
	AllOutletAccess bool
	IsSuperAdmin    bool
}

// HasOutlet reports whether the scope grants access to an outlet.
func (s Tenant) HasOutlet(outletID uuid.UUID) bool {
	if s.IsSuperAdmin {
		return true
	}
	if s.AllOutletAccess {
		var found bool
		// OutletIDs contains every outlet for all-access roles.
		for _, id := range s.OutletIDs {
			if id == outletID {
				found = true
				break
			}
		}
		return found
	}
	for _, id := range s.OutletIDs {
		if id == outletID {
			return true
		}
	}
	return false
}

// RequireOutlet returns the active outlet or false when the request has no
// usable outlet context.
func (s Tenant) RequireOutlet() (uuid.UUID, bool) {
	if s.ActiveOutletID == nil {
		return uuid.Nil, false
	}
	return *s.ActiveOutletID, true
}
