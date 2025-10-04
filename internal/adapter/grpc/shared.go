package grpc

import (
	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	"github.com/eslsoft/vocnet/internal/entity"
)

const _maxPageSize = 10000

func convertPagination(p *commonv1.PaginationRequest) entity.Pagination {
	pageNo := p.GetPageNo()
	if pageNo <= 0 {
		pageNo = 1
	}
	pageSize := p.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > _maxPageSize {
		pageSize = _maxPageSize
	}

	return entity.Pagination{PageNo: pageNo, PageSize: pageSize}
}
