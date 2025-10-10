package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/repository"
)

// minimal in-memory mock repository for testing forms logic
type mockVocRepo struct {
	word         *entity.Word
	forms        []entity.WordFormRef
	lookupErr    error
	listFormsErr error
}

func (m *mockVocRepo) Create(ctx context.Context, word *entity.Word) (*entity.Word, error) {
	return nil, errors.New("not implemented")
}
func (m *mockVocRepo) Update(ctx context.Context, word *entity.Word) (*entity.Word, error) {
	return nil, errors.New("not implemented")
}
func (m *mockVocRepo) GetByID(ctx context.Context, id int64) (*entity.Word, error) {
	return nil, errors.New("not implemented")
}
func (m *mockVocRepo) Lookup(ctx context.Context, text string, language entity.Language) (*entity.Word, error) {
	return m.word, m.lookupErr
}
func (m *mockVocRepo) List(ctx context.Context, filter *repository.ListWordQuery) ([]*entity.Word, int64, error) {
	return nil, 0, errors.New("not implemented")
}
func (m *mockVocRepo) ListFormsByLemma(ctx context.Context, lemma string, language entity.Language) ([]entity.WordFormRef, error) {
	return m.forms, m.listFormsErr
}
func (m *mockVocRepo) Delete(ctx context.Context, id int64) error {
	return errors.New("not implemented")
}

func TestLookup_PopulatesFormsForLemma(t *testing.T) {
	lemmaText := "run"
	repo := &mockVocRepo{word: &entity.Word{ID: 1, Text: lemmaText, Language: entity.LanguageEnglish, WordType: entity.WordTypeLemma}, forms: []entity.WordFormRef{{Text: "ran", WordType: "past"}, {Text: "running", WordType: "ing"}}}
	uc := NewWordUsecase(repo)

	v, err := uc.Lookup(context.Background(), lemmaText, entity.LanguageEnglish)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(v.Forms) != 2 {
		t.Fatalf("expected 2 forms, got %d", len(v.Forms))
	}
}

func TestLookup_NoFormsWhenNotLemma(t *testing.T) {
	lemmaStr := "run"
	repo := &mockVocRepo{word: &entity.Word{ID: 2, Text: "ran", Language: entity.LanguageEnglish, WordType: "past", Lemma: &lemmaStr}, forms: []entity.WordFormRef{{Text: "ran", WordType: "past"}}}
	uc := NewWordUsecase(repo)

	v, err := uc.Lookup(context.Background(), "ran", entity.LanguageEnglish)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(v.Forms) != 0 {
		t.Fatalf("expected 0 forms for non-lemma, got %d", len(v.Forms))
	}
}
