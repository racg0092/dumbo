package dumbo

import "errors"

var (
	ErrSessionExpired  = errors.New("session expired")
	ErrValIsNil        = errors.New("val is <nil>")
	ErrValIsNotPointer = errors.New("val is now a pointer")
	ErrCantSetValue    = errors.New("can't set val")
	ErrNoTypeMatch     = errors.New("key content and value type do not match")
)
