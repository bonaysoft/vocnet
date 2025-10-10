package grpc

import (
	"context"

	"connectrpc.com/connect"
	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	dictv1 "github.com/eslsoft/vocnet/api/gen/dict/v1"
	"github.com/eslsoft/vocnet/api/gen/dict/v1/dictv1connect"
	"github.com/eslsoft/vocnet/internal/adapter/mapping"
	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/repository"
	"github.com/eslsoft/vocnet/internal/usecase"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ dictv1connect.WordServiceHandler = (*WordServiceServer)(nil)

type WordServiceServer struct {
	dictv1connect.UnimplementedWordServiceHandler
	uc usecase.WordUsecase
}

func NewWordServiceServer(uc usecase.WordUsecase) *WordServiceServer {
	return &WordServiceServer{uc: uc}
}

func (s *WordServiceServer) CreateWord(ctx context.Context, req *connect.Request[dictv1.CreateWordRequest]) (*connect.Response[dictv1.Word], error) {
	if req.Msg == nil || req.Msg.Word == nil {
		return nil, status.Error(codes.InvalidArgument, "word payload required")
	}

	result, err := s.uc.Create(ctx, mapping.FromPbWord(req.Msg.Word))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(mapping.ToPbWord(result)), nil
}

func (s *WordServiceServer) UpdateWord(ctx context.Context, req *connect.Request[dictv1.Word]) (*connect.Response[dictv1.Word], error) {
	if req.Msg == nil {
		return nil, status.Error(codes.InvalidArgument, "word payload required")
	}

	result, err := s.uc.Update(ctx, mapping.FromPbWord(req.Msg))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(mapping.ToPbWord(result)), nil
}

func (s *WordServiceServer) GetWord(ctx context.Context, req *connect.Request[commonv1.IDRequest]) (*connect.Response[dictv1.Word], error) {
	if req.Msg == nil {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}

	result, err := s.uc.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(mapping.ToPbWord(result)), nil
}

func (s *WordServiceServer) ListWords(ctx context.Context, req *connect.Request[dictv1.ListWordsRequest]) (*connect.Response[dictv1.ListWordsResponse], error) {
	if req.Msg == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	msg := req.Msg
	query := &repository.ListWordQuery{
		Pagination: convertPagination(msg.GetPagination()),
		FilterOrder: repository.FilterOrder{
			Filter:  msg.GetFilter(),
			OrderBy: msg.GetOrderBy(),
		},
	}
	items, total, err := s.uc.List(ctx, query)
	if err != nil {
		return nil, err
	}

	total32, err := safeInt32("total words", total)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return connect.NewResponse(&dictv1.ListWordsResponse{
		Words: lo.Map(items, func(item *entity.Word, _ int) *dictv1.Word {
			return mapping.ToPbWord(item)
		}),
		Pagination: &commonv1.PaginationResponse{
			Total:  total32,
			PageNo: query.PageNo,
		},
	}), nil
}

func (s *WordServiceServer) DeleteWord(ctx context.Context, req *connect.Request[commonv1.IDRequest]) (*connect.Response[emptypb.Empty], error) {
	if req.Msg == nil {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	if err := s.uc.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// LookupWord looks up a word by text and language.
func (s *WordServiceServer) LookupWord(ctx context.Context, req *connect.Request[dictv1.LookupWordRequest]) (*connect.Response[dictv1.Word], error) {
	if req.Msg == nil || req.Msg.Word == "" {
		return nil, status.Error(codes.InvalidArgument, "text required")
	}

	v, err := s.uc.Lookup(ctx, req.Msg.Word, mapping.FromPbLanguage(req.Msg.Language))
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(mapping.ToPbWord(v)), nil
}
