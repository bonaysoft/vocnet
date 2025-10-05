package grpc

import (
	"context"

	"connectrpc.com/connect"
	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	"github.com/eslsoft/vocnet/api/gen/dict/v1/dictv1connect"
	vocnetv1 "github.com/eslsoft/vocnet/api/gen/vocnet/v1"
	"github.com/eslsoft/vocnet/internal/adapter/mapping"
	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/repository"
	"github.com/eslsoft/vocnet/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UserWordServiceServer struct {
	dictv1connect.UnimplementedWordServiceHandler

	uc usecase.UserWordUsecase
}

func NewUserWordServiceServer(uc usecase.UserWordUsecase) *UserWordServiceServer {
	return &UserWordServiceServer{uc: uc}
}

func (s *UserWordServiceServer) CollectWord(ctx context.Context, req *connect.Request[vocnetv1.CollectWordRequest]) (*connect.Response[vocnetv1.UserWord], error) {
	if req.Msg == nil || req.Msg.Word == nil {
		return nil, status.Error(codes.InvalidArgument, "word payload required")
	}

	userID := int64(1000)
	entityWord := mapping.FromPbUserWord(req.Msg.Word)
	result, err := s.uc.CollectWord(ctx, userID, entityWord)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(mapping.ToPbUserWord(result)), nil
}

func (s *UserWordServiceServer) UpdateUserWordMastery(ctx context.Context, req *connect.Request[vocnetv1.UpdateUserWordMasteryRequest]) (*connect.Response[vocnetv1.UserWord], error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}

	msg := req.Msg
	userID := int64(1000)
	result, err := s.uc.UpdateMastery(ctx, userID, msg.GetWordId(), mapping.FromPbMastery(msg.GetMastery()), entity.ReviewTiming{}, msg.GetNotes())
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(mapping.ToPbUserWord(result)), nil
}

func (s *UserWordServiceServer) ListUserWords(ctx context.Context, req *connect.Request[vocnetv1.ListUserWordsRequest]) (*connect.Response[vocnetv1.ListUserWordsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	msg := req.Msg
	query := &repository.ListUserWordQuery{
		Pagination: convertPagination(msg.GetPagination()),
		FilterOrder: repository.FilterOrder{
			Filter:  msg.GetFilter(),
			OrderBy: msg.GetOrderBy(),
		},
		UserID: int64(1000),
	}
	items, total, err := s.uc.ListUserWords(ctx, query)
	if err != nil {
		return nil, err
	}

	resp := &vocnetv1.ListUserWordsResponse{
		Pagination: &commonv1.PaginationResponse{
			Total:  int32(total),
			PageNo: query.PageNo,
		},
	}
	for _, item := range items {
		resp.UserWords = append(resp.UserWords, mapping.ToPbUserWord(item))
	}

	return connect.NewResponse(resp), nil
}

func (s *UserWordServiceServer) DeleteUserWord(ctx context.Context, req *connect.Request[commonv1.IDRequest]) (*connect.Response[emptypb.Empty], error) {
	msg := req.Msg
	userID := int64(1000)
	if err := s.uc.DeleteUserWord(ctx, userID, msg.GetId()); err != nil {
		return nil, err
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}
