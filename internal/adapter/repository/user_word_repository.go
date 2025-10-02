package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eslsoft/vocnet/internal/entity"
	db "github.com/eslsoft/vocnet/internal/infrastructure/database/db"
	"github.com/eslsoft/vocnet/internal/infrastructure/database/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// UserWordRepository abstracts persistence for user words to keep usecases storage agnostic.
type UserWordRepository interface {
	Create(ctx context.Context, userWord *entity.UserWord) (*entity.UserWord, error)
	Update(ctx context.Context, userWord *entity.UserWord) (*entity.UserWord, error)
	GetByID(ctx context.Context, userID, id int64) (*entity.UserWord, error)
	FindByWord(ctx context.Context, userID int64, word string) (*entity.UserWord, error)
	List(ctx context.Context, filter entity.UserWordFilter) ([]*entity.UserWord, int64, error)
	Delete(ctx context.Context, userID, id int64) error
}

type userWordRepository struct {
	q *db.Queries
}

// NewUserWordRepository constructs a sqlc-backed repository.
func NewUserWordRepository(q *db.Queries) UserWordRepository {
	return &userWordRepository{q: q}
}

func (r *userWordRepository) Create(ctx context.Context, userWord *entity.UserWord) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	params := toCreateParams(userWord)
	row, err := r.q.CreateUserWord(ctx, params)
	if err != nil {
		return nil, translatePgError(err)
	}
	return mapUserWordRow(row), nil
}

func (r *userWordRepository) Update(ctx context.Context, userWord *entity.UserWord) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	params := toUpdateParams(userWord)
	row, err := r.q.UpdateUserWord(ctx, params)
	if err != nil {
		return nil, translatePgError(err)
	}
	return mapUserWordRow(row), nil
}

func (r *userWordRepository) GetByID(ctx context.Context, userID, id int64) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	row, err := r.q.GetUserWord(ctx, db.GetUserWordParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrUserWordNotFound
		}
		return nil, fmt.Errorf("get user word: %w", err)
	}
	return mapUserWordRow(row), nil
}

func (r *userWordRepository) FindByWord(ctx context.Context, userID int64, word string) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if word == "" {
		return nil, nil
	}
	row, err := r.q.FindUserWordByWord(ctx, db.FindUserWordByWordParams{UserID: userID, Lower: word})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user word: %w", err)
	}
	return mapUserWordRow(row), nil
}

func (r *userWordRepository) List(ctx context.Context, filter entity.UserWordFilter) ([]*entity.UserWord, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	rows, err := r.q.ListUserWords(ctx, db.ListUserWordsParams{
		UserID:  filter.UserID,
		Column2: filter.Keyword,
		Limit:   filter.Limit,
		Offset:  filter.Offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list user words: %w", err)
	}

	userWords := make([]*entity.UserWord, 0, len(rows))
	var total int64
	for _, row := range rows {
		userWords = append(userWords, mapUserWordRow(row))
		total = row.TotalCount
	}
	return userWords, total, nil
}

func (r *userWordRepository) Delete(ctx context.Context, userID, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	tag, err := r.q.DeleteUserWord(ctx, db.DeleteUserWordParams{ID: id, UserID: userID})
	if err != nil {
		return fmt.Errorf("delete user word: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrUserWordNotFound
	}
	return nil
}

func translatePgError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return entity.ErrDuplicateUserWord
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.ErrUserWordNotFound
	}
	return err
}

