package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/eslsoft/vocnet/internal/entity"
	entdb "github.com/eslsoft/vocnet/internal/infrastructure/database/ent"
	entlearnedlexeme "github.com/eslsoft/vocnet/internal/infrastructure/database/ent/learnedlexeme"
	entword "github.com/eslsoft/vocnet/internal/infrastructure/database/ent/word"
	"github.com/eslsoft/vocnet/internal/repository"
	"github.com/eslsoft/vocnet/pkg/filterexpr"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/samber/lo"
)

type LearnedLexemeRepository struct {
	client *entdb.Client
}

func int32ToInt16(value int32, field string) (int16, error) {
	if value > math.MaxInt16 || value < math.MinInt16 {
		return 0, fmt.Errorf("%s out of int16 range: %d", field, value)
	}
	return int16(value), nil
}

// NewLearnedLexemeRepository constructs an ent-backed repository.
func NewLearnedLexemeRepository(client *entdb.Client) repository.LearnedLexemeRepository {
	return &LearnedLexemeRepository{client: client}
}

type listLearnedLexemesParams struct {
	Keyword       string
	Lexemes       []string
	Tags          []string
	Categories    []string
	PrimaryKey    string
	PrimaryDesc   bool
	SecondaryKey  string
	SecondaryDesc bool
}

func (r *LearnedLexemeRepository) Create(ctx context.Context, lexeme *entity.LearnedLexeme) (*entity.LearnedLexeme, error) {
	listen, err := int32ToInt16(lexeme.Mastery.Listen, "mastery.listen")
	if err != nil {
		return nil, err
	}
	read, err := int32ToInt16(lexeme.Mastery.Read, "mastery.read")
	if err != nil {
		return nil, err
	}
	spell, err := int32ToInt16(lexeme.Mastery.Spell, "mastery.spell")
	if err != nil {
		return nil, err
	}
	pronounce, err := int32ToInt16(lexeme.Mastery.Pronounce, "mastery.pronounce")
	if err != nil {
		return nil, err
	}

	normalizedTerm := entity.NormalizeWordToken(lexeme.Term)
	languageCode := entity.NormalizeLanguage(lexeme.Language).Code()

	builder := r.client.LearnedLexeme.Create().
		SetUserID(lexeme.UserID).
		SetTerm(lexeme.Term).
		SetNormalized(normalizedTerm).
		SetLanguage(languageCode).
		SetMasteryListen(listen).
		SetMasteryRead(read).
		SetMasterySpell(spell).
		SetMasteryPronounce(pronounce).
		SetMasteryOverall(lexeme.Mastery.Overall).
		SetReviewIntervalDays(lexeme.Review.IntervalDays).
		SetReviewFailCount(lexeme.Review.FailCount).
		SetQueryCount(lexeme.QueryCount).
		SetSentences(lexeme.Sentences).
		SetRelations(lexeme.Relations).
		SetCreatedBy(lexeme.CreatedBy).
		SetCreatedAt(lexeme.CreatedAt).
		SetUpdatedAt(lexeme.UpdatedAt)

	if lexeme.Tags != nil {
		builder.SetTags(append([]string{}, lexeme.Tags...))
	}

	if err := r.attachDictionaryWord(ctx, builder.Mutation(), languageCode, normalizedTerm); err != nil {
		return nil, err
	}

	if !lexeme.Review.LastReviewAt.IsZero() {
		builder.SetReviewLastReviewAt(lexeme.Review.LastReviewAt)
	}
	if !lexeme.Review.NextReviewAt.IsZero() {
		builder.SetReviewNextReviewAt(lexeme.Review.NextReviewAt)
	}
	if lexeme.Notes != "" {
		builder.SetNotes(lexeme.Notes)
	}

	rec, err := builder.Save(ctx)
	if err != nil {
		return nil, translateLearnedLexemeError(err)
	}
	return mapEntLearnedLexeme(rec), nil
}

