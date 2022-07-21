package services

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
)

// GetAcceptedOrderByParams ...
func GetAcceptedOrderByParams() []string {
	return []string{"host", "cloud_provider", "cluster_id", "created_at", "href", "id", "instance_type", "multi_az", "name", "organisation_id", "owner", "region", "status", "updated_at", "version"}
}

// ListArguments are arguments relevant for listing objects.
// This struct is common to all service List funcs in this package
type ListArguments struct {
	Page     int
	Size     int
	Preloads []string
	Search   string
	OrderBy  []string
}

// NewListArguments - Create ListArguments from url query parameters with sane defaults
func NewListArguments(params url.Values) *ListArguments {
	listArgs := &ListArguments{
		Page:   1,
		Size:   100,
		Search: "",
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
	if v := params.Get("search"); v != "" {
		listArgs.Search = v
	}
	if v := params.Get("orderBy"); v != "" {
		listArgs.OrderBy = strings.Split(v, ",")
		// remove spaces
		for i, s := range listArgs.OrderBy {
			listArgs.OrderBy[i] = strings.Trim(s, " ")
		}
	}
	return listArgs
}

// Validate ...
func (la *ListArguments) Validate() error {
	if la.Page < 0 {
		return errors.Errorf("page must be equal or greater than 0")
	}
	if la.Size < 1 {
		return errors.Errorf("size must be equal or greater than 1")
	}

	if len(la.OrderBy) > 0 {
		space := regexp.MustCompile(`\s+`)
		for _, orderByClause := range la.OrderBy {
			field := strings.ToLower(orderByClause)
			// remove multiple spaces
			field = space.ReplaceAllString(field, " ")
			keywords := strings.Split(field, " ") // this could contain only the column name or the column name and the direction (asc/desc)

			if len(keywords) > 2 {
				return errors.Errorf("invalid order by clause '%s'", orderByClause)
			}

			if !shared.Contains(GetAcceptedOrderByParams(), keywords[0]) {
				return errors.Errorf("unknown order by field '%s'", keywords[0])
			}

			if len(keywords) == 2 {
				if keywords[1] != "asc" && keywords[1] != "desc" {
					return errors.Errorf("invalid order by direction '%s'", keywords[1])
				}
			}

		}
	}

	return nil
}
