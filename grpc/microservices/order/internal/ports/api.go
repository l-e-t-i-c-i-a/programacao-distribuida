package ports

import "github.com/l-e-t-i-c-i-a/microservices/order/internal/application/core/domain"

type APIPort interface {
	PlaceOrder(order domain.Order) (domain.Order, error)
}