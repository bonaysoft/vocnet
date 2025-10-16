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
	entlearnedword "github.com/eslsoft/vocnet/internal/infrastructure/database/ent/learnedword"
	"github.com/eslsoft/vocnet/internal/repository"
	"github.com/eslsoft/vocnet/pkg/filterexpr"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/samber/lo"
)

type LearnedWordRepository struct {
	client *entdb.Client
}

func int32ToInt16(value int32, field string) (int16, error) {
	if value > math.MaxInt16 || value < math.MinInt16 {
		return 0, fmt.Errorf("%s out of int16 range: %d", field, value)
	}
	return int16(value), nil
}

// NewLearnedWordRepository constructs an ent-backed repository.
func NewLearnedWordRepository(client *entdb.Client) repository.LearnedWordRepository {
	return &LearnedWordRepository{client: client}
}

type listLearnedWordsParams struct {
	Keyword       string
	Words         []string
	PrimaryKey    string
	PrimaryDesc   bool
	SecondaryKey  string
	SecondaryDesc bool
}

func (r *LearnedWordRepository) Create(ctx context.Context, learnedWord *entity.LearnedWord) (*entity.LearnedWord, error) {
	listen, err := int32ToInt16(learnedWord.Mastery.Listen, "mastery.listen")
	if err != nil {
		return nil, err
	}
	read, err := int32ToInt16(learnedWord.Mastery.Read, "mastery.read")
	if err != nil {
		return nil, err
	}
	spell, err := int32ToInt16(learnedWord.Mastery.Spell, "mastery.spell")
	if err != nil {
		return nil, err
	}
	pronounce, err := int32ToInt16(learnedWord.Mastery.Pronounce, "mastery.pronounce")
	if err != nil {
		return nil, err
	}

	builder := r.client.LearnedWord.Create().
		SetUserID(learnedWord.UserID).
		SetTerm(learnedWord.Term).
		SetNormalized(entity.NormalizeWordToken(learnedWord.Term)).
		SetLanguage(entity.NormalizeLanguage(learnedWord.Language).Code()).
		SetMasteryListen(listen).
		SetMasteryRead(read).
		SetMasterySpell(spell).
		SetMasteryPronounce(pronounce).
		SetMasteryOverall(learnedWord.Mastery.Overall).
		SetReviewIntervalDays(learnedWord.Review.IntervalDays).
		SetReviewFailCount(learnedWord.Review.FailCount).
		SetQueryCount(learnedWord.QueryCount).
		SetSentences(learnedWord.Sentences).
		SetRelations(learnedWord.Relations).
		SetCreatedBy(learnedWord.CreatedBy).
		SetCreatedAt(learnedWord.CreatedAt).
		SetUpdatedAt(learnedWord.UpdatedAt)

	if !learnedWord.Review.LastReviewAt.IsZero() {
		builder.SetReviewLastReviewAt(learnedWord.Review.LastReviewAt)
	}
	if !learnedWord.Review.NextReviewAt.IsZero() {
		builder.SetReviewNextReviewAt(learnedWord.Review.NextReviewAt)
	}
	if learnedWord.Notes != "" {
		builder.SetNotes(learnedWord.Notes)
	}

	rec, err := builder.Save(ctx)
	if err != nil {
		return nil, translateLearnedWordError(err)
	}
	return mapEntLearnedWord(rec), nil
}

func (r *LearnedWordRepository) Update(ctx context.Context, learnedWord *entity.LearnedWord) (*entity.LearnedWord, error) {
	listen, err := int32ToInt16(learnedWord.Mastery.Listen, "mastery.listen")
	if err != nil {
		return nil, err
	}
	read, err := int32ToInt16(learnedWord.Mastery.Read, "mastery.read")
	if err != nil {
		return nil, err
	}
	spell, err := int32ToInt16(learnedWord.Mastery.Spell, "mastery.spell")
	if err != nil {
		return nil, err
	}
	pronounce, err := int32ToInt16(learnedWord.Mastery.Pronounce, "mastery.pronounce")
	if err != nil {
		return nil, err
	}

	mutation := r.client.LearnedWord.UpdateOneID(int(learnedWord.ID)).
		Where(entlearnedword.UserIDEQ(learnedWord.UserID)).
		SetTerm(learnedWord.Term).
		SetNormalized(entity.NormalizeWordToken(learnedWord.Term)).
		SetLanguage(entity.NormalizeLanguage(learnedWord.Language).Code()).
		SetMasteryListen(listen).
		SetMasteryRead(read).
		SetMasterySpell(spell).
		SetMasteryPronounce(pronounce).
		SetMasteryOverall(learnedWord.Mastery.Overall).
		SetReviewIntervalDays(learnedWord.Review.IntervalDays).
		SetReviewFailCount(learnedWord.Review.FailCount).
		SetQueryCount(learnedWord.QueryCount).
		SetSentences(learnedWord.Sentences).
		SetRelations(learnedWord.Relations).
		SetCreatedBy(learnedWord.CreatedBy).
		SetUpdatedAt(learnedWord.UpdatedAt)

	if !learnedWord.Review.LastReviewAt.IsZero() {
		mutation.SetReviewLastReviewAt(learnedWord.Review.LastReviewAt)
	} else {
		mutation.ClearReviewLastReviewAt()
	}
	if !learnedWord.Review.NextReviewAt.IsZero() {
		mutation.SetReviewNextReviewAt(learnedWord.Review.NextReviewAt)
	} else {
		mutation.ClearReviewNextReviewAt()
	}

	if learnedWord.Notes != "" {
		mutation.SetNotes(learnedWord.Notes)
	} else {
		mutation.ClearNotes()
	}

	rec, err := mutation.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, entity.ErrLearnedWordNotFound
		}
		return nil, translateLearnedWordError(err)
	}

	return mapEntLearnedWord(rec), nil
}

