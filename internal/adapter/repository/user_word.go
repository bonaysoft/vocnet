package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/eslsoft/vocnet/internal/entity"
	entdb "github.com/eslsoft/vocnet/internal/infrastructure/database/ent"
	entuserword "github.com/eslsoft/vocnet/internal/infrastructure/database/ent/userword"
	"github.com/eslsoft/vocnet/internal/repository"
	"github.com/eslsoft/vocnet/pkg/filterexpr"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/samber/lo"
)

type userWordRepository struct {
	client *entdb.Client
}

func int32ToInt16(value int32, field string) (int16, error) {
	if value > math.MaxInt16 || value < math.MinInt16 {
		return 0, fmt.Errorf("%s out of int16 range: %d", field, value)
	}
	return int16(value), nil
}

// NewUserWordRepository constructs an ent-backed repository.
func NewUserWordRepository(client *entdb.Client) repository.UserWordRepository {
	return &userWordRepository{client: client}
}

type listUserWordsParams struct {
	Keyword       string
	Words         []string
	PrimaryKey    string
	PrimaryDesc   bool
	SecondaryKey  string
	SecondaryDesc bool
}

func (r *userWordRepository) Create(ctx context.Context, userWord *entity.UserWord) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	listen, err := int32ToInt16(userWord.Mastery.Listen, "mastery.listen")
	if err != nil {
		return nil, err
	}
	read, err := int32ToInt16(userWord.Mastery.Read, "mastery.read")
	if err != nil {
		return nil, err
	}
	spell, err := int32ToInt16(userWord.Mastery.Spell, "mastery.spell")
	if err != nil {
		return nil, err
	}
	pronounce, err := int32ToInt16(userWord.Mastery.Pronounce, "mastery.pronounce")
	if err != nil {
		return nil, err
	}

	builder := r.client.UserWord.Create().
		SetUserID(userWord.UserID).
		SetWord(userWord.Word).
		SetNormalized(entity.NormalizeWordToken(userWord.Word)).
		SetLanguage(entity.NormalizeLanguage(userWord.Language).Code()).
		SetMasteryListen(listen).
		SetMasteryRead(read).
		SetMasterySpell(spell).
		SetMasteryPronounce(pronounce).
		SetMasteryOverall(userWord.Mastery.Overall).
		SetReviewIntervalDays(userWord.Review.IntervalDays).
		SetReviewFailCount(userWord.Review.FailCount).
		SetQueryCount(userWord.QueryCount).
		SetSentences(userWord.Sentences).
		SetRelations(userWord.Relations).
		SetCreatedBy(userWord.CreatedBy).
		SetCreatedAt(userWord.CreatedAt).
		SetUpdatedAt(userWord.UpdatedAt)

	if !userWord.Review.LastReviewAt.IsZero() {
		builder.SetReviewLastReviewAt(userWord.Review.LastReviewAt)
	}
	if !userWord.Review.NextReviewAt.IsZero() {
		builder.SetReviewNextReviewAt(userWord.Review.NextReviewAt)
	}
	if userWord.Notes != "" {
		builder.SetNotes(userWord.Notes)
	}

	rec, err := builder.Save(ctx)
	if err != nil {
		return nil, translateUserWordError(err)
	}
	return mapEntUserWord(rec), nil
}

func (r *userWordRepository) Update(ctx context.Context, userWord *entity.UserWord) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	listen, err := int32ToInt16(userWord.Mastery.Listen, "mastery.listen")
	if err != nil {
		return nil, err
	}
	read, err := int32ToInt16(userWord.Mastery.Read, "mastery.read")
	if err != nil {
		return nil, err
	}
	spell, err := int32ToInt16(userWord.Mastery.Spell, "mastery.spell")
	if err != nil {
		return nil, err
	}
	pronounce, err := int32ToInt16(userWord.Mastery.Pronounce, "mastery.pronounce")
	if err != nil {
		return nil, err
	}

	mutation := r.client.UserWord.UpdateOneID(int(userWord.ID)).
		Where(entuserword.UserIDEQ(userWord.UserID)).
		SetWord(userWord.Word).
		SetNormalized(entity.NormalizeWordToken(userWord.Word)).
		SetLanguage(entity.NormalizeLanguage(userWord.Language).Code()).
		SetMasteryListen(listen).
		SetMasteryRead(read).
		SetMasterySpell(spell).
		SetMasteryPronounce(pronounce).
		SetMasteryOverall(userWord.Mastery.Overall).
		SetReviewIntervalDays(userWord.Review.IntervalDays).
		SetReviewFailCount(userWord.Review.FailCount).
		SetQueryCount(userWord.QueryCount).
		SetSentences(userWord.Sentences).
		SetRelations(userWord.Relations).
		SetCreatedBy(userWord.CreatedBy).
		SetUpdatedAt(userWord.UpdatedAt)

	if !userWord.Review.LastReviewAt.IsZero() {
		mutation.SetReviewLastReviewAt(userWord.Review.LastReviewAt)
	} else {
		mutation.ClearReviewLastReviewAt()
	}
	if !userWord.Review.NextReviewAt.IsZero() {
		mutation.SetReviewNextReviewAt(userWord.Review.NextReviewAt)
	} else {
		mutation.ClearReviewNextReviewAt()
	}

	if userWord.Notes != "" {
		mutation.SetNotes(userWord.Notes)
	} else {
		mutation.ClearNotes()
	}

	rec, err := mutation.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, entity.ErrUserWordNotFound
		}
		return nil, translateUserWordError(err)
	}

	return mapEntUserWord(rec), nil
}

