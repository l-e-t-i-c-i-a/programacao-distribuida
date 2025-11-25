package remotelist

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Define nomes dos arquivos usados na persistência e o intervalo entre snapshots
const (
	// Arquivos de persistência
	snapshotFile = "remotelist.json"
	logFile      = "remotelist.log"
	// Intervalo para salvar snapshots (ex: 30 segundos)
	snapshotInterval = 30 * time.Second
)

// --- Structs para Argumentos e Respostas RPC ---
// Cada operação tem seu par de structs Args/Reply.
type AppendArgs struct {
	ListID string
	Value  int
}
type AppendReply struct {
	Success bool
}

type GetArgs struct {
	ListID string
	Index  int
}
type GetReply struct {
	Value int
}

type RemoveArgs struct {
	ListID string
}
type RemoveReply struct {
	Value int
}

type SizeArgs struct {
	ListID string
}
type SizeReply struct {
	Size int
}

// --- Estruturas de Dados do Servidor ---

// ManagedList encapsula uma única lista e seu próprio mutex.
// Isso permite o bloqueio refinado: operações em listas diferentes; diferentes listas podem ser acessadas em paralelo
// (ex: 'listaA' e 'listaB') podem ocorrer em paralelo.
// Cada lista tem seu próprio mutex
type ManagedList struct {
	mu   sync.RWMutex // Protege os dados desta lista específica; mutex de leitura/escrita
	Data []int
}

// RemoteList é a estrutura principal do serviço, registrada com o RPC.
// Ela gerencia o mapa de todas as ManagedLists.
type RemoteList struct {
	mapMu sync.RWMutex // Protege o map 'lists' (criação/deleção de listas)
	lists map[string]*ManagedList //guarda todas as listas

	logLock sync.Mutex // Protege o acesso ao arquivo de log (apenas 1 escrita por vez)
	logFile *os.File
}

// NewRemoteList é o construtor do nosso serviço; cria o objeto RemoteList
// Ele inicializa as estruturas e carrega o estado do disco.
func NewRemoteList() *RemoteList {
	rl := &RemoteList{
		lists: make(map[string]*ManagedList),
	}

	// Carrega o estado persistido (snapshot e depois logs)
	if err := rl.loadFromDisk(); err != nil {
		log.Printf("Erro ao carregar dados do disco: %v. Começando com estado vazio.", err)
		// Garante que o logFile seja criado mesmo se o load falhar
		if rl.logFile == nil {
			f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				// Se não podemos escrever o log, é um erro fatal.
				log.Fatalf("FATAL: Não foi possível abrir o arquivo de log: %v", err)
			}
			rl.logFile = f
		}
	}

	// Inicia a goroutine de background para salvar snapshots
	go rl.snapshotScheduler()

	log.Println("Serviço RemoteList iniciado.")
	return rl
}

// --- Métodos RPC ---

// Append adiciona um valor ao final da lista 'list_id'.
// Cria a lista se ela não existir.
// trava a lista antes de logar para garantir consistência
func (rl *RemoteList) Append(args AppendArgs, reply *AppendReply) error {
	ml := rl.getOrCreateList(args.ListID) //busca ou cria a lista

	// Bloqueia apenas esta lista específica para escrita
	ml.mu.Lock()
	defer ml.mu.Unlock()

	// 1. WAL (Write-Ahead Log): Tenta persistir no disco antes de tudo.
	// Se der erro aqui (disco cheio, falha de I/O), retorna o erro.
	// Como não tocou na memória ainda, o estado do servidor continua consistente.
	err := rl.logOperation("APPEND", args.ListID, &args.Value)
	if err != nil {
		log.Printf("Erro crítico de persistência (Append): %v", err)
		return fmt.Errorf("falha ao gravar no disco (WAL): %w", err)
	}

	// 2. Memória: O disco confirmou a gravação. Agora é seguro alterar a RAM.
	ml.Data = append(ml.Data, args.Value)

	reply.Success = true
	return nil
}

// Get retorna o valor no índice 'i' da lista 'list_id'.
func (rl *RemoteList) Get(args GetArgs, reply *GetReply) error {
	ml, exists := rl.getList(args.ListID)
	if !exists {
		return errors.New("lista não encontrada")
	}

	// Bloqueia esta lista para leitura
	ml.mu.RLock() //bloqueio de leitura, permite múltiplos clientes acessando a lista simultaneamente
	defer ml.mu.RUnlock()

	if args.Index < 0 || args.Index >= len(ml.Data) {
		return errors.New("índice fora dos limites")
	}

	reply.Value = ml.Data[args.Index]
	return nil
}

