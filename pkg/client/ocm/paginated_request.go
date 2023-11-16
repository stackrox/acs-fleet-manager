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

func fetchPages[RQ paginatedRequest[RQ, RS], RS paginatedResponse[I], I pageItem[Data], Data any](
	r RQ, pageSize int, maxPages int, f func(Data) bool) error {

	req := r.Size(pageSize)
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