func (r *LearnedLexemeRepository) Update(ctx context.Context, lexeme *entity.LearnedLexeme) (*entity.LearnedLexeme, error) {
	listen, err := int32ToInt16(lexeme.Mastery.Listen, "mastery.listen")
	if err != nil {
		return nil, err
	}
	read, err := int32ToInt16(lexeme.Mastery.Read, "mastery.read")
	if err != nil {
		return nil, err
	}
	spell, err := int32ToInt16(lexeme.Mastery.Spell, "mastery.spell")
	if err != nil {
		return nil, err
	}
	pronounce, err := int32ToInt16(lexeme.Mastery.Pronounce, "mastery.pronounce")
	if err != nil {
		return nil, err
	}

	normalizedTerm := entity.NormalizeWordToken(lexeme.Term)
	languageCode := entity.NormalizeLanguage(lexeme.Language).Code()

	mutation := r.client.LearnedLexeme.UpdateOneID(int(lexeme.ID)).
		Where(entlearnedlexeme.UserIDEQ(lexeme.UserID)).
		SetTerm(lexeme.Term).
		SetNormalized(normalizedTerm).
		SetLanguage(languageCode).
		SetMasteryListen(listen).
		SetMasteryRead(read).
		SetMasterySpell(spell).
		SetMasteryPronounce(pronounce).
		SetMasteryOverall(lexeme.Mastery.Overall).
		SetReviewIntervalDays(lexeme.Review.IntervalDays).
		SetReviewFailCount(lexeme.Review.FailCount).
		SetQueryCount(lexeme.QueryCount).
		SetSentences(lexeme.Sentences).
		SetRelations(lexeme.Relations).
		SetCreatedBy(lexeme.CreatedBy).
		SetUpdatedAt(lexeme.UpdatedAt)

	if lexeme.Tags != nil {
		mutation.SetTags(append([]string{}, lexeme.Tags...))
	}

	if err := r.attachDictionaryWord(ctx, mutation.Mutation(), languageCode, normalizedTerm); err != nil {
		return nil, err
	}

	if !lexeme.Review.LastReviewAt.IsZero() {
		mutation.SetReviewLastReviewAt(lexeme.Review.LastReviewAt)
	} else {
		mutation.ClearReviewLastReviewAt()
	}
	if !lexeme.Review.NextReviewAt.IsZero() {
		mutation.SetReviewNextReviewAt(lexeme.Review.NextReviewAt)
	} else {
		mutation.ClearReviewNextReviewAt()
	}

	if lexeme.Notes != "" {
		mutation.SetNotes(lexeme.Notes)
	} else {
		mutation.ClearNotes()
	}

	rec, err := mutation.Save(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, entity.ErrLearnedLexemeNotFound
		}
		return nil, translateLearnedLexemeError(err)
	}

	return mapEntLearnedLexeme(rec), nil
}

func (r *LearnedLexemeRepository) GetByID(ctx context.Context, userID, id int64) (*entity.LearnedLexeme, error) {
	rec, err := r.client.LearnedLexeme.Query().
		Where(
			entlearnedlexeme.IDEQ(int(id)),
			entlearnedlexeme.UserIDEQ(userID),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, entity.ErrLearnedLexemeNotFound
		}
		return nil, fmt.Errorf("get user lexeme: %w", err)
	}
	return mapEntLearnedLexeme(rec), nil
}

func (r *LearnedLexemeRepository) FindByTerm(ctx context.Context, userID int64, term string) (*entity.LearnedLexeme, error) {
	if term == "" {
		return nil, nil
	}

	rec, err := r.client.LearnedLexeme.Query().
		Where(
			entlearnedlexeme.UserIDEQ(userID),
			entlearnedlexeme.TermEQ(term),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user lexeme: %w", err)
	}
	return mapEntLearnedLexeme(rec), nil
}

func (r *LearnedLexemeRepository) List(ctx context.Context, query *repository.ListLearnedLexemeQuery) ([]entity.LearnedLexeme, int64, error) {
	var params listLearnedLexemesParams
	if err := filterexpr.Bind(query, &params, listLearnedLexemesSchema); err != nil {
		return nil, 0, err
	}

	qbuilder := r.client.LearnedLexeme.Query().
		Where(entlearnedlexeme.UserIDEQ(query.UserID))

	applyLearnedLexemeFilters(qbuilder, params)

	total, err := qbuilder.Clone().Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count user lexemes: %w", err)
	}

	applyLearnedLexemeOrdering(qbuilder, params)

	offset := query.Offset()
	if offset > 0 {
		qbuilder.Offset(int(offset))
	}
	if query.PageSize > 0 {
		qbuilder.Limit(int(query.PageSize))
	}

	rows, err := qbuilder.All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list user lexemes: %w", err)
	}

	results := make([]entity.LearnedLexeme, 0, len(rows))
	for _, row := range rows {
		if mapped := mapEntLearnedLexeme(row); mapped != nil {
			results = append(results, *mapped)
		}
	}

	return results, int64(total), nil
}

