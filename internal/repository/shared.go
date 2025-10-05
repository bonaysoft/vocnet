package repository

// Pagination holds pagination parameters for listing entities.
type Pagination struct {
	PageNo   int32
	PageSize int32
}

func (p *Pagination) Offset() int32 { return (p.PageNo - 1) * p.PageSize }

type FilterOrder struct {
	Filter  string
	OrderBy string
}

func (fo *FilterOrder) GetFilter() string { return fo.Filter }

func (fo *FilterOrder) GetOrderBy() string { return fo.OrderBy }
