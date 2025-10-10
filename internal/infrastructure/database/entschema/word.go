package entschema

import (
	"time"

	"github.com/eslsoft/vocnet/internal/infrastructure/database/types"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Word holds the schema definition for the words table.
type Word struct {
	ent.Schema
}

// Fields of the Word.
func (Word) Fields() []ent.Field {
	return []ent.Field{
		field.String("text").NotEmpty(),
		field.String("language").Default("en"),
		field.String("word_type").Default("lemma"),
		field.String("lemma").Optional().Nillable(),
		field.JSON("phonetics", types.WordPhonetics{}).
			Default(types.WordPhonetics{}),
		field.JSON("meanings", types.WordMeanings{}).
			Default(types.WordMeanings{}),
		field.JSON("tags", []string{}).
			Default([]string{}).
			SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.JSON("phrases", types.Phrases{}).
			Default(types.Phrases{}),
		field.JSON("sentences", types.Sentences{}).
			Default(types.Sentences{}),
		field.JSON("relations", types.WordRelations{}).
			Default(types.WordRelations{}),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the Word.
func (Word) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("language", "text", "word_type").
			Unique().
			StorageKey("uq_words_lang_text_type"),
	}
}

// Annotations of the Word.
func (Word) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "words",
			Checks: map[string]string{
				"chk_words_lemma_ref": "((word_type = 'lemma' AND lemma IS NULL) OR (word_type <> 'lemma' AND lemma IS NOT NULL))",
			},
		},
	}
}
