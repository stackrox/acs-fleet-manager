package services

import (
	"net/url"
	"strconv"
	
	"github.com/pkg/errors"
)

// ListArguments are arguments relevant for listing objects.
// This struct is common to all service List funcs in this package
type ListArguments struct {
	Page     int
	Size     int
}

// NewListArguments - Create ListArguments from url query parameters with sane defaults
func NewListArguments(params url.Values) *ListArguments {
	listArgs := &ListArguments{
		Page: 	1,
		Size:   100,
	}
	if v := params.Get("page"); v != "" {
		listArgs.Page, _ = strconv.Atoi(v)
	}
	if v := params.Get("size"); v != "" {
		listArgs.Size, _ = strconv.Atoi(v)
	}
	if listArgs.Size > 65500 || listArgs.Size < 0 {
		// 65500 is the maximum number of parameters that can be provided to a postgres WHERE IN clause
		// Use it as a sane max
		listArgs.Size = 65500
	}

	return listArgs
}

func (la *ListArguments) Validate() error {
	if la.Size < 1 {
		return errors.Errorf("size must be equal or greater than 1")
	}

	return nil
}
