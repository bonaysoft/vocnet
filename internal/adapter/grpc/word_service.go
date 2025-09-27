package grpc

import (
	"context"

	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	dictv1 "github.com/eslsoft/vocnet/api/gen/dict/v1"
	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type WordServiceServer struct {
	dictv1.UnimplementedWordServiceServer
	uc usecase.WordUsecase
}

func NewWordServiceServer(uc usecase.WordUsecase) *WordServiceServer {
	return &WordServiceServer{uc: uc}
}

func (s *WordServiceServer) toProto(v *entity.Voc) *dictv1.Word {
	if v == nil {
		return nil
	}
	pv := &dictv1.Word{
		Id:        v.ID,
		Text:      v.Text,
		Language:  languageStringToEnum(v.Language),
		WordType:  v.VocType,
		Phonetics: []*dictv1.Phonetic{{Ipa: v.Phonetic, Dialect: "en-US"}}, // TODO: multiple phonetics
		Tags:      v.Tags,
		CreatedAt: timestamppb.New(v.CreatedAt),
	}
	if v.Lemma != nil {
		pv.Lemma = *v.Lemma
	}
	// Meanings
	for _, m := range v.Meanings {
		lang := commonv1.Language_LANGUAGE_ENGLISH
		text := m.Definition
		if m.Translation != "" {
			lang = commonv1.Language_LANGUAGE_CHINESE
			text = m.Translation
		}

		pv.Definitions = append(pv.Definitions, &dictv1.Definition{
			Pos:      m.POS,
			Text:     text,
			Language: lang,
		})
	}
	// Forms (only set for lemma entries)
	if len(v.Forms) > 0 {
		for _, f := range v.Forms {
			pv.Forms = append(pv.Forms, &dictv1.WordFormRef{Text: f.Text, WordType: f.VocType})
		}
	}
	return pv
}

func languageStringToEnum(code string) commonv1.Language {
	switch code {
	case "en":
		return commonv1.Language_LANGUAGE_ENGLISH
	case "zh":
		return commonv1.Language_LANGUAGE_CHINESE
	case "es":
		return commonv1.Language_LANGUAGE_SPANISH
	case "fr":
		return commonv1.Language_LANGUAGE_FRENCH
	case "de":
		return commonv1.Language_LANGUAGE_GERMAN
	case "ja":
		return commonv1.Language_LANGUAGE_JAPANESE
	case "ko":
		return commonv1.Language_LANGUAGE_KOREAN
	default:
		return commonv1.Language_LANGUAGE_UNSPECIFIED
	}
}

// Lookup implements exact lemma lookup
func (s *WordServiceServer) LookupWord(ctx context.Context, req *dictv1.LookupWordRequest) (*dictv1.Word, error) {
	if req == nil || req.Word == "" {
		return nil, status.Error(codes.InvalidArgument, "text required")
	}
	lang := "en"
	if req.Language != commonv1.Language_LANGUAGE_UNSPECIFIED {
		lang = req.Language.String()
	}
	v, err := s.uc.Lookup(ctx, req.Word, lang)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return s.toProto(v), nil
}
