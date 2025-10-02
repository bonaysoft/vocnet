package types

import (
	"github.com/eslsoft/vocnet/internal/entity"
)

type WordMeanings []entity.WordDefinition

type WordPhonetics []entity.WordPhonetic

type WordForms []entity.WordFormRef

type WordSentences []entity.Sentence

type WordRelations []entity.WordRelation
