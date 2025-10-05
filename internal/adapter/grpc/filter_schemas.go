package grpc

import "github.com/eslsoft/vocnet/internal/pkg/filterexpr"

// Schema 定义步骤说明：
// 1. 先声明一个 bindings 结构体，字段名要和 sqlc/usecase 参数结构对应。
// 2. 在 Schema.Fields 中列出允许出现在 CEL 表达式里的字段名（map 的 key）。
// 3. 为每个字段配置 FieldRule：
//    - Kind 指定字面量类型（目前支持 string/number/timestamp）。
//    - Ops 映射 “允许的操作符” -> “bindings 中的字段名”。
//      例如 OpEQ 表示 ==，OpSW 表示 startsWith，OpIN 表示 in 列表。
//    - 若需要自定义赋值，可提供 Setter；否则 binder 会按类型自动写入。
// 4. List/ListUserWords 方法中调用 BindCELTo(filter, &bindings, schema)，即可把 CEL 过滤条件
//    转成 sqlc 参数。所有不在 schema 中的字段或操作符都会被拒绝，保证行为可控。

type listWordsBindings struct {
	WordType string
	Keyword  string
	Words    []string
}

var listWordsSchema = filterexpr.Schema{
	Fields: map[string]filterexpr.FieldRule{
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
}

type listUserWordsBindings struct {
	Keyword string
	Words   []string
}

var listUserWordsSchema = filterexpr.Schema{
	Fields: map[string]filterexpr.FieldRule{
		"word": {
			Kind: filterexpr.KindString,
			Ops: map[filterexpr.Op]string{
				filterexpr.OpSW: "Keyword",
				filterexpr.OpIN: "Words",
			},
		},
		"keyword": {
			Kind: filterexpr.KindString,
			Ops:  map[filterexpr.Op]string{filterexpr.OpEQ: "Keyword"},
		},
	},
}
