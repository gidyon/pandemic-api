package services

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NilRequestError return a nil request error
func NilRequestError(request string) error {
	return status.Errorf(codes.InvalidArgument, "%s must not be nil", request)
}

// MissingFieldError return missing field error
func MissingFieldError(field string) error {
	return status.Errorf(codes.InvalidArgument, "missing %s", field)
}

// FailedToBeginTx is error from failed start transaction
func FailedToBeginTx(err error) error {
	return status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
}

// FailedToCommitTx is error from failed commit
func FailedToCommitTx(err error) error {
	return status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
}

const defaultPageSize = 10

// NormalizePage parses page token and page size and reset weird values
func NormalizePage(pageToken, pageSize int32) (int, int) {
	if pageToken <= 0 {
		pageToken = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > 20 {
		pageSize = 20
	}
	return int(pageToken), int(pageSize)
}
