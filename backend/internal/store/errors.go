package store

import "errors"

var (
	ErrClassNotFound        = errors.New("class not found")
	ErrClassNotPublished    = errors.New("class is not published")
	ErrEnrollmentNotOpenYet = errors.New("enrollment has not opened yet")
	ErrEnrollmentClosed     = errors.New("enrollment is closed")
	ErrClassFull            = errors.New("class is at capacity")
	ErrAlreadyEnrolled      = errors.New("already enrolled in this class")
)
