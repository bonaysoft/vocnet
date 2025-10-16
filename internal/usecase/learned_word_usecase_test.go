package usecase

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/repository"
)

type fakeLearnedWordRepo struct {
	mu    sync.RWMutex
	seq   int64
	items map[int64]*entity.LearnedWord
}

func newFakeLearnedWordRepo() *fakeLearnedWordRepo {
	return &fakeLearnedWordRepo{items: make(map[int64]*entity.LearnedWord)}
}

func (r *fakeLearnedWordRepo) Create(ctx context.Context, uw *entity.LearnedWord) (*entity.LearnedWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.lookupLocked(uw.UserID, uw.Term); ok {
		return nil, entity.ErrDuplicateLearnedWord
	}
	r.seq++
	copy := cloneLearnedWord(uw)
	copy.ID = r.seq
	r.items[copy.ID] = copy
	return cloneLearnedWord(copy), nil
}

func (r *fakeLearnedWordRepo) Update(ctx context.Context, uw *entity.LearnedWord) (*entity.LearnedWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.items[uw.ID]
	if !ok || existing.UserID != uw.UserID {
		return nil, entity.ErrLearnedWordNotFound
	}
	if other, ok := r.lookupLocked(uw.UserID, uw.Term); ok && other.ID != uw.ID {
		return nil, entity.ErrDuplicateLearnedWord
	}
	copy := cloneLearnedWord(uw)
	r.items[copy.ID] = copy
	return cloneLearnedWord(copy), nil
}

func (r *fakeLearnedWordRepo) GetByID(ctx context.Context, userID, id int64) (*entity.LearnedWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok || item.UserID != userID {
		return nil, entity.ErrLearnedWordNotFound
	}
	return cloneLearnedWord(item), nil
}

func (r *fakeLearnedWordRepo) FindByWord(ctx context.Context, userID int64, word string) (*entity.LearnedWord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if item, ok := r.lookupLocked(userID, word); ok {
		return cloneLearnedWord(item), nil
	}
	return nil, nil
}

