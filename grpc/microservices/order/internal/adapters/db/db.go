package db

import (
	"fmt"

	"github.com/l-e-t-i-c-i-a/microservices/order/internal/application/core/domain"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Order struct {
	gorm.Model
	CustomerID int64
	Status string
	OrderItems []OrderItem
}

type OrderItem struct {
	gorm.Model
	ProductCode string
	UnitPrice float32
	Quantity int32
	OrderID uint
}

type Adapter struct {
	db *gorm.DB
}

func NewAdapter(dataSourceUrl string) (*Adapter, error) {
	db, openErr := gorm.Open(mysql.Open(dataSourceUrl), &gorm.Config{})
	if openErr != nil {
		return nil, fmt.Errorf("db connection error: %v", openErr)
	}
	err := db.AutoMigrate(&Order{}, OrderItem{})
	if err != nil {
		return nil, fmt.Errorf("db migration error: %v", err)
	}
	return &Adapter{db: db}, nil
}

func (a Adapter) Get(id string) (domain.Order, error) {
	var orderEntity Order

	// Busca o primeiro registro que bata com o ID e preenche &orderEntity
	res := a.db.First(&orderEntity, id)
	var orderItems []domain.OrderItem
	for _, orderItem := range orderEntity.OrderItems {
		orderItems = append(orderItems, domain.OrderItem{
			ProductCode: orderItem.ProductCode,
			UnitPrice: orderItem.UnitPrice,
			Quantity: orderItem.Quantity,
		})
	}

	// Cria o objeto do DOMÍNIO para retornar
	order := domain.Order{
		ID: int64(orderEntity.ID),
		CustomerID: orderEntity.CustomerID,
		Status: orderEntity.Status,
		OrderItems: orderItems,
		CreatedAt: orderEntity.CreatedAt.UnixNano(),
	}
	return order, res.Error
}

func (a Adapter) Save(order *domain.Order) error {
	var orderItems []OrderItem // Slice de itens do BANCO

	for _, orderItem := range order.OrderItems {
		orderItems = append(orderItems, OrderItem{
			ProductCode: orderItem.ProductCode,
			UnitPrice: orderItem.UnitPrice,
			Quantity: orderItem.Quantity,
		})
	}
	orderModel := Order{
		CustomerID: order.CustomerID,
		Status: order.Status,
		OrderItems: orderItems,
	}

	// Salva no banco (INSERT INTO orders ...)
    // Passa &orderModel porque o GORM precisa preencher o ID gerado lá dentro
	res := a.db.Create(&orderModel)
	if res.Error == nil {
		// Pega o ID gerado pelo banco e atualiza o objeto de domínio original
		order.ID = int64(orderModel.ID)
	}
	return res.Error
}