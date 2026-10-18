package wallet

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) Recharge(req RechargeRequest) (RechargeResponse, error) {
	return Recharge(s.repository.db, req)
}
