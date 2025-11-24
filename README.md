# programacao-distribuida

## Discussão sobre Limitações do projeto de RPC 
https://github.com/l-e-t-i-c-i-a/programacao-distribuida/tree/main/remotelist

Após implementar e testar o sistema `RemoteList`, observa-se que ele funciona bem para um servidor único, mas possui limites naturais quando pensamos em um cenário de grande escala. Abaixo, detalho como o sistema se comporta em relação à escalabilidade, disponibilidade e consistência.

**Escalabilidade (O Gargalo do Servidor Único)**
A principal limitação é que tudo roda em uma única máquina.
* **Processamento e Memória:** Conforme o número de clientes aumenta, o servidor pode ficar sobrecarregado (CPU em 100% ou falta de RAM). Embora o uso de travas (`mutex`) individuais por lista ajude bastante — permitindo que clientes mexam na *Lista A* e *Lista B* ao mesmo tempo —, se muitos clientes tentarem acessar a **mesma** lista, cria-se uma fila de espera.
* **Gargalo de Disco:** Tanto o arquivo de log quanto o snapshot precisam ser gravados no disco. Como o disco é um recurso compartilhado e protegido por travas, as operações de escrita acabam sendo feitas "uma por vez". Isso limita quantas operações por segundo o sistema consegue aguentar.
* **Solução mais complexa:** Para resolver isso, uma opção seria usar **Sharding** (dividir as listas entre vários servidores), mas isso tornaria a arquitetura muito mais complexa.

**Disponibilidade**
O sistema possui um "Ponto Único de Falha" (*Single Point of Failure*).
* **O Problema:** Se o processo do servidor travar ou a máquina desligar, o serviço fica totalmente fora do ar. Nenhum cliente consegue fazer nada.
* **A Recuperação:** A boa notícia é que, graças à persistência (Log + Snapshot), os dados estão salvos. Assim que o servidor reiniciar, ele recupera o estado e volta a funcionar. Porém, durante o tempo em que ele estiver desligado, a disponibilidade é zero.
* **Como melhorar:** A solução seria ter réplicas (cópias do servidor). Se o principal cair, um reserva assume.

**Consistência (Forte, mas com detalhes)**
Como existe apenas um servidor, a consistência é o ponto forte da solução. Não dá para dois clientes verem estados diferentes da lista ao mesmo tempo. No entanto, há um detalhe técnico importante na forma como os dados estão sendo salvos:
* **O risco da ordem de gravação:** Atualmente, o sistema altera a memória RAM primeiro e depois grava no log. Existe um risco: se a gravação no disco falhar (ex: disco cheio), o servidor continua rodando com o dado na memória, mas sem ter salvo ele de verdade.
* **Write-Ahead Log (WAL):** O ideal seria usar uma técnica chamada *Write-Ahead Log*, onde garante-se a escrita no disco *antes* de mexer na memória. Isso é mais seguro, mas deixa o sistema mais lento, pois toda operação teria que esperar o disco confirmar.

**Conclusão e Trade-offs**
Para resolver os problemas de escalabilidade e disponibilidade, seria necessário transformar esse sistema em um sistema distribuído com vários nós (vários servidores). Porém, isso traria um novo problema: manter a consistência entre todas as cópias.
Se um cliente atualizar o Servidor A, demora um tempo para essa informação chegar no Servidor B. Nesse intervalo, os dados ficariam inconsistentes.
Ou seja, para ganhar escalabilidade, geralmente tem-se que abrir mão da consistência imediata ou implementar algoritmos complexos de consenso (como Raft ou Paxos). Para o sistema atual, optou-se por ser **simples e consistente**, aceitando ser **menos escalável**.