// Remove e retorna o último elemento da lista 'list_id'.
func (rl *RemoteList) Remove(args RemoveArgs, reply *RemoveReply) error {
	ml, exists := rl.getList(args.ListID)
	if !exists {
		return errors.New("lista não encontrada")
	}

	// Bloqueia esta lista para escrita
	ml.mu.Lock()
	defer ml.mu.Unlock() // Defer garante que o unlock será chamado

	if len(ml.Data) == 0 {
		return errors.New("lista vazia")
	}

	// 1. Preparação: Descobre o valor e índice, mas NÃO remove ainda.
	lastIndex := len(ml.Data) - 1
	valueToRemove := ml.Data[lastIndex]

	// 2. WAL: Registra a intenção de remover no disco.
	err := rl.logOperation("REMOVE", args.ListID, nil)
	if err != nil {
		log.Printf("Erro crítico de persistência (Remove): %v", err)
		return fmt.Errorf("falha ao gravar no disco (WAL): %w", err)
	}

	// 3. Memória: O disco confirmou. Agora podemos cortar o slice na RAM.
	ml.Data = ml.Data[:lastIndex]
	reply.Value = valueToRemove

	return nil
}

// Size retorna o número de elementos na lista 'list_id'.
func (rl *RemoteList) Size(args SizeArgs, reply *SizeReply) error {
	ml, exists := rl.getList(args.ListID)
	if !exists {
		// Se a lista não existe, seu tamanho é 0.
		reply.Size = 0
		return nil
	}

	// Bloqueia esta lista para leitura
	ml.mu.RLock()
	reply.Size = len(ml.Data)
	ml.mu.RUnlock()

	return nil
}

// --- Funções Auxiliares de Gerenciamento de Lista ---

// getList obtém uma lista (apenas leitura do map)
func (rl *RemoteList) getList(listID string) (*ManagedList, bool) {
	rl.mapMu.RLock()
	ml, exists := rl.lists[listID]
	rl.mapMu.RUnlock()
	return ml, exists
}

// getOrCreateList obtém uma lista ou a cria se não existir (escrita no map)
func (rl *RemoteList) getOrCreateList(listID string) *ManagedList {
	// Primeiro, tenta com um Read Lock (otimista)
	rl.mapMu.RLock()
	ml, exists := rl.lists[listID]
	rl.mapMu.RUnlock()
	if exists {
		return ml
	}

	// Se não existir, usa um Write Lock para criar
	rl.mapMu.Lock()
	defer rl.mapMu.Unlock()

	// Double-check: outro cliente pode ter criado a lista
	// enquanto esperávamos pelo Write Lock.
	ml, exists = rl.lists[listID]
	if !exists {
		ml = &ManagedList{
			Data: make([]int, 0),
		}
		rl.lists[listID] = ml
		log.Printf("Lista '%s' criada dinamicamente.", listID)
	}
	return ml
}

// --- Lógica de Persistência (Log e Snapshot) ---

// logOperation escreve uma operação no arquivo de log.
// DEVE ser chamado com a lista (ml.mu) já bloqueada, se aplicável.
func (rl *RemoteList) logOperation(op string, listID string, value *int) error {
	rl.logLock.Lock()
	defer rl.logLock.Unlock()

	var line string
	if op == "APPEND" && value != nil {
		line = fmt.Sprintf("%s %s %d\n", op, listID, *value)
	} else if op == "REMOVE" {
		line = fmt.Sprintf("%s %s\n", op, listID)
	} else {
		return errors.New("operação de log inválida")
	}

	_, err := rl.logFile.WriteString(line)
	if err != nil {
		return err
	}
	// sync: Garante que os dados sejam escritos no disco.
	// É custoso, mas necessário para persistência real.
	return rl.logFile.Sync()
}

// snapshotScheduler executa createSnapshot em intervalos definidos.
func (rl *RemoteList) snapshotScheduler() {
	ticker := time.NewTicker(snapshotInterval)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Iniciando rotina de snapshot...")
		if err := rl.createSnapshot(); err != nil {
			log.Printf("Erro ao criar snapshot: %v", err)
		} else {
			log.Println("Snapshot criado com sucesso.")
		}
	}
}

// createSnapshot salva o estado atual de todas as listas em um arquivo
// e limpa o arquivo de log de forma atômica em relação aos dados copiados.
func (rl *RemoteList) createSnapshot() error {
	// --- FASE 1: Preparação (Coletar referências) ---

	rl.mapMu.RLock()
	listIDs := make([]string, 0, len(rl.lists))
	listsToLock := make([]*ManagedList, 0, len(rl.lists))
	for id, ml := range rl.lists {
		listIDs = append(listIDs, id)
		listsToLock = append(listsToLock, ml)
	}
	rl.mapMu.RUnlock()

	// --- FASE 2: Região Crítica (Cópia e Limpeza do Log) ---

	rl.logLock.Lock()
	defer rl.logLock.Unlock()

	for _, ml := range listsToLock {
		ml.mu.RLock()
	}

	unlockLists := func() {
		for _, ml := range listsToLock {
			ml.mu.RUnlock()
		}
	}

	snapshotData := make(map[string][]int)
	for i, ml := range listsToLock {
		id := listIDs[i]
		dataCopy := make([]int, len(ml.Data))
		copy(dataCopy, ml.Data)
		snapshotData[id] = dataCopy
	}

	// --- CORREÇÃO PARA WINDOWS AQUI ---
	
	// 1. Fecha o arquivo atual (pois o Windows bloqueia Truncate em arquivo O_APPEND)
	rl.logFile.Close()

	// 2. Trunca (zera) o arquivo diretamente pelo caminho no disco
	if err := os.Truncate(logFile, 0); err != nil {
		unlockLists()
		// Tenta reabrir para não quebrar o servidor totalmente
		f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		rl.logFile = f
		return fmt.Errorf("falha ao truncar log: %w", err)
	}

	// 3. Reabre o arquivo limpo para as próximas operações
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		unlockLists()
		return fmt.Errorf("falha ao reabrir log após truncate: %w", err)
	}
	rl.logFile = f

	// --- FIM DA CORREÇÃO ---

	unlockLists()

	// --- FASE 3: Persistência (IO Pesado - Sem Locks de Lista) ---

	tempFile := snapshotFile + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("falha ao criar arquivo temp de snapshot: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshotData); err != nil {
		file.Close()
		os.Remove(tempFile)
		return fmt.Errorf("falha ao serializar (json) snapshot: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("falha ao fechar arquivo temp: %w", err)
	}

	if err := os.Rename(tempFile, snapshotFile); err != nil {
		return fmt.Errorf("falha ao renomear snapshot final: %w", err)
	}

	return nil
}

