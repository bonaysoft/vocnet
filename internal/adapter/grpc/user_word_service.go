package grpc

import (
	"context"
	"strings"
	"time"

	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	vocnetv1 "github.com/eslsoft/vocnet/api/gen/vocnet/v1"
	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserWordServiceServer struct {
	vocnetv1.UnimplementedUserWordServiceServer
	uc            usecase.UserWordUsecase
	defaultUserID int64
}

func NewUserWordServiceServer(uc usecase.UserWordUsecase, defaultUserID int64) *UserWordServiceServer {
	if defaultUserID <= 0 {
		defaultUserID = 1
	}
	return &UserWordServiceServer{uc: uc, defaultUserID: defaultUserID}
}

func (s *UserWordServiceServer) CollectWord(ctx context.Context, req *vocnetv1.CollectWordRequest) (*vocnetv1.UserWord, error) {
	if req == nil || req.Word == nil {
		return nil, status.Error(codes.InvalidArgument, "word payload required")
	}

	entityWord := protoToEntityUserWord(req.Word)
	result, err := s.uc.CollectWord(ctx, s.defaultUserID, entityWord)
	if err != nil {
		return nil, toStatus(err)
	}

	return entityToProtoUserWord(result), nil
}

func (s *UserWordServiceServer) UpdateUserWordMastery(ctx context.Context, req *vocnetv1.UpdateUserWordMasteryRequest) (*vocnetv1.UserWord, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}

	result, err := s.uc.UpdateMastery(ctx, s.defaultUserID, req.GetWordId(), protoToEntityMastery(req.GetMastery()), entity.ReviewTiming{}, req.GetNotes())
	if err != nil {
		return nil, toStatus(err)
	}

	return entityToProtoUserWord(result), nil
}

func (s *UserWordServiceServer) ListUserWords(ctx context.Context, req *vocnetv1.ListUserWordsRequest) (*vocnetv1.ListUserWordsResponse, error) {
	if req == nil {
		req = &vocnetv1.ListUserWordsRequest{}
	}

	pagination := req.GetPagination()
	filter := entity.UserWordFilter{
		UserID:  s.defaultUserID,
		Keyword: req.GetKeyword(),
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

	return resp, nil
}

func (s *UserWordServiceServer) DeleteUserWord(ctx context.Context, req *commonv1.IDRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	if err := s.uc.DeleteUserWord(ctx, s.defaultUserID, req.GetId()); err != nil {
		return nil, toStatus(err)
	}
	return &emptypb.Empty{}, nil
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
