package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

var numeros []int

// Função para adicionar número (sem permitir negativos)
func adicionarNumero(num int) error {
	if num < 0 {
		return errors.New("não é permitido adicionar números negativos")
	}
	numeros = append(numeros, num)
	return nil
}

// Função para listar números
func listarNumeros() {
	if len(numeros) == 0 {
		fmt.Println("Nenhum número armazenado.")
		return
	}
	fmt.Println("Números armazenados:", numeros)
}

// Função para remover por índice
func removerPorIndice(ind int) error {
	if ind < 0 || ind >= len(numeros) {
		return errors.New("índice inválido")
	}
	numeros = append(numeros[:ind], numeros[ind+1:]...)
	return nil
}

// Função para calcular estatísticas
func estatisticas() (int, int, float64, error) {
	if len(numeros) == 0 {
		return 0, 0, 0, errors.New("lista vazia")
	}
	min := numeros[0]
	max := numeros[0]
	soma := 0
	for _, v := range numeros {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		soma += v
	}
	media := float64(soma) / float64(len(numeros))
	return min, max, media, nil
}

// Função de divisão segura
func divisaoSegura(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("divisão por zero não é permitida")
	}
	return a / b, nil
}

// Função para limpar lista
func limparLista() {
	numeros = []int{}
	fmt.Println("Lista limpa com sucesso.")
}

// Função para exibir apenas pares
func listarPares() {
	var pares []int
	for _, n := range numeros {
		if n%2 == 0 {
			pares = append(pares, n)
		}
	}
	if len(pares) == 0 {
		fmt.Println("Nenhum número par encontrado.")
		return
	}
	fmt.Println("Números pares:", pares)
}

// Função para exportar lista para arquivo
func exportarParaArquivo(nome string) error {
	arquivo, err := os.Create(nome)
	if err != nil {
		return err
	}
	defer arquivo.Close()

	for _, n := range numeros {
		_, err := arquivo.WriteString(fmt.Sprintf("%d\n", n))
		if err != nil {
			return err
		}
	}
	return nil
}

// Função auxiliar para ler entrada do usuário
func lerEntrada() string {
	reader := bufio.NewReader(os.Stdin)
	texto, _ := reader.ReadString('\n')
	return strings.TrimSpace(texto)
}

func main() {
	for {
		fmt.Println("\n==== Gerenciador de Números ====")
		fmt.Println("1) Adicionar número")
		fmt.Println("2) Listar números")
		fmt.Println("3) Remover por índice")
		fmt.Println("4) Estatísticas (mínimo, máximo, média)")
		fmt.Println("5) Divisão segura")
		fmt.Println("6) Limpar lista")
		fmt.Println("7) Ordenar crescente")
		fmt.Println("8) Ordenar decrescente")
		fmt.Println("9) Exibir apenas pares")
		fmt.Println("10) Exportar para arquivo")
		fmt.Println("0) Sair")
		fmt.Print("Escolha uma opção: ")

		opcao := lerEntrada()

		switch opcao {
		case "1":
			fmt.Print("Digite um número inteiro: ")
			numStr := lerEntrada()
			num, err := strconv.Atoi(numStr)
			if err != nil {
				fmt.Println("Erro: valor inválido.")
				continue
			}
			if err := adicionarNumero(num); err != nil {
				fmt.Println("Erro:", err)
			} else {
				fmt.Println("Número adicionado com sucesso.")
			}
		case "2":
			listarNumeros()
		case "3":
			fmt.Print("Digite o índice a remover: ")
			indStr := lerEntrada()
			ind, err := strconv.Atoi(indStr)
			if err != nil {
				fmt.Println("Erro: índice inválido.")
				continue
			}
			if err := removerPorIndice(ind); err != nil {
				fmt.Println("Erro:", err)
			} else {
				fmt.Println("Número removido com sucesso.")
			}
		case "4":
			min, max, media, err := estatisticas()
			if err != nil {
				fmt.Println("Erro:", err)
			} else {
				fmt.Printf("Mínimo: %d | Máximo: %d | Média: %.2f\n", min, max, media)
			}
		case "5":
			fmt.Print("Digite o dividendo: ")
			aStr := lerEntrada()
			a, err1 := strconv.Atoi(aStr)
			fmt.Print("Digite o divisor: ")
			bStr := lerEntrada()
			b, err2 := strconv.Atoi(bStr)
			if err1 != nil || err2 != nil {
				fmt.Println("Erro: valores inválidos.")
				continue
			}
			res, err := divisaoSegura(a, b)
			if err != nil {
				fmt.Println("Erro:", err)
			} else {
				fmt.Printf("Resultado: %d\n", res)
			}
		case "6":
			limparLista()
		case "7":
			sort.Ints(numeros)
			fmt.Println("Lista ordenada em ordem crescente:", numeros)
		case "8":
			sort.Sort(sort.Reverse(sort.IntSlice(numeros)))
			fmt.Println("Lista ordenada em ordem decrescente:", numeros)
		case "9":
			listarPares()
		case "10":
			fmt.Print("Digite o nome do arquivo: ")
			nome := lerEntrada()
			if err := exportarParaArquivo(nome); err != nil {
				fmt.Println("Erro ao exportar:", err)
			} else {
				fmt.Println("Lista exportada com sucesso para", nome)
			}
		case "0":
			fmt.Println("Encerrando aplicação...")
			return
		default:
			fmt.Println("Opção inválida, tente novamente.")
		}
	}
}
