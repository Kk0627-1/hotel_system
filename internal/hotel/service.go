package hotel

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(req CreateHotelRequest) (CreateHotelResult, error) {
	return CreateHotel(s.repository.db, req)
}
func (s *Service) Update(req UpdateHotelRequest) (UpdateHotelResult, error) {
	return UpdateHotel(s.repository.db, req)
}
func (s *Service) Delete(req DeleteHotelRequest) (DeleteHotelResult, error) {
	return DeleteHotel(s.repository.db, req)
}
func (s *Service) List(req ListHotelsRequest) (ListHotelsResult, error) {
	return ListHotels(s.repository.db, req)
}
