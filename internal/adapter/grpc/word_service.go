package grpc

import (
	"context"

	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	wordv1 "github.com/eslsoft/vocnet/api/gen/word/v1"
	"github.com/eslsoft/vocnet/internal/entity"
	"github.com/eslsoft/vocnet/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type WordServiceServer struct {
	wordv1.UnimplementedWordServiceServer
	uc usecase.WordUsecase
}

func NewWordServiceServer(uc usecase.WordUsecase) *WordServiceServer {
	return &WordServiceServer{uc: uc}
}

func (s *WordServiceServer) toProto(w *entity.Word) *wordv1.Word {
	if w == nil {
		return nil
	}
	return &wordv1.Word{
		Id:          w.ID,
		Lemma:       w.Lemma,
		Language:    languageStringToEnum(w.Language),
		Phonetic:    w.Phonetic,
		Pos:         w.POS,
		Definition:  w.Definition,
		Translation: w.Translation,
		Exchange:    w.Exchange,
		Tags:        w.Tags,
	}
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
func (s *WordServiceServer) Lookup(ctx context.Context, req *wordv1.LookupRequest) (*wordv1.LookupResponse, error) {
	if req == nil || req.Lemma == "" {
		return nil, status.Error(codes.InvalidArgument, "lemma required")
	}

	// Convert language enum to string name
	lang := "en"
	if req.Language != commonv1.Language_LANGUAGE_UNSPECIFIED {
		lang = req.Language.String()
	}
	w, err := s.uc.Lookup(ctx, req.Lemma, lang)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if w == nil {
		return &wordv1.LookupResponse{Found: false}, nil
	}
	return &wordv1.LookupResponse{Found: true, Word: s.toProto(w)}, nil
}
