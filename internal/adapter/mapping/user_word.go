package mapping

import (
	"strings"

	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	dictv1 "github.com/eslsoft/vocnet/api/gen/dict/v1"
	vocnetv1 "github.com/eslsoft/vocnet/api/gen/vocnet/v1"
	"github.com/samber/lo"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/eslsoft/vocnet/internal/entity"
)

func FromPbUserWord(in *vocnetv1.UserWord) *entity.UserWord {
	return &entity.UserWord{
		ID:       in.GetId(),
		Word:     strings.TrimSpace(in.Spec.GetWord()),
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
		Relations: lo.Map(in.Spec.GetRelations(), func(rel *vocnetv1.UserWordRelation, _ int) entity.UserWordRelation {
			return entity.UserWordRelation{
				Word:         rel.GetWord(),
				RelationType: int32(rel.GetRelationType()),
				Note:         rel.GetNote(),
			}
		}),
	}
}

func ToPbUserWord(in *entity.UserWord) *vocnetv1.UserWord {
	out := &vocnetv1.UserWord{
		Id: in.ID,
		Spec: &vocnetv1.UserWordSpec{
			Word:     in.Word,
			Language: ToPbLanguage(in.Language),
			Sentences: lo.Map(in.Sentences, func(s entity.Sentence, _ int) *dictv1.Sentence {
				return &dictv1.Sentence{
					Text:      s.Text,
					Source:    commonv1.SourceType(s.Source),
					SourceRef: s.SourceRef,
				}
			}),
			Relations: lo.Map(in.Relations, func(rel entity.UserWordRelation, _ int) *vocnetv1.UserWordRelation {
				return &vocnetv1.UserWordRelation{
					Word:         rel.Word,
					RelationType: commonv1.RelationType(rel.RelationType),
					Note:         rel.Note,
					CreatedAt:    timestamppb.New(rel.CreatedAt),
					UpdatedAt:    timestamppb.New(rel.UpdatedAt),
				}
			}),
			// Notes: in.Notes,
		},
		Status: &vocnetv1.UserWordStatus{
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

func FromPbMastery(in *vocnetv1.MasteryBreakdown) entity.MasteryBreakdown {
	return entity.MasteryBreakdown{
		Listen:    in.GetListen(),
		Read:      in.GetRead(),
		Spell:     in.GetSpell(),
		Pronounce: in.GetPronounce(),
		Overall:   in.GetOverall(),
	}
}

func ToPbMastery(in entity.MasteryBreakdown) *vocnetv1.MasteryBreakdown {
	return &vocnetv1.MasteryBreakdown{
		Listen:    in.Listen,
		Read:      in.Read,
		Spell:     in.Spell,
		Pronounce: in.Pronounce,
		Overall:   in.Overall,
	}
}

func ToPbReview(in entity.ReviewTiming) *vocnetv1.ReviewTiming {
	return &vocnetv1.ReviewTiming{
		LastReviewAt: timestamppb.New(in.LastReviewAt),
		NextReviewAt: timestamppb.New(in.NextReviewAt),
		IntervalDays: in.IntervalDays,
		FailCount:    in.FailCount,
	}
}
