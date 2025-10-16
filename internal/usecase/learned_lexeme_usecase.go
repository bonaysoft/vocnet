package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/repository"
)

// LearnedLexemeUsecase encapsulates business logic for managing user vocabulary entries.
type LearnedLexemeUsecase interface {
	CollectLexeme(ctx context.Context, userID int64, lexeme *entity.LearnedLexeme) (*entity.LearnedLexeme, error)
	UpdateMastery(ctx context.Context, userID, id int64, mastery entity.MasteryBreakdown, review entity.ReviewTiming, notes string) (*entity.LearnedLexeme, error)
	ListLearnedLexemes(ctx context.Context, filter *repository.ListLearnedLexemeQuery) ([]entity.LearnedLexeme, int64, error)
	DeleteLearnedLexeme(ctx context.Context, userID, id int64) error
}

// NewLearnedLexemeUsecase wires the repository with default behaviour.
func NewLearnedLexemeUsecase(repo repository.LearnedLexemeRepository) LearnedLexemeUsecase {
	return &learnedLexemeUsecase{
		repo:  repo,
		clock: time.Now,
	}
}

type learnedLexemeUsecase struct {
	repo  repository.LearnedLexemeRepository
	clock func() time.Time
}

func (u *learnedLexemeUsecase) CollectLexeme(ctx context.Context, userID int64, lexeme *entity.LearnedLexeme) (*entity.LearnedLexeme, error) {
	if lexeme == nil {
		return nil, entity.ErrInvalidLearnedLexemeText
	}
	text := strings.TrimSpace(lexeme.Term)
	if text == "" {
		return nil, entity.ErrInvalidLearnedLexemeText
	}

	existing, err := u.repo.FindByTerm(ctx, userID, text)
	if err != nil {
		return nil, err
	}

	now := u.clock()
	if existing != nil {
		// Update lightweight fields on duplicate collects.
		existing.QueryCount++
		if lexeme.Language.Code() != "" {
			existing.Language = entity.NormalizeLanguage(lexeme.Language)
		}
		if lexeme.Notes != "" {
			existing.Notes = lexeme.Notes
		}
		existing.Mastery = lexeme.Mastery
		existing.Review = lexeme.Review
		existing.Normalize(now)
		return u.repo.Update(ctx, existing)
	}

	copy := *lexeme
	copy.Term = text
	copy.UserID = userID
	if copy.QueryCount == 0 {
		copy.QueryCount = 1
	}
	if copy.CreatedBy == "" {
		copy.CreatedBy = "user"
	}
	copy.Normalize(now)

	created, err := u.repo.Create(ctx, &copy)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (u *learnedLexemeUsecase) UpdateMastery(ctx context.Context, userID, id int64, mastery entity.MasteryBreakdown, review entity.ReviewTiming, notes string) (*entity.LearnedLexeme, error) {
	if id <= 0 {
		return nil, entity.ErrLearnedLexemeNotFound
	}

	existing, err := u.repo.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	existing.Mastery = mastery
	existing.Review = review
	if notes != "" {
		existing.Notes = notes
	}
	existing.Normalize(u.clock())

	return u.repo.Update(ctx, existing)
}

func (u *learnedLexemeUsecase) ListLearnedLexemes(ctx context.Context, query *repository.ListLearnedLexemeQuery) ([]entity.LearnedLexeme, int64, error) {
	return u.repo.List(ctx, query)
}

func (u *learnedLexemeUsecase) DeleteLearnedLexeme(ctx context.Context, userID, id int64) error {
	if id <= 0 {
		return entity.ErrLearnedLexemeNotFound
	}
	return u.repo.Delete(ctx, userID, id)
}
