package authorization

// mock returns allowed=true for every request
type mock struct{}

var _ Authorization = &mock{}

func NewMockAuthorization() Authorization {
	return &mock{}
}

func (a mock) CheckUserValid(username string, orgId string) (bool, error) {
	return true, nil
}
