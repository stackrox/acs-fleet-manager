package impl

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

// fetchPages sets the requested size of a page, and fetches pages until all
// data is fetched, or f, called on every element, returns false, or maxPages
// pages have been fetched.
// In the latter case, if the last retrieved page contains pageSize elements, an
// error is returned, indicating that there could potentially be more pages to
// fetch.
func fetchPages[RQ paginatedRequest[RQ, RS], RS paginatedResponse[I], I pageItem[Data], Data any](
	request paginatedRequest[RQ, RS], pageSize int, maxPages int, f func(Data) bool) error {

	req := request.Size(pageSize)
	for page := 1; page <= maxPages; page++ {
		response, err := req.Page(page).Send()
		if err != nil {
			return pkgerrors.Wrapf(err, "error retrieving page %d", page)
		}
		keepGoing := true
		response.Items().Each(func(data Data) bool {
			keepGoing = f(data)
			return keepGoing
		})
		if !keepGoing || response.Size() < pageSize {
			break
		}
		if page == maxPages {
			return pkgerrors.New("too many pages")
		}
	}
	return nil
}
