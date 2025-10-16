package repository

import "github.com/eslsoft/vocnet/pkg/filterexpr"

var listWordsSchema = filterexpr.ResourceSchema{
	Filter: map[string]filterexpr.FilterField{
		"keyword": {
			Kind: filterexpr.KindString,
			Ops:  map[filterexpr.Op]string{filterexpr.OpEQ: "Keyword"},
		},
		"word": {
			Kind: filterexpr.KindString,
			Ops: map[filterexpr.Op]string{
				filterexpr.OpSW: "Keyword",
				filterexpr.OpIN: "Words",
			},
		},
		"word_type": {
			Kind: filterexpr.KindString,
			Ops:  map[filterexpr.Op]string{filterexpr.OpEQ: "WordType"},
		},
	},
	Order: filterexpr.OrderSchema{
		DefaultPrimary:     "created_at",
		DefaultPrimaryDesc: true,
		FallbackKey:        "id",
		FallbackDesc:       false,
		Fields: map[string]filterexpr.OrderField{
			"created_at": {Expr: "created_at", Nulls: "last"},
			"updated_at": {Expr: "updated_at", Nulls: "last"},
			"text":       {Expr: "text", Nulls: "last"},
			"id":         {Expr: "id", Nulls: "last"},
		},
	},
}

var listLearnedLexemesSchema = filterexpr.ResourceSchema{
	Filter: map[string]filterexpr.FilterField{
		"keyword": {
			Kind: filterexpr.KindString,
			Ops:  map[filterexpr.Op]string{filterexpr.OpEQ: "Keyword"},
		},
		"lexeme": {
			Kind: filterexpr.KindString,
			Ops: map[filterexpr.Op]string{
				filterexpr.OpSW: "Keyword",
				filterexpr.OpIN: "Lexemes",
			},
		},
		"tag": {
			Kind: filterexpr.KindString,
			Ops:  map[filterexpr.Op]string{filterexpr.OpIN: "Tags"},
		},
		"category": {
			Kind: filterexpr.KindString,
			Ops:  map[filterexpr.Op]string{filterexpr.OpIN: "Categories"},
		},
	},
	Order: filterexpr.OrderSchema{
		DefaultPrimary:     "updated_at",
		DefaultPrimaryDesc: true,
		FallbackKey:        "id",
		FallbackDesc:       false,
		Fields: map[string]filterexpr.OrderField{
			"created_at":      {Expr: "created_at", Nulls: "last"},
			"updated_at":      {Expr: "updated_at", Nulls: "last"},
			"lexeme":          {Expr: "lexeme", Nulls: "last"},
			"mastery_overall": {Expr: "mastery_overall", Nulls: "last"},
			"id":              {Expr: "id", Nulls: "last"},
		},
	},
}
