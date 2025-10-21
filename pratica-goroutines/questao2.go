package main

import (
	"fmt"
	"math/rand"
	"time"
)

// 1. Tipo estruturado para carregar a leitura e o nome do sensor
type Leitura struct {
	Sensor string
	Valor  string
}

// 2. Funções dos sensores (enviando struct)
func sensorTemperatura(ch chan Leitura) {
	for {
		valor := fmt.Sprintf("%d°C", rand.Intn(100))
		ch <- Leitura{Sensor: "Temperatura", Valor: valor}
		time.Sleep(time.Second)
	}
}

func sensorPressao(ch chan Leitura) {
	for {
		valor := fmt.Sprintf("%d hPa", rand.Intn(1000))
		ch <- Leitura{Sensor: "Pressão", Valor: valor}
		time.Sleep(time.Second)
	}
}

func sensorUmidade(ch chan Leitura) {
	for {
		valor := fmt.Sprintf("%d%%", rand.Intn(100))
		ch <- Leitura{Sensor: "Umidade", Valor: valor}
		time.Sleep(time.Second)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())

	
	ch := make(chan Leitura)

	// Cria goroutines para cada sensor
	go sensorTemperatura(ch)
	go sensorPressao(ch)
	go sensorUmidade(ch)

	// Usa o select para ler e imprimir os dados dos sensores
	fmt.Println("Simulação de Sensores Iniciada. Pressione Ctrl+C para parar.")
	for {
		select {
		case leitura := <-ch:
			// Imprime qual sensor enviou a mensagem
			fmt.Printf("Sensor [%s] enviou a leitura: %s\n", leitura.Sensor, leitura.Valor)
		}
	}
}