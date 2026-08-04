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

// PagedResponse wraps a paginated collection along with the total number of
// matching rows so that clients can render full pagination controls.
type PagedResponse struct {
	Total int64       `json:"total"`
	Data  interface{} `json:"data"`
}
