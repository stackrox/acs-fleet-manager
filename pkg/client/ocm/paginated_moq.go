package ocm

// Here goes implementation of paginatedResponse and paginatedRequest interfaces
// for testing purposes. They provide paged access to an array of testDataType.

type testDataType int

// paginatedResponseMoqImpl implements paginatedResponse with Size() and Items()
// methods working over array of elements, and also pageItem interface by
// implementing Each(f), just to not produce one more implementation.
type paginatedResponseMoqImpl struct {
	data  []testDataType
	total int
	page  int
	size  int
}

var _ paginatedResponse[*paginatedResponseMoqImpl] = (*paginatedResponseMoqImpl)(nil)

func (d *paginatedResponseMoqImpl) Size() int {
	return len(d.data)
}

func (d *paginatedResponseMoqImpl) Items() *paginatedResponseMoqImpl {
	return d
}

// Each calls f on all items. Implements pageItem interface.
func (d *paginatedResponseMoqImpl) Each(f func(testDataType) bool) {
	for _, v := range d.data {
		if !f(v) {
			break
		}
	}
}

func (d *paginatedResponseMoqImpl) getPage(page int) paginatedResponseMoqImpl {
	first := page * d.size
	last := first + d.size
	if last > d.total {
		last = d.total
	}
	return paginatedResponseMoqImpl{
		data:  d.data[first:last],
		total: d.total,
		size:  last - first,
		page:  page,
	}
}

type paginatedRequestMoqImpl struct {
	data paginatedResponseMoqImpl
}

// NewMoqPager builds and returns a test implementation of paginatedRequest,
// working over provided data.
func NewMoqPager(data []testDataType) paginatedRequest[*paginatedRequestMoqImpl, *paginatedResponseMoqImpl] {
	return &paginatedRequestMoqImpl{
		data: paginatedResponseMoqImpl{
			data:  data,
			total: len(data),
		},
	}
}

var _ paginatedRequest[*paginatedRequestMoqImpl, *paginatedResponseMoqImpl] = (*paginatedRequestMoqImpl)(nil)

// Size sets the maximum number of elements in a page.
func (p *paginatedRequestMoqImpl) Size(size int) *paginatedRequestMoqImpl {
	newData := p.data
	newData.size = size
	return &paginatedRequestMoqImpl{data: newData}
}

// Page returns data on given page, counting from 1.
func (p *paginatedRequestMoqImpl) Page(page int) *paginatedRequestMoqImpl {
	return &paginatedRequestMoqImpl{
		data: p.data.getPage(page - 1),
	}
}

// Send returns the response.
func (p *paginatedRequestMoqImpl) Send() (*paginatedResponseMoqImpl, error) {
	return &p.data, nil
}
