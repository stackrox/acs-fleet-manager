package authorization

// mock returns allowed=true for every request
type mock struct{}

var _ Authorization = &mock{}

// NewMockAuthorization ...
func NewMockAuthorization() Authorization {
	return &mock{}
}

// CheckUserValid ...
func (a mock) CheckUserValid(username string, orgId string) (bool, error) {
	return true, nil
}