func toCreateParams(uw *entity.UserWord) db.CreateUserWordParams {
	return db.CreateUserWordParams{
		UserID:             uw.UserID,
		Word:               uw.Word,
		Language:           uw.Language,
		MasteryListen:      int16(uw.Mastery.Listen),
		MasteryRead:        int16(uw.Mastery.Read),
		MasterySpell:       int16(uw.Mastery.Spell),
		MasteryPronounce:   int16(uw.Mastery.Pronounce),
		MasteryUse:         int16(uw.Mastery.Use),
		MasteryOverall:     uw.Mastery.Overall,
		ReviewLastReviewAt: toPgTimestamp(uw.Review.LastReviewAt),
		ReviewNextReviewAt: toPgTimestamp(uw.Review.NextReviewAt),
		ReviewIntervalDays: uw.Review.IntervalDays,
		ReviewFailCount:    uw.Review.FailCount,
		QueryCount:         uw.QueryCount,
		Notes:              toPgText(uw.Notes),
		Sentences:          toUserSentences(uw.Sentences),
		Relations:          toUserWordRelations(uw.Relations),
		CreatedBy:          uw.CreatedBy,
		CreatedAt:          toPgTimestamp(ptrTime(uw.CreatedAt)),
		UpdatedAt:          toPgTimestamp(ptrTime(uw.UpdatedAt)),
	}
}

func toUpdateParams(uw *entity.UserWord) db.UpdateUserWordParams {
	return db.UpdateUserWordParams{
		ID:                 uw.ID,
		UserID:             uw.UserID,
		Word:               uw.Word,
		Language:           uw.Language,
		MasteryListen:      int16(uw.Mastery.Listen),
		MasteryRead:        int16(uw.Mastery.Read),
		MasterySpell:       int16(uw.Mastery.Spell),
		MasteryPronounce:   int16(uw.Mastery.Pronounce),
		MasteryUse:         int16(uw.Mastery.Use),
		MasteryOverall:     uw.Mastery.Overall,
		ReviewLastReviewAt: toPgTimestamp(uw.Review.LastReviewAt),
		ReviewNextReviewAt: toPgTimestamp(uw.Review.NextReviewAt),
		ReviewIntervalDays: uw.Review.IntervalDays,
		ReviewFailCount:    uw.Review.FailCount,
		QueryCount:         uw.QueryCount,
		Notes:              toPgText(uw.Notes),
		Sentences:          toUserSentences(uw.Sentences),
		Relations:          toUserWordRelations(uw.Relations),
		CreatedBy:          uw.CreatedBy,
		UpdatedAt:          toPgTimestamp(ptrTime(uw.UpdatedAt)),
	}
}

func mapUserWordRow(row interface{}) *entity.UserWord {
	switch v := row.(type) {
	case db.CreateUserWordRow:
		return mapFromRecord(v.ID, v.UserID, v.Word, v.Language, v.MasteryListen, v.MasteryRead, v.MasterySpell, v.MasteryPronounce, v.MasteryUse, v.MasteryOverall, v.ReviewLastReviewAt, v.ReviewNextReviewAt, v.ReviewIntervalDays, v.ReviewFailCount, v.QueryCount, v.Notes, v.Sentences, v.Relations, v.CreatedBy, v.CreatedAt, v.UpdatedAt)
	case db.UpdateUserWordRow:
		return mapFromRecord(v.ID, v.UserID, v.Word, v.Language, v.MasteryListen, v.MasteryRead, v.MasterySpell, v.MasteryPronounce, v.MasteryUse, v.MasteryOverall, v.ReviewLastReviewAt, v.ReviewNextReviewAt, v.ReviewIntervalDays, v.ReviewFailCount, v.QueryCount, v.Notes, v.Sentences, v.Relations, v.CreatedBy, v.CreatedAt, v.UpdatedAt)
	case db.GetUserWordRow:
		return mapFromRecord(v.ID, v.UserID, v.Word, v.Language, v.MasteryListen, v.MasteryRead, v.MasterySpell, v.MasteryPronounce, v.MasteryUse, v.MasteryOverall, v.ReviewLastReviewAt, v.ReviewNextReviewAt, v.ReviewIntervalDays, v.ReviewFailCount, v.QueryCount, v.Notes, v.Sentences, v.Relations, v.CreatedBy, v.CreatedAt, v.UpdatedAt)
	case db.FindUserWordByWordRow:
		return mapFromRecord(v.ID, v.UserID, v.Word, v.Language, v.MasteryListen, v.MasteryRead, v.MasterySpell, v.MasteryPronounce, v.MasteryUse, v.MasteryOverall, v.ReviewLastReviewAt, v.ReviewNextReviewAt, v.ReviewIntervalDays, v.ReviewFailCount, v.QueryCount, v.Notes, v.Sentences, v.Relations, v.CreatedBy, v.CreatedAt, v.UpdatedAt)
	case db.ListUserWordsRow:
		return mapFromRecord(v.ID, v.UserID, v.Word, v.Language, v.MasteryListen, v.MasteryRead, v.MasterySpell, v.MasteryPronounce, v.MasteryUse, v.MasteryOverall, v.ReviewLastReviewAt, v.ReviewNextReviewAt, v.ReviewIntervalDays, v.ReviewFailCount, v.QueryCount, v.Notes, v.Sentences, v.Relations, v.CreatedBy, v.CreatedAt, v.UpdatedAt)
	default:
		return nil
	}
}

