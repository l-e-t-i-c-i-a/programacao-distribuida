package main

import (
	"rpc/remotelist/pkg"
	"log"
	"net"
	"net/rpc"
)

func main() {
	log.Println("Iniciando servidor RemoteList...")

	// 1. Cria a instância do serviço.
	// O construtor NewRemoteList() agora cuida de carregar do disco
	// e iniciar a rotina de snapshot.
	list := remotelist.NewRemoteList()

	// 2. Configura o servidor RPC
	rpcs := rpc.NewServer()
	// Registra a instância 'list'. O RPC vai expor os métodos
	// do tipo 'RemoteList' (Append, Get, Remove, Size)
	err := rpcs.Register(list)
	if err != nil {
		log.Fatalf("Falha ao registrar o serviço RPC: %v", err)
	}

	// 3. Ouve por conexões TCP
	l, e := net.Listen("tcp", "[localhost]:5000")
	if e != nil {
		log.Fatalf("Erro ao escutar na porta :5000: %v", e)
	}
	defer l.Close()
	log.Println("Servidor escutando na porta :5000")

	// 4. Loop de aceitação de conexões
	for {
		conn, err := l.Accept()
		if err != nil {
			// Se houver erro no Accept, loga e continua (a menos que seja um erro fatal)
			log.Printf("Erro ao aceitar conexão: %v", err)
			continue
		}
		// Lança uma goroutine para lidar com cada cliente
		// Isso permite que múltiplos clientes sejam atendidos simultaneamente
		log.Printf("Nova conexão de %s", conn.RemoteAddr())
		go rpcs.ServeConn(conn)
	}
}