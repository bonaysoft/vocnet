package grpc

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"
	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	dictv1 "github.com/eslsoft/vocnet/api/gen/dict/v1"
	"github.com/eslsoft/vocnet/api/gen/dict/v1/dictv1connect"
	vocnetv1 "github.com/eslsoft/vocnet/api/gen/vocnet/v1"
	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/pkg/filterexpr"
	"github.com/eslsoft/vocnet/internal/usecase"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type WordServiceServer struct {
	dictv1connect.UnimplementedWordServiceHandler
	uc usecase.WordUsecase
}

func NewWordServiceServer(uc usecase.WordUsecase) *WordServiceServer {
	return &WordServiceServer{uc: uc}
}

func (s *WordServiceServer) CreateWord(ctx context.Context, req *connect.Request[dictv1.CreateWordRequest]) (*connect.Response[dictv1.Word], error) {
	if req.Msg == nil || req.Msg.Word == nil {
		return nil, status.Error(codes.InvalidArgument, "word payload required")
	}
	entityWord := fromProtoWord(req.Msg.Word)
	result, err := s.uc.Create(ctx, entityWord)
	if err != nil {
		return nil, mapWordError(err)
	}
	return connect.NewResponse(s.toProto(result)), nil
}

func (s *WordServiceServer) UpdateWord(ctx context.Context, req *connect.Request[dictv1.Word]) (*connect.Response[dictv1.Word], error) {
	if req.Msg == nil {
		return nil, status.Error(codes.InvalidArgument, "word payload required")
	}
	entityWord := fromProtoWord(req.Msg)
	result, err := s.uc.Update(ctx, entityWord)
	if err != nil {
		return nil, mapWordError(err)
	}
	return connect.NewResponse(s.toProto(result)), nil
}

func (s *WordServiceServer) GetWord(ctx context.Context, req *connect.Request[commonv1.IDRequest]) (*connect.Response[dictv1.Word], error) {
	if req.Msg == nil {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	result, err := s.uc.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, mapWordError(err)
	}
	return connect.NewResponse(s.toProto(result)), nil
}

func (s *WordServiceServer) ListWords(ctx context.Context, req *connect.Request[dictv1.ListWordsRequest]) (*connect.Response[dictv1.ListWordsResponse], error) {
	if req.Msg == nil {
		return nil, status.Error(codes.InvalidArgument, "request required")
	}
	msg := req.Msg
	bindings := listWordsBindings{}
	if err := filterexpr.BindCELTo(msg.GetFilter(), &bindings, listWordsSchema); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	filter := entity.WordFilter{
		Pagination: convertPagination(msg.GetPagination()),
		Keyword:    strings.TrimSpace(bindings.Keyword),
		WordType:   strings.TrimSpace(bindings.WordType),
		Words:      bindings.Words,
	}

	items, total, err := s.uc.List(ctx, filter)
	if err != nil {
		return nil, mapWordError(err)
	}

	return connect.NewResponse(&dictv1.ListWordsResponse{
		Words: lo.Map(items, func(item *entity.Word, _ int) *dictv1.Word {
			return s.toProto(item)
		}),
		Pagination: &commonv1.PaginationResponse{
			Total:  int32(total),
			PageNo: filter.PageNo,
		},
	}), nil
}

