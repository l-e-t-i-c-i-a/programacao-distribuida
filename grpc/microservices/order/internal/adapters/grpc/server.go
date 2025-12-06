package grpc

import (
	"context"
	"fmt"
	"net"

	"log"

	"github.com/l-e-t-i-c-i-a/microservices-proto/golang/order"
	"github.com/l-e-t-i-c-i-a/microservices/order/config"
	"github.com/l-e-t-i-c-i-a/microservices/order/internal/application/core/domain"
	"github.com/l-e-t-i-c-i-a/microservices/order/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func (a Adapter) Create (ctx context.Context, request *order.CreateOrderRequest) (*order.CreateOrderResponse, error) {
	// 1. Tradução (Mapping): De Proto para Domínio
	var orderItems []domain.OrderItem
	for _, orderItem := range request.OrderItems {
		// Copiando dados do formato gRPC para o formato interno
		orderItems = append(orderItems, domain.OrderItem {
			ProductCode: orderItem.ProductCode,
			UnitPrice: orderItem.UnitPrice,
			Quantity: orderItem.Quantity,
		})
	}

	// 2. Chamada ao Core (Aplicação)
    // Conversão de tipos: int64(request.CostumerId)
	newOrder := domain.NewOrder(int64(request.CostumerId), orderItems)
	result, err := a.api.PlaceOrder(newOrder)
	if err != nil {
		return nil, err
	}

	// 3. Tradução de Volta: De Domínio para Proto
    // Uso do & para retornar o endereço da resposta
	return &order.CreateOrderResponse{OrderId: int32(result.ID)}, nil
}

type Adapter struct {
	api ports.APIPort
	port int
	order.UnimplementedOrderServer
}

func NewAdapter(api ports.APIPort, port int) *Adapter {
	return &Adapter{api: api, port: port}
}

func (a Adapter) Run() {
	var err error
	// Abre a porta TCP
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		log.Fatalf("failed to listen on port %d, error: %v", a.port, err)
	}

	// Cria o motor do gRPC
	grpcServer := grpc.NewServer()

	// "Cola" a implementação (a) no motor do gRPC
	order.RegisterOrderServer(grpcServer, a)

	// Reflection (Útil para Desenvolvimento)
	if config.GetEnv() == "development" {
		reflection.Register(grpcServer)
	}
	if err := grpcServer.Serve(listen); err != nil {
		log.Fatalf("failed to serve grpc on port ")
	}
}

