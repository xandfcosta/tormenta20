package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

// avisoDeGravacao é a frase que a Mesa desenha, escrita à mão aqui de propósito.
//
// Importar a constante da cena faria o teste andar junto com o defeito: trocar
// a frase nos dois lugares deixaria o guarda verde sobre um texto que ninguém
// escolheu. É a mesma razão pela qual a bancada da porta copia o parser em vez
// de importá-lo.
const avisoDeGravacao = "não está sendo salva"

// O MESTRE É AVISADO QUANDO A GRAVAÇÃO FALHA (ALE-288).
//
// A mesa roda de MEMÓRIA e grava no disco a cada mutação. Quando essa gravação
// falha — disco cheio, banco fechado, permissão — os dois stores marcam a
// sessão como suja e continuam servindo o estado da memória: a tela fica certa,
// o jogo segue, e o disco não tem nada.
//
// Isso já aconteceu neste repositório: o tabuleiro passou um dia inteiro
// vivendo só em memória, cada gravação falhando numa linha de log que ninguém
// lê (ALE-154). O que existia como remédio era um `persistence-warning` emitido
// no `SSEHub` — e o hub não tem ouvinte em produção desde a ALE-272, quando a
// SPA (a única coisa que abria conexão nele) foi apagada. **O aviso ia para o
// vazio, e a Mesa em Datastar não desenhava nada.**
//
// # Por que ESTADO e não evento
//
// Um aviso perdido é um aviso que não existiu, e este precisa valer enquanto
// durar: quem abre a aba dez minutos depois da primeira falha merece vê-lo. A
// verdade mora no store (é ele quem sabe se a última gravação deu certo), e a
// tela a LÊ a cada quadro — que é a regra que o próprio barramento de eventos
// desta casa escreve: *o evento é a notícia, a verdade está no store.*
func TestTheGmIsWarnedWhenSavingFails(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)

	// O CONTROLE primeiro: com o disco saudável a frase NÃO está lá. Sem esta
	// metade, "vi o aviso" não distingue a ligação certa de um texto fixo.
	saudavel := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(saudavel, avisoDeGravacao) {
		t.Fatalf("a mesa saudável já avisava que a gravação falhou")
	}

	quebraAGravacao(t, f)

	comFalha := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(comFalha, avisoDeGravacao) {
		t.Errorf("a gravação falhou e o mestre não foi avisado")
	}
}

// O JOGADOR NÃO recebe o aviso, e isso é decisão de produto: quem pode parar a
// sessão e chamar alguém é o mestre. Para o jogador seria um alarme sobre o qual
// ele não tem o que fazer.
func TestThePlayerIsNotWarnedAboutSaving(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	quebraAGravacao(t, f)

	corpo := f.pede(t, f.jogador, http.MethodGet, f.tableUrl(), "").Body.String()

	// O controle é afirmar que o jogador VIU a mesa: sem isto, um 403 passaria
	// como "não recebeu o aviso".
	if !strings.Contains(corpo, "Arcanista") {
		t.Fatal("o jogador não viu a própria mesa; a ausência abaixo não prova nada")
	}
	if strings.Contains(corpo, avisoDeGravacao) {
		t.Errorf("o jogador recebeu o aviso de gravação")
	}
}

// quebraAGravacao faz a TABELA do tabuleiro sumir debaixo do store.
//
// É a reprodução literal da ALE-154, e ela foi escolhida depois de duas
// tentativas piores:
//
//   - `boards.Persist` com o fixture cru não tentava escrever nada — o
//     `f.scene(t)` abre a cena e enche a fila, mas não abre tabuleiro. O store
//     não achava o que gravar, devolvia `changed=false`, e o teste DISSE isso em
//     vez de passar;
//   - fechar o `*sql.DB` reproduz a falha de escrita e mata a LEITURA junto,
//     então a página nem renderiza. O defeito de verdade não é esse: é escrita
//     falhando com leitura funcionando, que é como ele fica invisível.
//
// Derrubar uma tabela dá exatamente isso — o `sessions`, o `users` e os membros
// continuam lá, a mesa desenha normalmente, e só a gravação do tabuleiro falha.
func quebraAGravacao(t *testing.T, f pilotoFixture) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.s.boards.Open(ctx, f.sessionID, "Taverna do Javali", "taverna"); err != nil {
		t.Fatalf("abrir tabuleiro: %v", err)
	}
	if _, err := f.s.db.ExecContext(ctx, "DROP TABLE open_boards"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}
	sujo, mudou := f.s.boards.Persist(ctx, f.sessionID, "")
	if !sujo || !mudou {
		t.Fatalf("a gravação não falhou (sujo=%v, mudou=%v) — o defeito não foi reproduzido", sujo, mudou)
	}
}

// UMA MUTAÇÃO PELA CENA CHEGA AO DISCO (ALE-288).
//
// Este é o guarda que faltava, e a ausência dele é o que tornava um acidente
// possível. Os casos do `board_store_test.go` dirigem o STORE direto
// (`bs.Open`, `bs.Persist`); nenhum media o caminho inteiro — comando da Mesa,
// regra, disco.
//
// Sem ele, apagar a gravação não quebra teste nenhum. E apagá-la era a leitura
// NATURAL do código de antes: a linha que gravava vivia dentro do
// `publishBoardState`, cujo canal (`SSEHub`) não tem ouvinte em produção desde a
// ALE-272 — "isto emite para ninguém, pode sair" levaria o disco junto, e a mesa
// passaria a viver só em memória. É literalmente a ALE-154, que custou um dia.
//
// A separação em `saveBoard` e `publishBoardState` tornou o engano difícil; este
// caso o torna impossível de passar despercebido.
func TestACommandFromTheTableReachesTheDisk(t *testing.T) {
	f := novoPiloto(t)
	f.scene(t)
	f.seedOpenBoard(t, "pedra")
	ficha, _ := sceneIds(t, f)

	f.posta(t, f.mestre, f.tableUrl()+"/tabuleiro/pecas", `{"escolhidosdomapa":"`+ficha+`"}`)

	// O CONTROLE: a peça entrou na memória. Sem isto, um disco vazio não
	// distingue "não gravou" de "não havia o que gravar".
	if b := f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab); len(b.Tokens) == 0 {
		t.Fatal("a peça não entrou no mapa — o guarda mediria o vazio")
	}
	esperaOTabuleiroNoDisco(t, f)
}

// esperaOTabuleiroNoDisco sonda até a linha aparecer.
//
// A gravação é em GOROUTINE de propósito — o mestre não espera o disco no meio
// do turno —, então ler uma vez logo depois do comando é uma corrida. Sondagem e
// não `sleep` fixo, pela razão de sempre: um tempo escolhido nesta máquina é o
// teste que pisca na de outra pessoa.
func esperaOTabuleiroNoDisco(t *testing.T, f pilotoFixture) {
	t.Helper()
	limite := time.Now().Add(2 * time.Second)
	for time.Now().Before(limite) {
		var linhas int
		if err := f.s.db.QueryRowContext(context.Background(),
			"SELECT COUNT(*) FROM open_boards WHERE sessionId = ?", f.sessionID).Scan(&linhas); err != nil {
			t.Fatalf("consultar o disco: %v", err)
		}
		if linhas > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("o tabuleiro mexeu e o disco não recebeu nada — a mesa está vivendo só em memória")
}
