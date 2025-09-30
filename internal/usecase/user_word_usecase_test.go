package usecase

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eslsoft/vocnet/internal/entity"
)

type fakeUserWordRepo struct {
	mu    sync.RWMutex
	seq   int64
	items map[int64]*entity.UserWord
}

func newFakeUserWordRepo() *fakeUserWordRepo {
	return &fakeUserWordRepo{items: make(map[int64]*entity.UserWord)}
}

func (r *fakeUserWordRepo) Create(ctx context.Context, uw *entity.UserWord) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.lookupLocked(uw.UserID, uw.Word); ok {
		return nil, entity.ErrDuplicateUserWord
	}
	r.seq++
	copy := cloneUserWord(uw)
	copy.ID = r.seq
	r.items[copy.ID] = copy
	return cloneUserWord(copy), nil
}

func (r *fakeUserWordRepo) Update(ctx context.Context, uw *entity.UserWord) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.items[uw.ID]
	if !ok || existing.UserID != uw.UserID {
		return nil, entity.ErrUserWordNotFound
	}
	if other, ok := r.lookupLocked(uw.UserID, uw.Word); ok && other.ID != uw.ID {
		return nil, entity.ErrDuplicateUserWord
	}
	copy := cloneUserWord(uw)
	r.items[copy.ID] = copy
	return cloneUserWord(copy), nil
}

func (r *fakeUserWordRepo) GetByID(ctx context.Context, userID, id int64) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok || item.UserID != userID {
		return nil, entity.ErrUserWordNotFound
	}
	return cloneUserWord(item), nil
}

func (r *fakeUserWordRepo) FindByWord(ctx context.Context, userID int64, word string) (*entity.UserWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if item, ok := r.lookupLocked(userID, word); ok {
		return cloneUserWord(item), nil
	}
	return nil, nil
}

