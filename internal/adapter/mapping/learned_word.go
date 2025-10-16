package mapping

import (
	"strings"

	commonv1 "github.com/eslsoft/vocnet/pkg/api/common/v1"
	dictv1 "github.com/eslsoft/vocnet/pkg/api/dict/v1"
	learningv1 "github.com/eslsoft/vocnet/pkg/api/learning/v1"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/eslsoft/vocnet/internal/entity"
)

func FromPbLearnedWord(in *learningv1.LearnedWord) *entity.LearnedWord {
	return &entity.LearnedWord{
		ID:       in.GetId(),
		Term:     strings.TrimSpace(in.Spec.GetTerm()),
		Language: FromPbLanguage(in.Spec.GetLanguage()),
		Mastery: entity.MasteryBreakdown{
			Overall: in.Spec.MasteryLevel,
		},
		// Notes:      in.Spec.GetNotes(),
		Sentences: lo.Map(in.Spec.GetSentences(), func(s *dictv1.Sentence, _ int) entity.Sentence {
			return entity.Sentence{
				Text:      strings.TrimSpace(s.GetText()),
				Source:    int32(s.GetSource()),
				SourceRef: strings.TrimSpace(s.GetSourceRef()),
			}
		}),
		Relations: lo.Map(in.Spec.GetRelations(), func(rel *learningv1.LearnedWordRelation, _ int) entity.LearnedWordRelation {
			return entity.LearnedWordRelation{
				Word:         rel.GetWord(),
				RelationType: int32(rel.GetRelationType()),
				Note:         rel.GetNote(),
			}
		}),
	}
}

func ToPbLearnedWord(in *entity.LearnedWord) *learningv1.LearnedWord {
	out := &learningv1.LearnedWord{
		Id: in.ID,
		Spec: &learningv1.LearnedWordSpec{
			Term:     in.Term,
			Language: ToPbLanguage(in.Language),
			Sentences: lo.Map(in.Sentences, func(s entity.Sentence, _ int) *dictv1.Sentence {
				return &dictv1.Sentence{
					Text:      s.Text,
					Source:    commonv1.SourceType(s.Source),
					SourceRef: s.SourceRef,
				}
			}),
			Relations: lo.Map(in.Relations, func(rel entity.LearnedWordRelation, _ int) *learningv1.LearnedWordRelation {
				return &learningv1.LearnedWordRelation{
					Word:         rel.Word,
					RelationType: commonv1.RelationType(rel.RelationType),
					Note:         rel.Note,
					CreatedAt:    timestamppb.New(rel.CreatedAt),
					UpdatedAt:    timestamppb.New(rel.UpdatedAt),
				}
			}),
			// Notes: in.Notes,
		},
		Status: &learningv1.LearnedWordStatus{
			Mastery:      ToPbMastery(in.Mastery),
			ReviewTiming: ToPbReview(in.Review),
			QueryCount:   in.QueryCount,
			CreatedBy:    in.CreatedBy,
			CreatedAt:    timestamppb.New(in.CreatedAt),
			UpdatedAt:    timestamppb.New(in.UpdatedAt),
		},
	}

	return out
}

func FromPbMastery(in *learningv1.MasteryBreakdown) entity.MasteryBreakdown {
	return entity.MasteryBreakdown{
		Listen:    in.GetListen(),
		Read:      in.GetRead(),
		Spell:     in.GetSpell(),
		Pronounce: in.GetPronounce(),
		Overall:   in.GetOverall(),
	}
}

func ToPbMastery(in entity.MasteryBreakdown) *learningv1.MasteryBreakdown {
	return &learningv1.MasteryBreakdown{
		Listen:    in.Listen,
		Read:      in.Read,
		Spell:     in.Spell,
		Pronounce: in.Pronounce,
		Overall:   in.Overall,
	}
}

func ToPbReview(in entity.ReviewTiming) *learningv1.ReviewTiming {
	return &learningv1.ReviewTiming{
		LastReviewAt: timestamppb.New(in.LastReviewAt),
		NextReviewAt: timestamppb.New(in.NextReviewAt),
		IntervalDays: in.IntervalDays,
		FailCount:    in.FailCount,
	}
}
