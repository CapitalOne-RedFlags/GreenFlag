package services

type ResponseService interface {
	UpdateTransaction() error
}

type GfResponseService struct {
}

func NewGfResponseService() *GfResponseService {
	return &GfResponseService{}
}

func (rs *GfResponseService) UpdateTransaction() error {
	return nil
}