// loadFromDisk restaura o estado do serviço a partir dos arquivos.
// Primeiro carrega o snapshot, abre log, depois aplica operações do log.
func (rl *RemoteList) loadFromDisk() error {
	// 1. Carregar o Snapshot (se existir)
	if err := rl.loadSnapshot(); err != nil {
		log.Printf("Nenhum snapshot encontrado ou erro ao ler: %v", err)
		// Continua mesmo assim, podemos ter apenas logs.
	} else {
		log.Println("Snapshot carregado com sucesso.")
	}

	// 2. Abrir/Criar o arquivo de Log
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("falha ao abrir arquivo de log para replay: %w", err)
	}
	rl.logFile = f
	// O 'defer f.Close()' não pode ser usado aqui, pois rl.logFile
	// precisa permanecer aberto para operações futuras.

	// 3. Aplicar o Log (replay)
	if err := rl.replayLog(); err != nil {
		return fmt.Errorf("falha ao aplicar replay do log: %w", err)
	}

	log.Println("Logs aplicados (replay) com sucesso.")
	return nil
}

// loadSnapshot carrega o map[string][]int do snapshot
func (rl *RemoteList) loadSnapshot() error {
	file, err := os.Open(snapshotFile)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("snapshot não encontrado")
		}
		return err
	}
	defer file.Close()

	var snapshotData map[string][]int
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&snapshotData); err != nil {
		return fmt.Errorf("falha ao decodificar (json) snapshot: %w", err)
	}

	// Converte os dados carregados para a estrutura de memória (ManagedList)
	// Não precisamos de locks aqui, pois o servidor está iniciando.
	rl.mapMu.Lock()
	for id, data := range snapshotData {
		rl.lists[id] = &ManagedList{
			Data: data,
		}
	}
	rl.mapMu.Unlock()
	return nil
}

// replayLog lê o arquivo de log (rl.logFile) do início
// e aplica as operações em memória.
func (rl *RemoteList) replayLog() error {
	// Volta ao início do arquivo para leitura
	if _, err := rl.logFile.Seek(0, 0); err != nil {
		return fmt.Errorf("falha ao buscar início do log: %w", err)
	}

	scanner := bufio.NewScanner(rl.logFile)
	logCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line) // strings.Fields lida com espaços
		if len(parts) < 2 {
			log.Printf("Linha de log mal formatada, pulando: %s", line)
			continue
		}

		op := parts[0]
		listID := parts[1]

		// Pega ou cria a lista (sem locks, estamos no init)
		ml, exists := rl.lists[listID]
		if !exists {
			ml = &ManagedList{Data: make([]int, 0)}
			rl.lists[listID] = ml
		}

		switch op {
		case "APPEND":
			if len(parts) < 3 {
				log.Printf("Log APPEND mal formatado, pulando: %s", line)
				continue
			}
			val, err := strconv.Atoi(parts[2])
			if err != nil {
				log.Printf("Log APPEND com valor inválido, pulando: %s", line)
				continue
			}
			// Aplica a operação diretamente
			ml.Data = append(ml.Data, val)
			logCount++

		case "REMOVE":
			if len(ml.Data) > 0 {
				ml.Data = ml.Data[:len(ml.Data)-1]
				logCount++
			} else {
				log.Printf("Log REMOVE em lista vazia ('%s'), ignorando.", listID)
			}

		default:
			log.Printf("Operação de log desconhecida, pulando: %s", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("erro ao escanear arquivo de log: %w", err)
	}

	// Volta ao final do arquivo para futuras escritas (Append)
	if _, err := rl.logFile.Seek(0, 2); err != nil {
		return fmt.Errorf("falha ao buscar fim do log: %w", err)
	}
	
	log.Printf("%d operações de log aplicadas.", logCount)
	return nil
}