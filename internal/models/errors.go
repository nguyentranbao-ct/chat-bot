package models

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrNotFound = status.Errorf(codes.NotFound, "not found")
