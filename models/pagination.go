package models

// PageParams carries optional server-side pagination parameters for list
// queries. A non-positive PageSize disables pagination, preserving the legacy
// behavior of returning every matching row.
type PageParams struct {
	Page     int
	PageSize int
}

// Valid returns true when pagination should be applied to a query.
func (p PageParams) Valid() bool {
	return p.PageSize > 0 && p.Page > 0
}

// Offset returns the number of rows to skip for the current page.
func (p PageParams) Offset() int {
	if !p.Valid() {
		return 0
	}
	return (p.Page - 1) * p.PageSize
}
