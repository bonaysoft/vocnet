package grpc

import (
	"context"

	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	vocv1 "github.com/eslsoft/vocnet/api/gen/voc/v1"
	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type VocServiceServer struct {
	vocv1.UnimplementedVocServiceServer
	uc usecase.WordUsecase
}

func NewVocServiceServer(uc usecase.WordUsecase) *VocServiceServer {
	return &VocServiceServer{uc: uc}
}

func (s *VocServiceServer) toProto(v *entity.Voc) *vocv1.Voc {
	if v == nil {
		return nil
	}
	pv := &vocv1.Voc{
		Id:       v.ID,
		Text:     v.Text,
		Language: languageStringToEnum(v.Language),
		VocType:  v.VocType,
		Phonetic: v.Phonetic,
		Tags:     v.Tags,
	}
	if v.Lemma != nil {
		pv.Lemma = *v.Lemma
	}
	// Meanings
	for _, m := range v.Meanings {
		pv.Meanings = append(pv.Meanings, &vocv1.VocMeaning{
			Pos:         m.POS,
			Definition:  m.Definition,
			Translation: m.Translation,
		})
	}
	// Forms (only set for lemma entries)
	if len(v.Forms) > 0 {
		for _, f := range v.Forms {
			pv.Forms = append(pv.Forms, &vocv1.VocFormRef{Text: f.Text, VocType: f.VocType})
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
func (s *VocServiceServer) Lookup(ctx context.Context, req *vocv1.LookupVocRequest) (*vocv1.Voc, error) {
	if req == nil || req.Text == "" {
		return nil, status.Error(codes.InvalidArgument, "text required")
	}
	lang := "en"
	if req.Language != commonv1.Language_LANGUAGE_UNSPECIFIED {
		lang = req.Language.String()
	}
	v, err := s.uc.Lookup(ctx, req.Text, lang)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return s.toProto(v), nil
}
