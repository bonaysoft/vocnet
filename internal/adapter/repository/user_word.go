package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eslsoft/vocnet/internal/entity"
	db "github.com/eslsoft/vocnet/internal/infrastructure/database/db"
	"github.com/eslsoft/vocnet/internal/repository"
	"github.com/eslsoft/vocnet/pkg/filterexpr"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type userWordRepository struct {
	q *db.Queries
}

// NewUserWordRepository constructs a sqlc-backed repository.
func NewUserWordRepository(q *db.Queries) repository.UserWordRepository {
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
	return mapDBUserWord(row), nil
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
	return mapDBUserWord(row), nil
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
	return mapDBUserWord(row), nil
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
	return mapDBUserWord(row), nil
}

func (r *userWordRepository) List(ctx context.Context, query *repository.ListUserWordQuery) ([]*entity.UserWord, int64, error) {
	var p db.ListUserWordsParams
	if err := filterexpr.Bind(query, &p, listUserWordsSchema); err != nil {
		return nil, 0, err
	}

	p.UserID = query.UserID
	p.Offset = query.Offset()
	p.Limit = query.PageSize
	fmt.Println(p)
	rows, err := r.q.ListUserWords(ctx, p)
	if err != nil {
		return nil, 0, fmt.Errorf("list user words: %w", err)
	}

	total, err := r.q.CountUserWords(ctx, db.CountUserWordsParams{
		UserID:  p.UserID,
		Keyword: p.Keyword,
		Words:   p.Words,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list user words: %w", err)
	}

	userWords := make([]*entity.UserWord, 0, len(rows))
	for _, row := range rows {
		userWord := mapDBUserWord(row.UserWord)
		userWord.WordContent = mapDBWord(row.Word)
		userWords = append(userWords, userWord)
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
		Language:           uw.Language.Code(),
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
		Sentences:          uw.Sentences,
		Relations:          uw.Relations,
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
		Language:           uw.Language.Code(),
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
		Sentences:          uw.Sentences,
		Relations:          uw.Relations,
		CreatedBy:          uw.CreatedBy,
		UpdatedAt:          toPgTimestamp(ptrTime(uw.UpdatedAt)),
	}
}

func mapDBUserWord(row db.UserWord) *entity.UserWord {
	uw := &entity.UserWord{
		ID:       row.ID,
		UserID:   row.UserID,
		Word:     row.Word,
		Language: entity.ParseLanguage(row.Language),
		Mastery: entity.MasteryBreakdown{
			Listen:    int32(row.MasteryListen),
			Read:      int32(row.MasteryRead),
			Spell:     int32(row.MasterySpell),
			Pronounce: int32(row.MasteryPronounce),
			Use:       int32(row.MasteryUse),
			Overall:   row.MasteryOverall,
		},
		Review:     entity.ReviewTiming{IntervalDays: row.ReviewIntervalDays, FailCount: row.ReviewFailCount},
		QueryCount: row.QueryCount,
		Sentences:  row.Sentences,
		Relations:  row.Relations,
		CreatedBy:  row.CreatedBy,
	}
	if row.ReviewLastReviewAt.Valid {
		reviewTime := row.ReviewLastReviewAt.Time
		uw.Review.LastReviewAt = &reviewTime
	}
	if row.ReviewNextReviewAt.Valid {
		reviewTime := row.ReviewNextReviewAt.Time
		uw.Review.NextReviewAt = &reviewTime
	}
	if row.Notes.Valid {
		uw.Notes = row.Notes.String
	}
	if row.CreatedAt.Valid {
		uw.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		uw.UpdatedAt = row.UpdatedAt.Time
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

func ptrTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
