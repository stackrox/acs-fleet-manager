package auth

// AuthAgentService ...
//
//go:generate moq -out auth_agent_service_moq.go . AuthAgentService
type AuthAgentService interface {
	GetClientId(clusterID string) (string, error)
}
