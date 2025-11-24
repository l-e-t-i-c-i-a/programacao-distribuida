package main

import (
	"fmt"
	"remotelist/pkg" // Importa o pacote com as structs Args/Reply
	"log"
	"net/rpc"
	"time"
)

func main() {
	client, err := rpc.Dial("tcp", ":5000")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer client.Close()

	// --- Testando Lista "minhaLista" ---
	fmt.Println("--- Testando 'minhaLista' ---")

	// 1. Append
	var appReply remotelist.AppendReply
	err = client.Call("RemoteList.Append", remotelist.AppendArgs{ListID: "minhaLista", Value: 10}, &appReply)
	err = client.Call("RemoteList.Append", remotelist.AppendArgs{ListID: "minhaLista", Value: 20}, &appReply)
	err = client.Call("RemoteList.Append", remotelist.AppendArgs{ListID: "minhaLista", Value: 30}, &appReply)
	if err != nil {
		log.Fatal("Erro no Append:", err)
	}
	fmt.Println("Append(10, 20, 30) para 'minhaLista'. Sucesso:", appReply.Success)

	// 2. Size
	var sizeReply remotelist.SizeReply
	err = client.Call("RemoteList.Size", remotelist.SizeArgs{ListID: "minhaLista"}, &sizeReply)
	if err != nil {
		log.Fatal("Erro no Size:", err)
	}
	fmt.Println("Size('minhaLista'):", sizeReply.Size) // Esperado: 3

	// 3. Get
	var getReply remotelist.GetReply
	err = client.Call("RemoteList.Get", remotelist.GetArgs{ListID: "minhaLista", Index: 1}, &getReply)
	if err != nil {
		log.Fatal("Erro no Get:", err)
	}
	fmt.Println("Get('minhaLista', 1):", getReply.Value) // Esperado: 20

	// 4. Remove
	var remReply remotelist.RemoveReply
	err = client.Call("RemoteList.Remove", remotelist.RemoveArgs{ListID: "minhaLista"}, &remReply)
	if err != nil {
		log.Fatal("Erro no Remove:", err)
	}
	fmt.Println("Remove('minhaLista'):", remReply.Value) // Esperado: 30

	// 5. Size (de novo)
	err = client.Call("RemoteList.Size", remotelist.SizeArgs{ListID: "minhaLista"}, &sizeReply)
	fmt.Println("Size('minhaLista') após Remove:", sizeReply.Size) // Esperado: 2

	// --- Testando Lista "outraLista" (para provar concorrência) ---
	fmt.Println("\n--- Testando 'outraLista' ---")
	err = client.Call("RemoteList.Append", remotelist.AppendArgs{ListID: "outraLista", Value: 99}, &appReply)
	err = client.Call("RemoteList.Size", remotelist.SizeArgs{ListID: "outraLista"}, &sizeReply)
	fmt.Println("Size('outraLista'):", sizeReply.Size) // Esperado: 1

	// Teste de persistência:
	// Pare o servidor (Ctrl+C) e rode o cliente novamente.
	// Os valores de 'minhaLista' (agora [10, 20]) e 'outraLista' (agora [99])
	// devem ser carregados do snapshot/log.
	fmt.Println("\n--- Teste de Persistência ---")
	fmt.Println("Pare o servidor (Ctrl+C) e reinicie-o.")
	fmt.Println("Depois, rode este cliente novamente.")
	fmt.Println("Os tamanhos das listas devem ser os mesmos da execução anterior.")

	// Teste de múltiplos clientes (simulado com goroutines)
	fmt.Println("\n--- Testando Múltiplos Clientes (simulado) ---")
	go func() {
		for i := 0; i < 5; i++ {
			c, err := rpc.Dial("tcp", ":5000")
			if err != nil { return }
			c.Call("RemoteList.Append", remotelist.AppendArgs{ListID: "concorrente", Value: i}, &remotelist.AppendReply{})
			fmt.Println("Cliente A 'concorrente' appended", i)
			c.Close()
			time.Sleep(100 * time.Millisecond)
		}
	}()
	go func() {
		for i := 100; i < 105; i++ {
			c, err := rpc.Dial("tcp", ":5000")
			if err != nil { return }
			c.Call("RemoteList.Append", remotelist.AppendArgs{ListID: "concorrente", Value: i}, &remotelist.AppendReply{})
			fmt.Println("Cliente B 'concorrente' appended", i)
			c.Close()
			time.Sleep(150 * time.Millisecond)
		}
	}()

	// Espera as goroutines terminarem
	time.Sleep(2 * time.Second)
	err = client.Call("RemoteList.Size", remotelist.SizeArgs{ListID: "concorrente"}, &sizeReply)
	fmt.Println("\nSize('concorrente') final:", sizeReply.Size) // Esperado: 10
}