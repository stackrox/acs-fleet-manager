package ocm

import pkgerrors "github.com/pkg/errors"

type pageItem[Data any] interface {
	Each(func(value Data) bool)
}

type paginatedResponse[Items any] interface {
	Size() int
	Items() Items
}

type paginatedRequest[RequestType any, ResponseType any] interface {
	Size(int) RequestType
	Page(int) RequestType
	Send() (ResponseType, error)
}

// fetchPages calls OCM API with size and page request parameters, allowing for
// paged access to the data. Example request type: *amsv1.QuotaCostListRequest,
// with *amsv1.QuotaCost as the data in the response list.
// Iteration stops when all or maxPages pages are retrieved, or f returns false.
func fetchPages[RQ paginatedRequest[RQ, RS], RS paginatedResponse[I], I pageItem[Data], Data any](
	request paginatedRequest[RQ, RS], pageSize int, maxPages int, f func(Data) bool) error {

	req := request.Size(pageSize)
	complete := false
	page := 1
	for ; !complete && page <= maxPages; page++ {
		response, err := req.Page(page).Send()
		if err != nil {
			return pkgerrors.Wrapf(err, "error retrieving page %d", page)
		}
		response.Items().Each(func(data Data) bool {
			complete = !f(data)
			return !complete
		})
		complete = complete || response.Size() < pageSize
	}
	if page > maxPages && !complete {
		return pkgerrors.New("too many pages")
	}
	return nil
}
