package grpc

import (
	"context"
	"strings"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	"github.com/eslsoft/vocnet/api/gen/dict/v1/dictv1connect"
	vocnetv1 "github.com/eslsoft/vocnet/api/gen/vocnet/v1"
	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	entityWord := protoToEntityUserWord(req.Msg.Word)
	result, err := s.uc.CollectWord(ctx, userID, entityWord)
	if err != nil {
		return nil, toStatus(err)
	}

	return connect.NewResponse(entityToProtoUserWord(result)), nil
}

func (s *UserWordServiceServer) UpdateUserWordMastery(ctx context.Context, req *connect.Request[vocnetv1.UpdateUserWordMasteryRequest]) (*connect.Response[vocnetv1.UserWord], error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}

	msg := req.Msg
	userID := int64(1000)
	result, err := s.uc.UpdateMastery(ctx, userID, msg.GetWordId(), protoToEntityMastery(msg.GetMastery()), entity.ReviewTiming{}, msg.GetNotes())
	if err != nil {
		return nil, toStatus(err)
	}

	return connect.NewResponse(entityToProtoUserWord(result)), nil
}

func (s *UserWordServiceServer) ListUserWords(ctx context.Context, req *connect.Request[vocnetv1.ListUserWordsRequest]) (*connect.Response[vocnetv1.ListUserWordsResponse], error) {
	msg := req.Msg
	userID := int64(1000)
	pagination := msg.GetPagination()
	filter := entity.UserWordFilter{
		UserID:  userID,
		Keyword: msg.GetKeyword(),
	}
	if pagination != nil {
		filter.Limit = pagination.GetLimit()
		filter.Offset = pagination.GetOffset()
	}

	items, total, err := s.uc.ListUserWords(ctx, filter)
	if err != nil {
		return nil, toStatus(err)
	}

	resp := &vocnetv1.ListUserWordsResponse{
		Pagination: &commonv1.PaginationResponse{
			Total:  int32(total),
			PageNo: computePageNo(filter),
		},
	}
	for _, item := range items {
		resp.UserWords = append(resp.UserWords, entityToProtoUserWord(item))
	}

	return connect.NewResponse(resp), nil
}

func (s *UserWordServiceServer) DeleteUserWord(ctx context.Context, req *connect.Request[commonv1.IDRequest]) (*connect.Response[emptypb.Empty], error) {
	msg := req.Msg
	userID := int64(1000)
	if err := s.uc.DeleteUserWord(ctx, userID, msg.GetId()); err != nil {
		return nil, toStatus(err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

func protoToEntityUserWord(in *vocnetv1.UserWord) *entity.UserWord {
	if in == nil {
		return nil
	}
	uw := &entity.UserWord{
		ID:         in.GetId(),
		Word:       strings.TrimSpace(in.GetWord()),
		Language:   "en",
		Mastery:    protoToEntityMastery(in.GetMastery()),
		Review:     protoToEntityReview(in.GetReviewTiming()),
		QueryCount: in.GetQueryCount(),
		Notes:      in.GetNotes(),
		CreatedBy:  in.GetCreatedBy(),
	}
	if ts := in.GetCreatedAt(); ts != nil {
		uw.CreatedAt = ts.AsTime()
	}
	if ts := in.GetUpdatedAt(); ts != nil {
		uw.UpdatedAt = ts.AsTime()
	}
	for _, s := range in.GetSentences() {
		uw.Sentences = append(uw.Sentences, entity.Sentence{Text: s.GetText(), Source: int32(s.GetSource())})
	}
	for _, rel := range in.GetRelations() {
		uw.Relations = append(uw.Relations, entity.WordRelation{
			Word:         rel.GetWord(),
			RelationType: int32(rel.GetRelationType()),
			Note:         rel.GetNote(),
			CreatedBy:    rel.GetCreatedBy(),
			CreatedAt:    toTime(rel.GetCreatedAt()),
			UpdatedAt:    toTime(rel.GetUpdatedAt()),
		})
	}
	return uw
}

func entityToProtoUserWord(in *entity.UserWord) *vocnetv1.UserWord {
	if in == nil {
		return nil
	}
	out := &vocnetv1.UserWord{
		Id:           in.ID,
		Word:         in.Word,
		Mastery:      entityToProtoMastery(in.Mastery),
		ReviewTiming: entityToProtoReview(in.Review),
		QueryCount:   in.QueryCount,
		Notes:        in.Notes,
		CreatedBy:    in.CreatedBy,
		CreatedAt:    timestampOrNil(in.CreatedAt),
		UpdatedAt:    timestampOrNil(in.UpdatedAt),
	}
	for _, s := range in.Sentences {
		out.Sentences = append(out.Sentences, &vocnetv1.Sentence{Text: s.Text, Source: vocnetv1.SourceType(s.Source)})
	}
	for _, rel := range in.Relations {
		out.Relations = append(out.Relations, &vocnetv1.WordRelation{
			Word:         rel.Word,
			RelationType: vocnetv1.RelationType(rel.RelationType),
			Note:         rel.Note,
			CreatedBy:    rel.CreatedBy,
			CreatedAt:    timestampOrNil(rel.CreatedAt),
			UpdatedAt:    timestampOrNil(rel.UpdatedAt),
		})
	}
	return out
}

func protoToEntityMastery(in *vocnetv1.MasteryBreakdown) entity.MasteryBreakdown {
	if in == nil {
		return entity.MasteryBreakdown{}
	}
	return entity.MasteryBreakdown{
		Listen:    in.GetListen(),
		Read:      in.GetRead(),
		Spell:     in.GetSpell(),
		Pronounce: in.GetPronounce(),
		Use:       in.GetUse(),
		Overall:   in.GetOverall(),
	}
}

func entityToProtoMastery(in entity.MasteryBreakdown) *vocnetv1.MasteryBreakdown {
	return &vocnetv1.MasteryBreakdown{
		Listen:    in.Listen,
		Read:      in.Read,
		Spell:     in.Spell,
		Pronounce: in.Pronounce,
		Use:       in.Use,
		Overall:   in.Overall,
	}
}

func protoToEntityReview(in *vocnetv1.ReviewTiming) entity.ReviewTiming {
	if in == nil {
		return entity.ReviewTiming{}
	}
	return entity.ReviewTiming{
		LastReviewAt: toTimePtr(in.GetLastReviewAt()),
		NextReviewAt: toTimePtr(in.GetNextReviewAt()),
		IntervalDays: in.GetIntervalDays(),
		FailCount:    in.GetFailCount(),
	}
}

func entityToProtoReview(in entity.ReviewTiming) *vocnetv1.ReviewTiming {
	return &vocnetv1.ReviewTiming{
		LastReviewAt: fromTimePtr(in.LastReviewAt),
		NextReviewAt: fromTimePtr(in.NextReviewAt),
		IntervalDays: in.IntervalDays,
		FailCount:    in.FailCount,
	}
}

func toTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func toTimePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func fromTimePtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func timestampOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func computePageNo(filter entity.UserWordFilter) int32 {
	if filter.Limit <= 0 {
		return 1
	}
	return filter.Offset/filter.Limit + 1
}

func toStatus(err error) error {
	switch err {
	case nil:
		return nil
	case entity.ErrInvalidUserWordText, entity.ErrInvalidUserID:
		return status.Error(codes.InvalidArgument, err.Error())
	case entity.ErrUserWordNotFound:
		return status.Error(codes.NotFound, err.Error())
	case entity.ErrDuplicateUserWord:
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
