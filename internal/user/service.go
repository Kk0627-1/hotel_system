package user

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) Register(req RegisterRequest) (RegisterResult, error) {
	return RegisterUser(s.repository.db, req)
}
func (s *Service) Login(req LoginRequest) (LoginResult, error) {
	return LoginUser(s.repository.db, req)
}