func (r *LearnedWordRepository) GetByID(ctx context.Context, userID, id int64) (*entity.LearnedWord, error) {
	rec, err := r.client.LearnedWord.Query().
		Where(
			entlearnedword.IDEQ(int(id)),
			entlearnedword.UserIDEQ(userID),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, entity.ErrLearnedWordNotFound
		}
		return nil, fmt.Errorf("get user word: %w", err)
	}
	return mapEntLearnedWord(rec), nil
}

func (r *LearnedWordRepository) FindByWord(ctx context.Context, userID int64, word string) (*entity.LearnedWord, error) {
	if word == "" {
		return nil, nil
	}

	rec, err := r.client.LearnedWord.Query().
		Where(
			entlearnedword.UserIDEQ(userID),
			entlearnedword.TermEQ(word),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user word: %w", err)
	}
	return mapEntLearnedWord(rec), nil
}

func (r *LearnedWordRepository) List(ctx context.Context, query *repository.ListLearnedWordQuery) ([]entity.LearnedWord, int64, error) {
	var params listLearnedWordsParams
	if err := filterexpr.Bind(query, &params, listLearnedWordsSchema); err != nil {
		return nil, 0, err
	}

	qbuilder := r.client.LearnedWord.Query().
		Where(entlearnedword.UserIDEQ(query.UserID))

	applyLearnedWordFilters(qbuilder, params)

	total, err := qbuilder.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count user words: %w", err)
	}

	applyLearnedWordOrdering(qbuilder, params)

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

	results := make([]entity.LearnedWord, 0, len(rows))
	for _, row := range rows {
		if mapped := mapEntLearnedWord(row); mapped != nil {
			results = append(results, *mapped)
		}
	}

	return results, int64(total), nil
}

func (r *LearnedWordRepository) Delete(ctx context.Context, userID, id int64) error {
	affected, err := r.client.LearnedWord.Delete().
		Where(
			entlearnedword.IDEQ(int(id)),
			entlearnedword.UserIDEQ(userID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete user word: %w", err)
	}
	if affected == 0 {
		return entity.ErrLearnedWordNotFound
	}
	return nil
}

func applyLearnedWordFilters(q *entdb.LearnedWordQuery, params listLearnedWordsParams) {
	if params.Keyword != "" {
		q.Where(entlearnedword.TermContainsFold(params.Keyword))
	}
	if words := uniqueFolded(params.Words); len(words) > 0 {
		q.Where(entlearnedword.NormalizedIn(lo.Map(words, func(word string, _ int) string { return strings.ToLower(word) })...))
	}
}

func applyLearnedWordOrdering(q *entdb.LearnedWordQuery, params listLearnedWordsParams) {
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
				q.Order(entlearnedword.ByCreatedAt(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entlearnedword.ByCreatedAt(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "updated_at":
			if term.desc {
				q.Order(entlearnedword.ByUpdatedAt(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entlearnedword.ByUpdatedAt(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "word":
			if term.desc {
				q.Order(entlearnedword.ByTerm(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entlearnedword.ByTerm(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "mastery_overall":
			if term.desc {
				q.Order(entlearnedword.ByMasteryOverall(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entlearnedword.ByMasteryOverall(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "id":
			if term.desc {
				q.Order(entlearnedword.ByID(sql.OrderDesc()))
			} else {
				q.Order(entlearnedword.ByID())
			}
		}
	}

	q.Order(entlearnedword.ByID())
}

func mapEntLearnedWord(rec *entdb.LearnedWord) *entity.LearnedWord {
	if rec == nil {
		return nil
	}

	LearnedWord := &entity.LearnedWord{
		ID:       int64(rec.ID),
		UserID:   rec.UserID,
		Term:     rec.Term,
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
		LearnedWord.Review.LastReviewAt = *rec.ReviewLastReviewAt
	}
	if rec.ReviewNextReviewAt != nil {
		LearnedWord.Review.NextReviewAt = *rec.ReviewNextReviewAt
	}
	if rec.Notes != nil {
		LearnedWord.Notes = *rec.Notes
	}

	return LearnedWord
}

func translateLearnedWordError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return entity.ErrDuplicateLearnedWord
	}
	if entdb.IsNotFound(err) {
		return entity.ErrLearnedWordNotFound
	}
	return err
}
