package services


// A CentralService exposes methods to retrieve, manipulate and store Central requests.
//go:generate moq -out centralservice_moq.go . CentralService
type CentralService interface {
	// TODO following internal/dinosaur/internal/services/dinosaur.go
}

var _ CentralService = &centralService{}

type centralService struct {}

func NewCentralService() *centralService {
	return &centralService{}
}