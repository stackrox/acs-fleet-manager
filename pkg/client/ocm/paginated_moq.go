package ocm

type paginatedResponseMoqImpl[T any] struct {
	data  []T
	total int
	page  int
	size  int
}

var _ paginatedResponse[*paginatedResponseMoqImpl[any]] = (*paginatedResponseMoqImpl[any])(nil)

func (d *paginatedResponseMoqImpl[T]) Size() int {
	return len(d.data)
}

func (d *paginatedResponseMoqImpl[T]) Items() *paginatedResponseMoqImpl[T] {
	return d
}

func (d *paginatedResponseMoqImpl[T]) Each(f func(T) bool) {
	for _, v := range d.data {
		if !f(v) {
			break
		}
	}
}

func (d *paginatedResponseMoqImpl[T]) getPage(page int) paginatedResponseMoqImpl[T] {
	first := page * d.size
	last := first + d.size
	if last > d.total {
		last = d.total
	}
	return paginatedResponseMoqImpl[T]{
		data:  d.data[first:last],
		total: d.total,
		size:  last - first,
		page:  page,
	}
}

type dataPagerMoqImpl[T any] struct {
	data paginatedResponseMoqImpl[T]
}

func NewMoqPager[T any](data []T) dataPagerMoqImpl[T] {
	return dataPagerMoqImpl[T]{
		data: paginatedResponseMoqImpl[T]{
			data:  data,
			total: len(data),
		},
	}
}

var _ paginatedRequest[*dataPagerMoqImpl[any], *paginatedResponseMoqImpl[any]] = (*dataPagerMoqImpl[any])(nil)

func (p *dataPagerMoqImpl[T]) Size(size int) *dataPagerMoqImpl[T] {
	newData := p.data
	newData.size = size
	return &dataPagerMoqImpl[T]{data: newData}
}

func (p *dataPagerMoqImpl[T]) Page(page int) *dataPagerMoqImpl[T] {
	return &dataPagerMoqImpl[T]{
		data: p.data.getPage(page - 1),
	}
}

func (p *dataPagerMoqImpl[T]) Send() (*paginatedResponseMoqImpl[T], error) {
	return &p.data, nil
}
