package mapping

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/eslsoft/vocnet/internal/entity"
)

func ToPbError(err error) error {
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

func toStatus(err error) error {
	switch err {
	case nil:
		return nil
	case entity.ErrInvalidUserWordText, entity.ErrInvalidUserID:
		return status.Error(codes.InvalidArgument, err.Error())
	case entity.ErrUserWordNotFound:
		return status.Error(codes.NotFound, err.Error())
	case entity.ErrDuplicateUserWord:
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
