package main

import (
	"fmt"
)

func escreverValores(id int, ch chan int) {
	for i := 1; i <= 10; i++ {
		ch <- i + id*10 // Escreve 10 valores diferentes para cada goroutine
	}
}

func lerValores(ch chan int, done chan bool) {
	for i := 0; i < 20; i++ {
		valor := <-ch
		fmt.Println("Valor lido:", valor)
	}
	done <- true // Sinaliza que a leitura terminou
}

func main() {
	ch := make(chan int)    // Canal para comunicação
	done := make(chan bool) // Canal para sinalizar o fim das goroutines

	// Cria as goroutines para escrever valores
	go escreverValores(0, ch)
	go escreverValores(1, ch)

	// Cria a goroutine para ler os valores
	go lerValores(ch, done)

	// Aguarda até que a leitura de todos os valores seja concluída
	<-done

	fmt.Println("Fim do programa")
}
