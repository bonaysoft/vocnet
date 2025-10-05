package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/repository"
)

// UserWordUsecase encapsulates business logic for managing user vocabulary entries.
type UserWordUsecase interface {
	CollectWord(ctx context.Context, userID int64, word *entity.UserWord) (*entity.UserWord, error)
	UpdateMastery(ctx context.Context, userID, id int64, mastery entity.MasteryBreakdown, review entity.ReviewTiming, notes string) (*entity.UserWord, error)
	ListUserWords(ctx context.Context, filter *repository.ListUserWordQuery) ([]*entity.UserWord, int64, error)
	DeleteUserWord(ctx context.Context, userID, id int64) error
}

// NewUserWordUsecase wires the repository with default behaviour.
func NewUserWordUsecase(repo repository.UserWordRepository) UserWordUsecase {
	return &userWordUsecase{
		repo:  repo,
		clock: time.Now,
	}
}

type userWordUsecase struct {
	repo  repository.UserWordRepository
	clock func() time.Time
}

func (u *userWordUsecase) CollectWord(ctx context.Context, userID int64, word *entity.UserWord) (*entity.UserWord, error) {
	if word == nil {
		return nil, entity.ErrInvalidUserWordText
	}
	text := strings.TrimSpace(word.Word)
	if text == "" {
		return nil, entity.ErrInvalidUserWordText
	}

	existing, err := u.repo.FindByWord(ctx, userID, text)
	if err != nil {
		return nil, err
	}

	now := u.clock()
	if existing != nil {
		// Update lightweight fields on duplicate collects.
		existing.QueryCount++
		if word.Notes != "" {
			existing.Notes = word.Notes
		}
		existing.Mastery = word.Mastery
		existing.Review = word.Review
		existing.Normalize(now)
		return u.repo.Update(ctx, existing)
	}

	copy := *word
	copy.Word = text
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

func (u *userWordUsecase) UpdateMastery(ctx context.Context, userID, id int64, mastery entity.MasteryBreakdown, review entity.ReviewTiming, notes string) (*entity.UserWord, error) {
	if id <= 0 {
		return nil, entity.ErrUserWordNotFound
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

func (u *userWordUsecase) ListUserWords(ctx context.Context, query *repository.ListUserWordQuery) ([]*entity.UserWord, int64, error) {
	return u.repo.List(ctx, query)
}

func (u *userWordUsecase) DeleteUserWord(ctx context.Context, userID, id int64) error {
	if id <= 0 {
		return entity.ErrUserWordNotFound
	}
	return u.repo.Delete(ctx, userID, id)
}
