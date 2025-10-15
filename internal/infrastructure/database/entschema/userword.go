package entschema

import (
	"time"

	"github.com/eslsoft/vocnet/internal/entity"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserWord holds the schema definition for the user_words table.
type UserWord struct {
	ent.Schema
}

// Fields of the UserWord.
func (UserWord) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("word").NotEmpty(),
		field.String("normalized").Default(""),
		field.String("language").Default("en"),
		field.Int16("mastery_listen").Default(0),
		field.Int16("mastery_read").Default(0),
		field.Int16("mastery_spell").Default(0),
		field.Int16("mastery_pronounce").Default(0),
		field.Int16("mastery_use").Default(0),
		field.Int32("mastery_overall").Default(0),
		field.Time("review_last_review_at").Optional().Nillable(),
		field.Time("review_next_review_at").Optional().Nillable(),
		field.Int32("review_interval_days").Default(0),
		field.Int32("review_fail_count").Default(0),
		field.Int64("query_count").Default(0),
		field.String("notes").Optional().Nillable(),
		field.JSON("sentences", []entity.Sentence{}).
			Default([]entity.Sentence{}),
		field.JSON("relations", []entity.UserWordRelation{}).
			Default([]entity.UserWordRelation{}),
		field.String("created_by").Default(""),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Indexes of the Word.
func (UserWord) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "language", "word").Unique(),
		index.Fields("language", "normalized"),
	}
}

// Annotations of the UserWord.
func (UserWord) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{
			Table: "user_words",
		},
	}
}
