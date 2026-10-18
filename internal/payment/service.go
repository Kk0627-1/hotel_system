package payment

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Confirm(req ConfirmPaymentRequest) (ConfirmPaymentResult, error) {
	return ConfirmPayment(s.repository.db, req)
}

func (s *Service) Cancel(req CancelPaymentRequest) (CancelPaymentResult, error) {
	return CancelPayment(s.repository.db, req)
}

func (s *Service) ListPending(req ListPendingRequest) (ListPendingResult, error) {
	return ListPending(s.repository.db, req)
}

func (s *Service) ListPurchasedBills(req PurchasedBillRequest) (PurchasedBillResult, error) {
	return ListPurchasedBills(s.repository.db, req)
}

func (s *Service) ListRecords(req ListRecordRequest) (ListRecordResult, error) {
	return ListRecord(s.repository.db, req)
}
