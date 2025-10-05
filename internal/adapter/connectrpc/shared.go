package grpc

import (
	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	"github.com/eslsoft/vocnet/internal/repository"
)

const _maxPageSize = 10000

func convertPagination(p *commonv1.PaginationRequest) repository.Pagination {
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

	return repository.Pagination{PageNo: pageNo, PageSize: pageSize}
}