func (r *fakeLearnedWordRepo) List(ctx context.Context, query *repository.ListLearnedWordQuery) ([]entity.LearnedWord, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if query == nil {
		return nil, 0, errors.New("list query required")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	keyword := strings.ToLower(strings.TrimSpace(extractKeyword(query.Filter)))
	var filtered []*entity.LearnedWord
	for _, item := range r.items {
		if item.UserID != query.UserID {
			continue
		}
		if keyword != "" {
			if !strings.Contains(strings.ToLower(item.Term), keyword) && !strings.Contains(strings.ToLower(item.Notes), keyword) {
				continue
			}
		}
		filtered = append(filtered, cloneLearnedWord(item))
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt.Equal(filtered[j].CreatedAt) {
			return filtered[i].ID < filtered[j].ID
		}
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	total := int64(len(filtered))
	pageNo := query.PageNo
	if pageNo <= 0 {
		pageNo = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = int32(len(filtered))
	}
	start := int((pageNo - 1) * pageSize)
	if start >= len(filtered) {
		return []entity.LearnedWord{}, total, nil
	}
	if start < 0 {
		start = 0
	}
	end := start + int(pageSize)
	if end > len(filtered) {
		end = len(filtered)
	}
	result := make([]entity.LearnedWord, 0, end-start)
	for _, item := range filtered[start:end] {
		if clone := cloneLearnedWord(item); clone != nil {
			result = append(result, *clone)
		}
	}
	return result, total, nil
}

func (r *fakeLearnedWordRepo) Delete(ctx context.Context, userID, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.UserID != userID {
		return entity.ErrLearnedWordNotFound
	}
	delete(r.items, id)
	return nil
}

func (r *fakeLearnedWordRepo) lookupLocked(userID int64, word string) (*entity.LearnedWord, bool) {
	if word == "" {
		return nil, false
	}
	needle := strings.ToLower(word)
	for _, item := range r.items {
		if item.UserID == userID && strings.ToLower(item.Term) == needle {
			return item, true
		}
	}
	return nil, false
}

func cloneLearnedWord(src *entity.LearnedWord) *entity.LearnedWord {
	if src == nil {
		return nil
	}
	copy := *src
	if src.Sentences != nil {
		copy.Sentences = append([]entity.Sentence(nil), src.Sentences...)
	}
	if src.Relations != nil {
		copy.Relations = append([]entity.LearnedWordRelation(nil), src.Relations...)
	}
	return &copy
}

func TestCollectWordCreatesNewEntry(t *testing.T) {
	repo := newFakeLearnedWordRepo()
	uc := NewLearnedWordUsecase(repo)
	impl := uc.(*learnedWordUsecase)
	fixed := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	impl.clock = func() time.Time { return fixed }

	got, err := uc.CollectWord(context.Background(), 42, &entity.LearnedWord{Term: "Hello", CreatedBy: "tester"})
	if err != nil {
		t.Fatalf("CollectWord returned error: %v", err)
	}
	if got == nil {
		t.Fatal("CollectWord returned nil result")
	}
	if got.ID == 0 {
		t.Errorf("expected ID to be set, got %d", got.ID)
	}
	if got.Term != "Hello" {
		t.Errorf("expected word to be 'Hello', got %q", got.Term)
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
	repo := newFakeLearnedWordRepo()
	uc := NewLearnedWordUsecase(repo)
	impl := uc.(*learnedWordUsecase)
	first := time.Date(2024, 1, 2, 8, 0, 0, 0, time.UTC)
	impl.clock = func() time.Time { return first }

	_, err := uc.CollectWord(context.Background(), 1, &entity.LearnedWord{Term: "Apple"})
	if err != nil {
		t.Fatalf("CollectWord initial call failed: %v", err)
	}

	second := time.Date(2024, 1, 3, 9, 0, 0, 0, time.UTC)
	impl.clock = func() time.Time { return second }
	updatedMastery := entity.MasteryBreakdown{Overall: 250}
	res, err := uc.CollectWord(context.Background(), 1, &entity.LearnedWord{Term: "Apple", Notes: "updated", Mastery: updatedMastery, Language: "fr"})
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
	repo := newFakeLearnedWordRepo()
	uc := NewLearnedWordUsecase(repo)
	impl := uc.(*learnedWordUsecase)
	impl.clock = func() time.Time { return time.Date(2024, 1, 4, 10, 0, 0, 0, time.UTC) }

	created, err := uc.CollectWord(context.Background(), 9, &entity.LearnedWord{Term: "Bridge"})
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

func TestListLearnedWordsFiltersByKeyword(t *testing.T) {
	repo := newFakeLearnedWordRepo()
	uc := NewLearnedWordUsecase(repo)
	impl := uc.(*learnedWordUsecase)
	impl.clock = time.Now

	_, _ = uc.CollectWord(context.Background(), 5, &entity.LearnedWord{Term: "Comet", Notes: "space"})
	_, _ = uc.CollectWord(context.Background(), 5, &entity.LearnedWord{Term: "Forest", Notes: "trees"})

	query := &repository.ListLearnedWordQuery{
		Pagination:  repository.Pagination{PageNo: 1, PageSize: 10},
		FilterOrder: repository.FilterOrder{Filter: "keyword == \"tre\""},
		UserID:      5,
	}
	items, total, err := uc.ListLearnedWords(context.Background(), query)
	if err != nil {
		t.Fatalf("ListLearnedWords returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(items) != 1 || items[0].Term != "Forest" {
		t.Fatalf("expected to retrieve Forest entry, got %+v", items)
	}
}

func extractKeyword(filter string) string {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return ""
	}
	for _, quote := range []string{"\"", "'"} {
		s := strings.Split(filter, quote)
		if len(s) >= 3 {
			return s[1]
		}
	}
	return strings.Trim(filter, "\"'")
}
