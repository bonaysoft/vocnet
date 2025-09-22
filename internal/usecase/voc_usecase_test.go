package usecase

import (
	"context"
	"testing"

	"github.com/eslsoft/vocnet/internal/entity"
)

// minimal in-memory mock repository for testing forms logic
type mockVocRepo struct {
	voc          *entity.Voc
	forms        []entity.VocFormRef
	lookupErr    error
	listFormsErr error
}

func (m *mockVocRepo) Lookup(ctx context.Context, text string, language string) (*entity.Voc, error) {
	return m.voc, m.lookupErr
}
func (m *mockVocRepo) ListFormsByLemma(ctx context.Context, lemma string, language string) ([]entity.VocFormRef, error) {
	return m.forms, m.listFormsErr
}

func TestLookup_PopulatesFormsForLemma(t *testing.T) {
	lemmaText := "run"
	repo := &mockVocRepo{voc: &entity.Voc{ID: 1, Text: lemmaText, Language: "en", VocType: "lemma"}, forms: []entity.VocFormRef{{Text: "ran", VocType: "past"}, {Text: "running", VocType: "ing"}}}
	uc := NewWordUsecase(repo, "en")

	v, err := uc.Lookup(context.Background(), lemmaText, "en")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(v.Forms) != 2 {
		t.Fatalf("expected 2 forms, got %d", len(v.Forms))
	}
}

func TestLookup_NoFormsWhenNotLemma(t *testing.T) {
	lemmaStr := "run"
	repo := &mockVocRepo{voc: &entity.Voc{ID: 2, Text: "ran", Language: "en", VocType: "past", Lemma: &lemmaStr}, forms: []entity.VocFormRef{{Text: "ran", VocType: "past"}}}
	uc := NewWordUsecase(repo, "en")

	v, err := uc.Lookup(context.Background(), "ran", "en")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(v.Forms) != 0 {
		t.Fatalf("expected 0 forms for non-lemma, got %d", len(v.Forms))
	}
}