func mapFromRecord(
	id, userID int64,
	word, language string,
	masteryListen, masteryRead, masterySpell, masteryPronounce, masteryUse int16,
	masteryOverall int32,
	reviewLast, reviewNext pgtype.Timestamptz,
	reviewInterval, reviewFail int32,
	queryCount int64,
	notes pgtype.Text,
	sentences types.UserSentences,
	relations types.UserWordRelations,
	createdBy string,
	createdAt, updatedAt pgtype.Timestamptz,
) *entity.UserWord {
	uw := &entity.UserWord{
		ID:         id,
		UserID:     userID,
		Word:       word,
		Language:   language,
		Mastery:    entity.MasteryBreakdown{Listen: int32(masteryListen), Read: int32(masteryRead), Spell: int32(masterySpell), Pronounce: int32(masteryPronounce), Use: int32(masteryUse), Overall: masteryOverall},
		Review:     entity.ReviewTiming{IntervalDays: reviewInterval, FailCount: reviewFail},
		QueryCount: queryCount,
		CreatedBy:  createdBy,
	}
	if reviewLast.Valid {
		reviewTime := reviewLast.Time
		uw.Review.LastReviewAt = &reviewTime
	}
	if reviewNext.Valid {
		reviewTime := reviewNext.Time
		uw.Review.NextReviewAt = &reviewTime
	}
	if notes.Valid {
		uw.Notes = notes.String
	}
	uw.Sentences = fromUserSentences(sentences)
	uw.Relations = fromUserWordRelations(relations)
	if createdAt.Valid {
		uw.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		uw.UpdatedAt = updatedAt.Time
	}
	return uw
}

func toPgTimestamp(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func toPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

func toUserSentences(sentences []entity.Sentence) types.UserSentences {
	if len(sentences) == 0 {
		return nil
	}
	res := make(types.UserSentences, 0, len(sentences))
	for _, s := range sentences {
		res = append(res, types.UserSentence{Text: s.Text, Source: s.Source})
	}
	return res
}

func toUserWordRelations(relations []entity.WordRelation) types.UserWordRelations {
	if len(relations) == 0 {
		return nil
	}
	res := make(types.UserWordRelations, 0, len(relations))
	for _, r := range relations {
		res = append(res, types.UserWordRelation{
			Word:         r.Word,
			RelationType: r.RelationType,
			Note:         r.Note,
			CreatedBy:    r.CreatedBy,
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
		})
	}
	return res
}

func fromUserSentences(sentences types.UserSentences) []entity.Sentence {
	if len(sentences) == 0 {
		return []entity.Sentence{}
	}
	res := make([]entity.Sentence, 0, len(sentences))
	for _, s := range sentences {
		res = append(res, entity.Sentence{Text: s.Text, Source: s.Source})
	}
	return res
}

func fromUserWordRelations(relations types.UserWordRelations) []entity.WordRelation {
	if len(relations) == 0 {
		return []entity.WordRelation{}
	}
	res := make([]entity.WordRelation, 0, len(relations))
	for _, r := range relations {
		res = append(res, entity.WordRelation{
			Word:         r.Word,
			RelationType: r.RelationType,
			Note:         r.Note,
			CreatedBy:    r.CreatedBy,
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
		})
	}
	return res
}

func ptrTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
