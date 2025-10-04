package entity

// Pagination holds pagination parameters for listing entities.
type Pagination struct {
	PageNo   int32
	PageSize int32
}

func (p *Pagination) Offset() int32 { return (p.PageNo - 1) * p.PageSize }
