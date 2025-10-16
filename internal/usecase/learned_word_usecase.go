package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/repository"
)

// LearnedWordUsecase encapsulates business logic for managing user vocabulary entries.
type LearnedWordUsecase interface {
	CollectWord(ctx context.Context, userID int64, word *entity.LearnedWord) (*entity.LearnedWord, error)
	UpdateMastery(ctx context.Context, userID, id int64, mastery entity.MasteryBreakdown, review entity.ReviewTiming, notes string) (*entity.LearnedWord, error)
	ListLearnedWords(ctx context.Context, filter *repository.ListLearnedWordQuery) ([]entity.LearnedWord, int64, error)
	DeleteLearnedWord(ctx context.Context, userID, id int64) error
}

// NewLearnedWordUsecase wires the repository with default behaviour.
func NewLearnedWordUsecase(repo repository.LearnedWordRepository) LearnedWordUsecase {
	return &learnedWordUsecase{
		repo:  repo,
		clock: time.Now,
	}
}

type learnedWordUsecase struct {
	repo  repository.LearnedWordRepository
	clock func() time.Time
}

func (u *learnedWordUsecase) CollectWord(ctx context.Context, userID int64, word *entity.LearnedWord) (*entity.LearnedWord, error) {
	if word == nil {
		return nil, entity.ErrInvalidLearnedWordText
	}
	text := strings.TrimSpace(word.Term)
	if text == "" {
		return nil, entity.ErrInvalidLearnedWordText
	}

	existing, err := u.repo.FindByWord(ctx, userID, text)
	if err != nil {
		return nil, err
	}

	now := u.clock()
	if existing != nil {
		// Update lightweight fields on duplicate collects.
		existing.QueryCount++
		if word.Language.Code() != "" {
			existing.Language = entity.NormalizeLanguage(word.Language)
		}
		if word.Notes != "" {
			existing.Notes = word.Notes
		}
		existing.Mastery = word.Mastery
		existing.Review = word.Review
		existing.Normalize(now)
		return u.repo.Update(ctx, existing)
	}

	copy := *word
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

func (u *learnedWordUsecase) UpdateMastery(ctx context.Context, userID, id int64, mastery entity.MasteryBreakdown, review entity.ReviewTiming, notes string) (*entity.LearnedWord, error) {
	if id <= 0 {
		return nil, entity.ErrLearnedWordNotFound
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

func (u *learnedWordUsecase) ListLearnedWords(ctx context.Context, query *repository.ListLearnedWordQuery) ([]entity.LearnedWord, int64, error) {
	return u.repo.List(ctx, query)
}

func (u *learnedWordUsecase) DeleteLearnedWord(ctx context.Context, userID, id int64) error {
	if id <= 0 {
		return entity.ErrLearnedWordNotFound
	}
	return u.repo.Delete(ctx, userID, id)
}