func (r *LearnedLexemeRepository) Delete(ctx context.Context, userID, id int64) error {
	affected, err := r.client.LearnedLexeme.Delete().
		Where(
			entlearnedlexeme.IDEQ(int(id)),
			entlearnedlexeme.UserIDEQ(userID),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete user lexeme: %w", err)
	}
	if affected == 0 {
		return entity.ErrLearnedLexemeNotFound
	}
	return nil
}

func applyLearnedLexemeFilters(q *entdb.LearnedLexemeQuery, params listLearnedLexemesParams) {
	if params.Keyword != "" {
		q.Where(entlearnedlexeme.TermContainsFold(params.Keyword))
	}
	if lexemes := uniqueFolded(params.Lexemes); len(lexemes) > 0 {
		q.Where(entlearnedlexeme.NormalizedIn(lo.Map(lexemes, func(term string, _ int) string { return strings.ToLower(term) })...))
	}
	if tags := uniqueFolded(params.Tags); len(tags) > 0 {
		q.Where(func(s *sql.Selector) {
			column := s.C(entlearnedlexeme.FieldTags)
			for _, tag := range tags {
				s.Where(sqljson.ValueContains(column, tag))
			}
		})
	}
	if categories := uniqueFolded(params.Categories); len(categories) > 0 {
		q.Where(entlearnedlexeme.HasWordWith(func(s *sql.Selector) {
			column := s.C(entword.FieldCategories)
			for _, category := range categories {
				s.Where(sqljson.ValueContains(column, category))
			}
		}))
	}
}

func applyLearnedLexemeOrdering(q *entdb.LearnedLexemeQuery, params listLearnedLexemesParams) {
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
				q.Order(entlearnedlexeme.ByCreatedAt(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entlearnedlexeme.ByCreatedAt(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "updated_at":
			if term.desc {
				q.Order(entlearnedlexeme.ByUpdatedAt(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entlearnedlexeme.ByUpdatedAt(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "word":
			if term.desc {
				q.Order(entlearnedlexeme.ByTerm(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entlearnedlexeme.ByTerm(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "mastery_overall":
			if term.desc {
				q.Order(entlearnedlexeme.ByMasteryOverall(sql.OrderDesc(), sql.OrderNullsLast()))
			} else {
				q.Order(entlearnedlexeme.ByMasteryOverall(sql.OrderAsc(), sql.OrderNullsLast()))
			}
		case "id":
			if term.desc {
				q.Order(entlearnedlexeme.ByID(sql.OrderDesc()))
			} else {
				q.Order(entlearnedlexeme.ByID())
			}
		}
	}

	q.Order(entlearnedlexeme.ByID())
}

func (r *LearnedLexemeRepository) attachDictionaryWord(ctx context.Context, mut *entdb.LearnedLexemeMutation, languageCode, normalizedTerm string) error {
	if mut == nil {
		return nil
	}

	if normalizedTerm == "" || languageCode == "" {
		mut.ClearWord()
		return nil
	}

	dictWord, err := r.client.Word.Query().
		Where(
			entword.LanguageEQ(languageCode),
			entword.NormalizedEQ(normalizedTerm),
		).
		First(ctx)
	if err != nil {
		if entdb.IsNotFound(err) {
			mut.ClearWord()
			return nil
		}
		return fmt.Errorf("lookup dictionary word: %w", err)
	}

	mut.SetWordID(dictWord.ID)
	return nil
}

func mapEntLearnedLexeme(rec *entdb.LearnedLexeme) *entity.LearnedLexeme {
	if rec == nil {
		return nil
	}

	out := &entity.LearnedLexeme{
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
		Tags:       append([]string{}, rec.Tags...),
		Sentences:  rec.Sentences,
		Relations:  rec.Relations,
		CreatedBy:  rec.CreatedBy,
		CreatedAt:  rec.CreatedAt,
		UpdatedAt:  rec.UpdatedAt,
	}

	if rec.WordID != nil {
		id := int64(*rec.WordID)
		out.WordID = &id
	}

	if rec.ReviewLastReviewAt != nil {
		out.Review.LastReviewAt = *rec.ReviewLastReviewAt
	}
	if rec.ReviewNextReviewAt != nil {
		out.Review.NextReviewAt = *rec.ReviewNextReviewAt
	}
	if rec.Notes != nil {
		out.Notes = *rec.Notes
	}

	return out
}

func translateLearnedLexemeError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return entity.ErrDuplicateLearnedLexeme
	}
	if entdb.IsNotFound(err) {
		return entity.ErrLearnedLexemeNotFound
	}
	return err
}
