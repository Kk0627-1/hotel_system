package reservation

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(req CreateReservationRequest) (CreateReservationResult, error) {
	return CreateReservation(s.repository.db, req)
}
