package api

import (
	"github.com/l-e-t-i-c-i-a/microservices/order/internal/application/core/domain"
	"github.com/l-e-t-i-c-i-a/microservices/order/internal/ports"
)

type Application struct {
	db ports.DBPort
}

func NewApplication(db ports.DBPort) *Application {
	return &Application{
		db:db,
	}
}

func (a Application) PlaceOrder(order domain.Order) (domain.Order, error) {
	// 1. Chama a porta do banco de dados para salvar
	err := a.db.Save(&order)
	// 2. Verifica se houve erro
	if err != nil {
		return domain.Order{}, err
	}
	// 3. Retorna o pedido salvo
	return order, nil
}