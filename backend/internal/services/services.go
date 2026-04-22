package services

// Services expõe casos de uso da aplicação (camada application/service).
type Services struct {
	Dependencies
}

// New constrói o container de serviços.
func New(deps Dependencies) *Services {
	return &Services{Dependencies: deps}
}

// Close libera recursos de infraestrutura.
func (s *Services) Close() error {
	if s.DB != nil {
		s.DB.Close()
	}
	if s.Redis != nil {
		return s.Redis.Close()
	}
	return nil
}
