package catalog

import "errors"

var (
	ErrServiceNotFound      = errors.New("service not found in catalog")
	ErrServiceAlreadyExists = errors.New("service already exists in catalog")
)
