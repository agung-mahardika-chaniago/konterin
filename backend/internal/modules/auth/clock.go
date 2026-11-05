package auth

import "time"

// nowUTC is a seam for cookie max-age calculations.
func nowUTC() time.Time { return time.Now().UTC() }
