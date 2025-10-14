package grpc

import (
	"context"

	"connectrpc.com/connect"
	"github.com/eslsoft/vocnet/internal/adapter/mapping"
	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/repository"
	"github.com/eslsoft/vocnet/internal/usecase"
	commonv1 "github.com/eslsoft/vocnet/pkg/api/common/v1"
	learningv1 "github.com/eslsoft/vocnet/pkg/api/learning/v1"
	"github.com/eslsoft/vocnet/pkg/api/learning/v1/learningv1connect"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ learningv1connect.LearningServiceHandler = (*LearningServiceServer)(nil)

type LearningServiceServer struct {
	learningv1connect.UnimplementedLearningServiceHandler

	uc usecase.UserWordUsecase
}

func NewUserWordServiceServer(uc usecase.UserWordUsecase) *LearningServiceServer {
	return &LearningServiceServer{uc: uc}
}

func (s *LearningServiceServer) CollectWord(ctx context.Context, req *connect.Request[learningv1.CollectWordRequest]) (*connect.Response[learningv1.LearnedWord], error) {
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

func (s *LearningServiceServer) UncollectWord(ctx context.Context, req *connect.Request[commonv1.IDRequest]) (*connect.Response[emptypb.Empty], error) {
	msg := req.Msg
	userID := int64(1000)
	if err := s.uc.DeleteUserWord(ctx, userID, msg.GetId()); err != nil {
		return nil, err
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func (s *LearningServiceServer) ListLearnedWords(ctx context.Context, req *connect.Request[learningv1.ListLearnedWordsRequest]) (*connect.Response[learningv1.ListLearnedWordsResponse], error) {
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

	total32, err := safeInt32("total user words", total)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &learningv1.ListLearnedWordsResponse{
		Pagination: &commonv1.PaginationResponse{
			Total:  total32,
			PageNo: query.PageNo,
		},
	}
	for _, item := range items {
		resp.Words = append(resp.Words, mapping.ToPbUserWord(&item))
	}

	return connect.NewResponse(resp), nil
}

func (s *LearningServiceServer) UpdateMastery(ctx context.Context, req *connect.Request[learningv1.UpdateMasteryRequest]) (*connect.Response[learningv1.LearnedWord], error) {
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
