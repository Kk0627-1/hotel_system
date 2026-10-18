package room

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) Create(req CreateRoomRequest) (CreateRoomResult, error) {
	return CreateRoom(s.repository.db, req)
}
func (s *Service) Update(req UpdateRoomRequest) (UpdateRoomResult, error) {
	return UpdateRoom(s.repository.db, req)
}
func (s *Service) Delete(req DeleteRoomRequest) (DeleteRoomResult, error) {
	return DeleteRoom(s.repository.db, req)
}
func (s *Service) List(req ListRoomsRequest) (ListRoomsResult, error) {
	return ListRooms(s.repository.db, req)
}