func (r *userWordRepository) GetByID(ctx context.Context, userID, id int64) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	rec, err := r.client.UserWord.Query().
		Where(
			entuserword.IDEQ(int(id)),
			entuserword.UserIDEQ(userID),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, entity.ErrUserWordNotFound
		}
		return nil, fmt.Errorf("get user word: %w", err)
	}
	return mapEntUserWord(rec), nil
}

func (r *userWordRepository) FindByWord(ctx context.Context, userID int64, word string) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if word == "" {
		return nil, nil
	}

	rec, err := r.client.UserWord.Query().
		Where(
			entuserword.UserIDEQ(userID),
			entuserword.WordEQ(word),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user word: %w", err)
	}
	return mapEntUserWord(rec), nil
}

func (r *userWordRepository) List(ctx context.Context, query *repository.ListUserWordQuery) ([]entity.UserWord, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}

	var params listUserWordsParams
	if err := filterexpr.Bind(query, &params, listUserWordsSchema); err != nil {
		return nil, 0, err
	}

	qbuilder := r.client.UserWord.Query().
		Where(entuserword.UserIDEQ(query.UserID))

	applyUserWordFilters(qbuilder, params)

	total, err := qbuilder.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count user words: %w", err)
	}

	applyUserWordOrdering(qbuilder, params)

	offset := query.Offset()
	if offset > 0 {
		qbuilder.Offset(int(offset))
	}
	if query.PageSize > 0 {
		qbuilder.Limit(int(query.PageSize))
	}

	rows, err := qbuilder.All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list user words: %w", err)
	}

	results := make([]entity.UserWord, 0, len(rows))
	for _, row := range rows {
		if mapped := mapEntUserWord(row); mapped != nil {
			results = append(results, *mapped)
		}
	}

	return results, int64(total), nil
}

func (r *userWordRepository) Delete(ctx context.Context, userID, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	affected, err := r.client.UserWord.Delete().
		Where(
			entuserword.IDEQ(int(id)),
			entuserword.UserIDEQ(userID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete user word: %w", err)
	}
	if affected == 0 {
		return entity.ErrUserWordNotFound
	}
	return nil
}

func applyUserWordFilters(q *entdb.UserWordQuery, params listUserWordsParams) {
	if params.Keyword != "" {
		q.Where(entuserword.WordContainsFold(params.Keyword))
	}
	if words := uniqueFolded(params.Words); len(words) > 0 {
		q.Where(entuserword.NormalizedIn(lo.Map(words, func(word string, _ int) string { return strings.ToLower(word) })...))
	}
}

func applyUserWordOrdering(q *entdb.UserWordQuery, params listUserWordsParams) {
	for _, term := range []struct {
		key  string
		desc bool
	}{
		{key: params.PrimaryKey, desc: params.PrimaryDesc},
		{key: params.SecondaryKey, desc: params.SecondaryDesc},
	} {
		if term.key == "" {
			continue
		}
		switch term.key {
		case "created_at":
			if term.desc {
				q.Order(entuserword.ByCreatedAt(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entuserword.ByCreatedAt(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "updated_at":
			if term.desc {
				q.Order(entuserword.ByUpdatedAt(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entuserword.ByUpdatedAt(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "word":
			if term.desc {
				q.Order(entuserword.ByWord(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entuserword.ByWord(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "mastery_overall":
			if term.desc {
				q.Order(entuserword.ByMasteryOverall(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entuserword.ByMasteryOverall(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "id":
			if term.desc {
				q.Order(entuserword.ByID(sql.OrderDesc()))
			} else {
				q.Order(entuserword.ByID())
			}
		}
	}

	q.Order(entuserword.ByID())
}

func mapEntUserWord(rec *entdb.UserWord) *entity.UserWord {
	if rec == nil {
		return nil
	}

	userWord := &entity.UserWord{
		ID:       int64(rec.ID),
		UserID:   rec.UserID,
		Word:     rec.Word,
		Language: entity.ParseLanguage(rec.Language),
		Mastery: entity.MasteryBreakdown{
			Listen:    int32(rec.MasteryListen),
			Read:      int32(rec.MasteryRead),
			Spell:     int32(rec.MasterySpell),
			Pronounce: int32(rec.MasteryPronounce),
			Overall:   rec.MasteryOverall,
		},
		Review: entity.ReviewTiming{
			IntervalDays: rec.ReviewIntervalDays,
			FailCount:    rec.ReviewFailCount,
		},
		QueryCount: rec.QueryCount,
		Sentences:  rec.Sentences,
		Relations:  rec.Relations,
		CreatedBy:  rec.CreatedBy,
		CreatedAt:  rec.CreatedAt,
		UpdatedAt:  rec.UpdatedAt,
	}

	if rec.ReviewLastReviewAt != nil {
		userWord.Review.LastReviewAt = *rec.ReviewLastReviewAt
	}
	if rec.ReviewNextReviewAt != nil {
		userWord.Review.NextReviewAt = *rec.ReviewNextReviewAt
	}
	if rec.Notes != nil {
		userWord.Notes = *rec.Notes
	}

	return userWord
}

func translateUserWordError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return entity.ErrDuplicateUserWord
	}
	if entdb.IsNotFound(err) {
		return entity.ErrUserWordNotFound
	}
	return err
}
