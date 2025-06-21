package hdl

import "errors"

var ErrInternal = errors.New("internal error")
var ErrDecodeRequest = errors.New("decode request")
var ErrFileTooLarge = errors.New("file too large")

var ErrToRetrievePathArg = errors.New("error to retrieve path argument")
var ErrFailedToGetUUID = errors.New("failed to get uid from context")
var ErrFailedToParseUUID = errors.New("failed to parse uid")
