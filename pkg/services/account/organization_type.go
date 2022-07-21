package account

import "time"

// Organization ...
type Organization struct {
	ID            string
	Name          string
	AccountNumber string
	ExternalID    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// OrganizationList ...
type OrganizationList struct {
	items []*Organization
}

// Len returns the length of the list.
func (l *OrganizationList) Len() int {
	if l == nil {
		return 0
	}
	return len(l.items)
}

// Empty returns true if the list is empty.
func (l *OrganizationList) Empty() bool {
	return l == nil || len(l.items) == 0
}

// Get returns the item of the list with the given index. If there is no item with
// that index it returns nil.
func (l *OrganizationList) Get(i int) *Organization {
	if l == nil || i < 0 || i >= len(l.items) {
		return nil
	}
	return l.items[i]
}

// Slice returns an slice containing the items of the list. The returned slice is a
// copy of the one used internally, so it can be modified without affecting the
// internal representation.
//
// If you don't need to modify the returned slice consider using the Each or Range
// functions, as they don't need to allocate a new slice.
func (l *OrganizationList) Slice() []*Organization {
	var slice []*Organization
	if l == nil {
		slice = make([]*Organization, 0)
	} else {
		slice = make([]*Organization, len(l.items))
		copy(slice, l.items)
	}
	return slice
}

// Each runs the given function for each item of the list, in order. If the function
// returns false the iteration stops, otherwise it continues till all the elements
// of the list have been processed.
func (l *OrganizationList) Each(f func(item *Organization) bool) {
	if l == nil {
		return
	}
	for _, item := range l.items {
		if !f(item) {
			break
		}
	}
}

// Range runs the given function for each index and item of the list, in order. If
// the function returns false the iteration stops, otherwise it continues till all
// the elements of the list have been processed.
func (l *OrganizationList) Range(f func(index int, item *Organization) bool) {
	if l == nil {
		return
	}
	for index, item := range l.items {
		if !f(index, item) {
			break
		}
	}
}