func (r *fakeUserWordRepo) List(ctx context.Context, filter entity.UserWordFilter) ([]*entity.UserWord, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*entity.UserWord
	lowerKeyword := strings.ToLower(strings.TrimSpace(filter.Keyword))
	for _, item := range r.items {
		if item.UserID != filter.UserID {
			continue
		}
		if lowerKeyword != "" {
			if !strings.Contains(strings.ToLower(item.Word), lowerKeyword) && !strings.Contains(strings.ToLower(item.Notes), lowerKeyword) {
				continue
			}
		}
		filtered = append(filtered, cloneUserWord(item))
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	total := int64(len(filtered))
	start := int(filter.Offset)
	if start >= len(filtered) {
		return []*entity.UserWord{}, total, nil
	}
	end := len(filtered)
	if limit := int(filter.Limit); limit > 0 && start+limit < end {
		end = start + limit
	}
	result := make([]*entity.UserWord, 0, end-start)
	for _, item := range filtered[start:end] {
		result = append(result, cloneUserWord(item))
	}
	return result, total, nil
}

func (r *fakeUserWordRepo) Delete(ctx context.Context, userID, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.UserID != userID {
		return entity.ErrUserWordNotFound
	}
	delete(r.items, id)
	return nil
}

func (r *fakeUserWordRepo) lookupLocked(userID int64, word string) (*entity.UserWord, bool) {
	if word == "" {
		return nil, false
	}
	needle := strings.ToLower(word)
	for _, item := range r.items {
		if item.UserID == userID && strings.ToLower(item.Word) == needle {
			return item, true
		}
	}
	return nil, false
}

func cloneUserWord(src *entity.UserWord) *entity.UserWord {
	if src == nil {
		return nil
	}
	copy := *src
	if src.Review.LastReviewAt != nil {
		last := *src.Review.LastReviewAt
		copy.Review.LastReviewAt = &last
	}
	if src.Review.NextReviewAt != nil {
		next := *src.Review.NextReviewAt
		copy.Review.NextReviewAt = &next
	}
	if src.Sentences != nil {
		copy.Sentences = append([]entity.Sentence(nil), src.Sentences...)
	}
	if src.Relations != nil {
		copy.Relations = append([]entity.WordRelation(nil), src.Relations...)
	}
	return &copy
}

func TestCollectWordCreatesNewEntry(t *testing.T) {
	repo := newFakeUserWordRepo()
	uc := NewUserWordUsecase(repo)
	impl := uc.(*userWordUsecase)
	fixed := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	impl.clock = func() time.Time { return fixed }

	got, err := uc.CollectWord(context.Background(), 42, &entity.UserWord{Word: "Hello", CreatedBy: "tester"})
	if err != nil {
		t.Fatalf("CollectWord returned error: %v", err)
	}
	if got == nil {
		t.Fatal("CollectWord returned nil result")
	}
	if got.ID == 0 {
		t.Errorf("expected ID to be set, got %d", got.ID)
	}
	if got.Word != "Hello" {
		t.Errorf("expected word to be 'Hello', got %q", got.Word)
	}
	if got.QueryCount != 1 {
		t.Errorf("expected query count to default to 1, got %d", got.QueryCount)
	}
	if got.Language != "en" {
		t.Errorf("expected language to default to 'en', got %q", got.Language)
	}
	if !got.CreatedAt.Equal(fixed) {
		t.Errorf("expected created_at to equal %v, got %v", fixed, got.CreatedAt)
	}
}

func TestCollectWordDuplicateUpdatesExisting(t *testing.T) {
	repo := newFakeUserWordRepo()
	uc := NewUserWordUsecase(repo)
	impl := uc.(*userWordUsecase)
	first := time.Date(2024, 1, 2, 8, 0, 0, 0, time.UTC)
	impl.clock = func() time.Time { return first }

	_, err := uc.CollectWord(context.Background(), 1, &entity.UserWord{Word: "Apple"})
	if err != nil {
		t.Fatalf("CollectWord initial call failed: %v", err)
	}

	second := time.Date(2024, 1, 3, 9, 0, 0, 0, time.UTC)
	impl.clock = func() time.Time { return second }
	updatedMastery := entity.MasteryBreakdown{Overall: 250}
	res, err := uc.CollectWord(context.Background(), 1, &entity.UserWord{Word: "Apple", Notes: "updated", Mastery: updatedMastery, Language: "fr"})
	if err != nil {
		t.Fatalf("CollectWord duplicate failed: %v", err)
	}
	if res.QueryCount != 2 {
		t.Errorf("expected query count to increment to 2, got %d", res.QueryCount)
	}
	if res.Notes != "updated" {
		t.Errorf("expected notes to be updated, got %q", res.Notes)
	}
	if res.Mastery.Overall != 250 {
		t.Errorf("expected overall mastery 250, got %d", res.Mastery.Overall)
	}
	if res.Language != "fr" {
		t.Errorf("expected language to update to 'fr', got %q", res.Language)
	}
	if !res.UpdatedAt.Equal(second) {
		t.Errorf("expected updated_at to equal %v, got %v", second, res.UpdatedAt)
	}
}

func TestUpdateMastery(t *testing.T) {
	repo := newFakeUserWordRepo()
	uc := NewUserWordUsecase(repo)
	impl := uc.(*userWordUsecase)
	impl.clock = func() time.Time { return time.Date(2024, 1, 4, 10, 0, 0, 0, time.UTC) }

	created, err := uc.CollectWord(context.Background(), 9, &entity.UserWord{Word: "Bridge"})
	if err != nil {
		t.Fatalf("CollectWord failed: %v", err)
	}

	reviewTime := entity.ReviewTiming{IntervalDays: 2}
	impl.clock = func() time.Time { return time.Date(2024, 1, 5, 11, 0, 0, 0, time.UTC) }
	mastery := entity.MasteryBreakdown{Listen: 2, Read: 3, Overall: 180}
	updated, err := uc.UpdateMastery(context.Background(), 9, created.ID, mastery, reviewTime, "keep going")
	if err != nil {
		t.Fatalf("UpdateMastery failed: %v", err)
	}
	if updated.Mastery != mastery {
		t.Errorf("expected mastery %+v, got %+v", mastery, updated.Mastery)
	}
	if updated.Review.IntervalDays != 2 {
		t.Errorf("expected interval days 2, got %d", updated.Review.IntervalDays)
	}
	if updated.Notes != "keep going" {
		t.Errorf("expected notes to be 'keep going', got %q", updated.Notes)
	}
}

func TestListUserWordsFiltersByKeyword(t *testing.T) {
	repo := newFakeUserWordRepo()
	uc := NewUserWordUsecase(repo)
	impl := uc.(*userWordUsecase)
	impl.clock = func() time.Time { return time.Now() }

	_, _ = uc.CollectWord(context.Background(), 5, &entity.UserWord{Word: "Comet", Notes: "space"})
	_, _ = uc.CollectWord(context.Background(), 5, &entity.UserWord{Word: "Forest", Notes: "trees"})

	filter := entity.UserWordFilter{UserID: 5, Keyword: "tre"}
	items, total, err := uc.ListUserWords(context.Background(), filter)
	if err != nil {
		t.Fatalf("ListUserWords returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(items) != 1 || items[0].Word != "Forest" {
		t.Fatalf("expected to retrieve Forest entry, got %+v", items)
	}
}