func (s *WordServiceServer) DeleteWord(ctx context.Context, req *connect.Request[commonv1.IDRequest]) (*connect.Response[emptypb.Empty], error) {
	if req.Msg == nil {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}
	if err := s.uc.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, mapWordError(err)
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// LookupWord looks up a word by text and language.
func (s *WordServiceServer) LookupWord(ctx context.Context, req *connect.Request[dictv1.LookupWordRequest]) (*connect.Response[dictv1.Word], error) {
	if req.Msg == nil || req.Msg.Word == "" {
		return nil, status.Error(codes.InvalidArgument, "text required")
	}
	lang := fromProtoLanguage(req.Msg.GetLanguage())
	if lang == entity.LanguageUnspecified {
		lang = entity.LanguageEnglish
	}
	v, err := s.uc.Lookup(ctx, req.Msg.Word, lang)
	if err != nil {
		return nil, mapWordError(err)
	}

	return connect.NewResponse(s.toProto(v)), nil
}

func (s *WordServiceServer) toProto(v *entity.Word) *dictv1.Word {
	if v == nil {
		return nil
	}
	pv := &dictv1.Word{
		Id:       v.ID,
		Text:     v.Text,
		Language: toProtoLanguage(v.Language),
		WordType: v.WordType,
		Phonetics: lo.Map(v.Phonetics, func(p entity.WordPhonetic, _ int) *dictv1.Phonetic {
			return &dictv1.Phonetic{Ipa: p.IPA, Dialect: p.Dialect}
		}),
		Definitions: lo.Map(v.Definitions, func(meaning entity.WordDefinition, _ int) *dictv1.Definition {
			lang := toProtoLanguage(meaning.Language)
			if lang == commonv1.Language_LANGUAGE_UNSPECIFIED {
				lang = commonv1.Language_LANGUAGE_ENGLISH
			}
			return &dictv1.Definition{
				Pos:      meaning.Pos,
				Text:     meaning.Text,
				Language: lang,
			}
		}),
		Forms: lo.Map(v.Forms, func(form entity.WordFormRef, _ int) *dictv1.WordFormRef {
			return &dictv1.WordFormRef{Text: form.Text, WordType: form.WordType}
		}),
		Tags:    v.Tags,
		Phrases: lo.Map(v.Phrases, func(phrase string, _ int) string { return phrase }),
		Sentences: lo.Map(v.Sentences, func(sent entity.Sentence, _ int) *vocnetv1.Sentence {
			return &vocnetv1.Sentence{Text: sent.Text, Source: vocnetv1.SourceType(sent.Source), SourceRef: sent.SourceRef}
		}),
		Relations: lo.Map(v.Relations, func(rel entity.WordRelation, _ int) *dictv1.WordRelation {
			return &dictv1.WordRelation{Word: rel.Word, RelationType: commonv1.RelationType(rel.RelationType)}
		}),
	}

	if v.Lemma != nil {
		pv.Lemma = *v.Lemma
	}

	if !v.CreatedAt.IsZero() {
		pv.CreatedAt = timestamppb.New(v.CreatedAt)
	}
	if !v.UpdatedAt.IsZero() {
		pv.UpdatedAt = timestamppb.New(v.UpdatedAt)
	}

	return pv
}

func fromProtoWord(in *dictv1.Word) *entity.Word {
	if in == nil {
		return nil
	}
	word := &entity.Word{
		ID:       in.GetId(),
		Text:     strings.TrimSpace(in.GetText()),
		Language: fromProtoLanguage(in.GetLanguage()),
		WordType: strings.TrimSpace(in.GetWordType()),
		Tags:     append([]string(nil), in.GetTags()...),
		Phrases:  append([]string(nil), in.GetPhrases()...),
	}
	if lemma := strings.TrimSpace(in.GetLemma()); lemma != "" {
		word.Lemma = &lemma
	}
	if phonetics := in.GetPhonetics(); len(phonetics) > 0 {
		word.Phonetics = lo.FilterMap(phonetics, func(p *dictv1.Phonetic, _ int) (entity.WordPhonetic, bool) {
			if p == nil {
				return entity.WordPhonetic{}, false
			}
			ipa := strings.TrimSpace(p.GetIpa())
			if ipa == "" {
				return entity.WordPhonetic{}, false
			}
			return entity.WordPhonetic{
				IPA:     ipa,
				Dialect: strings.TrimSpace(p.GetDialect()),
			}, true
		})
		if len(word.Phonetics) == 0 {
			word.Phonetics = nil
		}
	}
	if ts := in.GetCreatedAt(); ts != nil {
		word.CreatedAt = ts.AsTime()
	}
	if ts := in.GetUpdatedAt(); ts != nil {
		word.UpdatedAt = ts.AsTime()
	}
	definitions := lo.FilterMap(in.GetDefinitions(), func(def *dictv1.Definition, _ int) (entity.WordDefinition, bool) {
		if def == nil {
			return entity.WordDefinition{}, false
		}
		text := strings.TrimSpace(def.GetText())
		if text == "" {
			return entity.WordDefinition{}, false
		}
		return entity.WordDefinition{
			Pos:      strings.TrimSpace(def.GetPos()),
			Text:     text,
			Language: fromProtoLanguage(def.GetLanguage()),
		}, true
	})
	if len(definitions) > 0 {
		word.Definitions = definitions
	}
	forms := lo.FilterMap(in.GetForms(), func(form *dictv1.WordFormRef, _ int) (entity.WordFormRef, bool) {
		if form == nil {
			return entity.WordFormRef{}, false
		}
		text := strings.TrimSpace(form.GetText())
		if text == "" {
			return entity.WordFormRef{}, false
		}
		return entity.WordFormRef{
			Text:     text,
			WordType: strings.TrimSpace(form.GetWordType()),
		}, true
	})
	if len(forms) > 0 {
		word.Forms = forms
	}
	phrases := lo.FilterMap(in.GetPhrases(), func(phrase string, _ int) (string, bool) {
		trimmed := strings.TrimSpace(phrase)
		if trimmed == "" {
			return "", false
		}
		return trimmed, true
	})
	if len(phrases) > 0 {
		word.Phrases = phrases
	}
	sentences := lo.FilterMap(in.GetSentences(), func(sent *vocnetv1.Sentence, _ int) (entity.Sentence, bool) {
		if sent == nil {
			return entity.Sentence{}, false
		}
		text := strings.TrimSpace(sent.GetText())
		if text == "" {
			return entity.Sentence{}, false
		}
		return entity.Sentence{
			Text:      text,
			Source:    int32(sent.GetSource()),
			SourceRef: strings.TrimSpace(sent.GetSourceRef()),
		}, true
	})
	if len(sentences) > 0 {
		word.Sentences = sentences
	}
	relations := lo.FilterMap(in.GetRelations(), func(rel *dictv1.WordRelation, _ int) (entity.WordRelation, bool) {
		if rel == nil {
			return entity.WordRelation{}, false
		}
		wordText := strings.TrimSpace(rel.GetWord())
		if wordText == "" {
			return entity.WordRelation{}, false
		}
		return entity.WordRelation{Word: wordText, RelationType: int32(rel.GetRelationType())}, true
	})
	if len(relations) > 0 {
		word.Relations = relations
	}
	return word
}

func mapWordError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, entity.ErrInvalidVocText), errors.Is(err, entity.ErrInvalidVocID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, entity.ErrVocNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, entity.ErrDuplicateWord):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func makePageNo(limit, offset int32) int32 {
	if limit <= 0 {
		return 1
	}
	if offset < 0 {
		offset = 0
	}
	return offset/limit + 1
}

func toProtoLanguage(lang entity.Language) commonv1.Language {
	switch lang {
	case entity.LanguageEnglish:
		return commonv1.Language_LANGUAGE_ENGLISH
	case entity.LanguageChinese:
		return commonv1.Language_LANGUAGE_CHINESE
	case entity.LanguageSpanish:
		return commonv1.Language_LANGUAGE_SPANISH
	case entity.LanguageFrench:
		return commonv1.Language_LANGUAGE_FRENCH
	case entity.LanguageGerman:
		return commonv1.Language_LANGUAGE_GERMAN
	case entity.LanguageJapanese:
		return commonv1.Language_LANGUAGE_JAPANESE
	case entity.LanguageKorean:
		return commonv1.Language_LANGUAGE_KOREAN
	case entity.LanguageUnspecified:
		fallthrough
	default:
		return commonv1.Language_LANGUAGE_UNSPECIFIED
	}
}

func fromProtoLanguage(lang commonv1.Language) entity.Language {
	switch lang {
	case commonv1.Language_LANGUAGE_ENGLISH:
		return entity.LanguageEnglish
	case commonv1.Language_LANGUAGE_CHINESE:
		return entity.LanguageChinese
	case commonv1.Language_LANGUAGE_SPANISH:
		return entity.LanguageSpanish
	case commonv1.Language_LANGUAGE_FRENCH:
		return entity.LanguageFrench
	case commonv1.Language_LANGUAGE_GERMAN:
		return entity.LanguageGerman
	case commonv1.Language_LANGUAGE_JAPANESE:
		return entity.LanguageJapanese
	case commonv1.Language_LANGUAGE_KOREAN:
		return entity.LanguageKorean
	case commonv1.Language_LANGUAGE_UNSPECIFIED:
		fallthrough
	default:
		return entity.LanguageUnspecified
	}
}
