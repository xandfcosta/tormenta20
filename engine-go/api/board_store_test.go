package api

import "t20engine/events"
import "t20engine/tabuleiro"

import "t20engine/aovivo"

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"t20engine/engine"
)

// O tabuleiro sobrevive ao reinício do servidor — a memória é a verdade da
// sessão, mas o servidor cai, e uma mesa que perde as posições no meio da noite
// perde a cena inteira (ALE-124).

func TestBoardPersistsAndComesBack(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))

	abre(t, s, sid, "Taverna do Javali", "taverna")
	if _, err := s.boards.AddToken(ctx, sid, defaultTab, tabuleiro.BoardToken{Label: "Ogro", X: 3, Y: 4, Footprint: 2}, true); err != nil {
		t.Fatalf("adicionar peça: %v", err)
	}
	s.boards.Persist(ctx, sid, defaultTab)

	// Um servidor novo sobre o MESMO banco: é o reinício, sem fingir.
	frio := tabuleiro.NewBoardStore(s.queries, aovivo.NewUUID, &events.Bus{})
	voltou := frio.Get(ctx, sid, defaultTab)

	if voltou == nil {
		t.Fatal("o tabuleiro não voltou do banco")
	}
	if voltou.Place != "Taverna do Javali" || voltou.Terrain != "taverna" {
		t.Errorf("o lugar ou o cenário se perderam: %+v", voltou)
	}
	if len(voltou.Tokens) != 1 || voltou.Tokens[0].X != 3 || voltou.Tokens[0].Y != 4 {
		t.Errorf("a peça voltou fora do lugar: %+v", voltou.Tokens)
	}
	// Quadrado é a unidade guardada: um footprint que volta 0 desenharia uma
	// peça sem corpo e o teto de alcance sairia errado.
	if voltou.Tokens[0].Footprint != 2 {
		t.Errorf("o tamanho da peça se perdeu: %d", voltou.Tokens[0].Footprint)
	}
}

// "Sem tabuleiro" tem de voltar como AUSÊNCIA. Um `tabuleiro.BoardState{}` de cortesia
// desenharia uma grade de 0×0 e o mestre acharia que abriu alguma coisa.
func TestSessionWithoutBoardStaysWithout(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))

	if b := s.boards.Get(ctx, sid, defaultTab); b != nil {
		t.Errorf("sessão nova já veio com tabuleiro: %+v", b)
	}
	if _, err := s.boards.AddToken(ctx, sid, defaultTab, tabuleiro.BoardToken{Label: "Ninguém"}, false); err == nil {
		t.Error("pôs peça num tabuleiro que não existe")
	}
}

func TestClosingBoardErasesItFromDiskToo(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	abre(t, s, sid, "Cripta", "pedra")
	s.boards.Persist(ctx, sid, defaultTab)

	s.boards.Close(ctx, sid, defaultTab)

	if b := s.boards.Get(ctx, sid, defaultTab); b != nil {
		t.Error("o tabuleiro encerrado continua na memória")
	}
	if b := tabuleiro.NewBoardStore(s.queries, aovivo.NewUUID, &events.Bus{}).Get(ctx, sid, defaultTab); b != nil {
		t.Error("o tabuleiro encerrado voltou do banco no próximo reinício")
	}
}

// Abrir um tabuleiro com outro aberto ACRESCENTA uma aba (ALE-205).
//
// Aqui morava `TestReopeningKeepsVersionMovingForward`, que prendia a regra
// oposta: abrir SUBSTITUÍA a cena, e a versão do novo tinha de continuar a do
// velho para o cliente não descartar o quadro como atrasado. Essa regra deixou
// de existir com a issue — são dois tabuleiros, com dois contadores —, e um
// teste sobre ela ficaria verde afirmando um mundo que não é este.
//
// O que ficou no lugar é a garantia que a mesa nota: a taverna continua aberta,
// com as peças onde estavam, e a masmorra nasce vazia.
func TestOpeningASecondBoardKeepsTheFirst(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	taverna := abre(t, s, sid, "Taverna", "taverna")
	if _, err := s.boards.AddToken(ctx, sid, taverna.ID, tabuleiro.BoardToken{Label: "Bandido"}, false); err != nil {
		t.Fatalf("adicionar: %v", err)
	}

	masmorra := abre(t, s, sid, "Masmorra", "pedra")

	if masmorra.ID == taverna.ID {
		t.Fatal("a segunda cena nasceu com o id da primeira: elas são a mesma aba")
	}
	if len(masmorra.Tokens) != 0 {
		t.Errorf("a masmorra nasceu com as peças da taverna: %+v", masmorra.Tokens)
	}
	aindaLa := s.boards.Get(ctx, sid, taverna.ID)
	if aindaLa == nil {
		t.Fatal("abrir a masmorra fechou a taverna — é a issue inteira")
	}
	if len(aindaLa.Tokens) != 1 {
		t.Errorf("a taverna perdeu as peças dela: %+v", aindaLa.Tokens)
	}
	// A PADRÃO continua sendo a mais antiga: quem não escolheu aba nenhuma não
	// pode ser arrastado para a cena que o mestre acabou de abrir — ele pode
	// estar montando a emboscada.
	if padrao := s.boards.Get(ctx, sid, defaultTab); padrao == nil || padrao.ID != taverna.ID {
		t.Errorf("a aba padrão pulou para a cena recém-aberta: %+v", padrao)
	}
}

// AS DUAS CENAS voltam do banco, e na ORDEM em que foram abertas (ALE-205).
//
// A ordem não é enfeite: ela é a ordem das abas na tela e a primeira é a PADRÃO
// de quem ainda não escolheu. Se a hidratação embaralhasse, um reinício no meio
// da noite poria a mesa inteira numa cena diferente da que ela estava olhando —
// e a coluna que a segura é o `openSeq`.
//
// ESTE CASO NASCEU VERMELHO E ACHOU UM DEFEITO DE VERDADE. A primeira versão
// ordenava por um `openedAt` em milissegundos, e as duas cenas do teste abrem no
// MESMO milissegundo: o empate caía no `boardId`, que é um UUID, e o reinício
// devolvia as abas na ordem do sorteio. O contador não empata.
func TestBothBoardsComeBackFromTheDatabaseInOrder(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	taverna := abre(t, s, sid, "Taverna", "taverna")
	cripta := abre(t, s, sid, "Cripta", "pedra")
	if _, err := s.boards.AddToken(ctx, sid, cripta.ID, tabuleiro.BoardToken{Label: "Ogro", X: 7, Y: 7}, true); err != nil {
		t.Fatalf("adicionar: %v", err)
	}
	s.boards.Persist(ctx, sid, taverna.ID)
	s.boards.Persist(ctx, sid, cripta.ID)

	// Um servidor novo sobre o MESMO banco: é o reinício, sem fingir.
	frio := tabuleiro.NewBoardStore(s.queries, aovivo.NewUUID, &events.Bus{})
	voltaram := frio.OpenBoards(ctx, sid)

	if len(voltaram) != 2 {
		t.Fatalf("voltaram %d cenas do banco, esperado 2", len(voltaram))
	}
	if voltaram[0].Place != "Taverna" || voltaram[1].Place != "Cripta" {
		t.Errorf("a ordem de abertura se perdeu no reinício: %q, %q", voltaram[0].Place, voltaram[1].Place)
	}
	// O id atravessa: é por ele que a escolha de aba de cada pessoa continua
	// apontando para a mesma cena depois do reinício.
	if voltaram[1].ID != cripta.ID {
		t.Errorf("o id da cripta mudou no reinício: %q virou %q", cripta.ID, voltaram[1].ID)
	}
	if len(voltaram[1].Tokens) != 1 {
		t.Errorf("a cripta voltou sem as peças dela: %+v", voltaram[1].Tokens)
	}
	if len(voltaram[0].Tokens) != 0 {
		t.Errorf("as peças da cripta apareceram na taverna: %+v", voltaram[0].Tokens)
	}
}

// FECHAR A DO MEIO não pode fazer a próxima nascer empatada (ALE-205).
//
// É o caso que separa `max(openSeq) + 1` de `len + 1`, e ele acontece numa noite
// normal: o mestre encerra a ponte e abre a cripta. Por `len`, a cripta nasceria
// com o número que a taverna já tem — e duas abas com o mesmo número é o empate
// que esta coluna existe para não ter, com a ordem caindo no desempate do
// SQLite.
func TestClosingATabDoesNotMakeTheNextOneTie(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	ponte := abre(t, s, sid, "Ponte", "pedra")
	taverna := abre(t, s, sid, "Taverna", "taverna")
	s.boards.Close(ctx, sid, ponte.ID)

	cripta := abre(t, s, sid, "Cripta", "pedra")

	if cripta.Seq == taverna.Seq {
		t.Fatalf("a cripta nasceu com o número da taverna (%d): a ordem das abas passou a depender do desempate do banco", cripta.Seq)
	}
	if cripta.Seq <= taverna.Seq {
		t.Errorf("a cripta (%d) nasceu ANTES da taverna (%d) na barra", cripta.Seq, taverna.Seq)
	}
}

// O TETO de abertos (ALE-205, decisão do dono: oito).
//
// Sem ele o estado cresce sem limite e toda hidratação e toda gravação o
// carregam — o mesmo argumento do teto de peças que já estava no código.
func TestOpeningRefusesPastTheCeiling(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	for i := 0; i < 8; i++ {
		if _, err := s.boards.Open(ctx, sid, "Cena", "pedra"); err != nil {
			t.Fatalf("a %da cena foi recusada antes do teto: %v", i+1, err)
		}
	}

	_, err := s.boards.Open(ctx, sid, "A nona", "pedra")

	if err == nil {
		t.Fatal("a nona cena abriu: o teto não existe")
	}
	// A recusa carrega o valor ofensor e o teto — "não deu" mandaria o mestre
	// adivinhar quantas ele pode ter no meio da sessão.
	if !strings.Contains(err.Error(), "8") {
		t.Errorf("a recusa não diz quantas cabem: %v", err)
	}
	if n := len(s.boards.OpenBoards(ctx, sid)); n != 8 {
		t.Errorf("a sessão ficou com %d cenas abertas depois da recusa", n)
	}
}

// abre é o `Open` dos testes: eles não medem o teto, e um `if err` por chamada
// esconderia o que cada caso está afirmando.
func abre(t *testing.T, s *Server, sid int64, lugar, chao string) *tabuleiro.BoardState {
	t.Helper()
	b, err := s.boards.Open(context.Background(), sid, lugar, chao)
	if err != nil {
		t.Fatalf("abrir %q: %v", lugar, err)
	}
	return b
}

// Gravação que falha PARA DE SER SILENCIOSA (ALE-124).
//
// Este teste existe por um defeito de verdade: a tabela do tabuleiro sumiu do
// banco de desenvolvimento — a migração constava aplicada, a tabela não existia
// — e o tabuleiro passou um dia inteiro vivendo só em memória. A tela estava
// impecável, e cada gravação falhava numa linha de log que ninguém lê. O que
// faltava não era a gravação: era a mesa SABER que ela parou.
//
// A transição é o que importa: avisa quando começa a falhar e avisa quando
// volta, e não a cada mensagem — um aviso por tique de peça viraria ruído e
// ninguém leria esse também.
func TestBoardPersistFailureIsReported(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	abre(t, s, sid, "Cripta", "pedra")

	if Dirty, changed := s.boards.Persist(ctx, sid, defaultTab); Dirty || changed {
		t.Fatalf("gravação saudável já saiu como falha: Dirty=%v changed=%v", Dirty, changed)
	}

	if _, err := s.db.Exec("DROP TABLE open_boards"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}
	Dirty, changed := s.boards.Persist(ctx, sid, defaultTab)
	if !Dirty || !changed {
		t.Fatalf("a tabela sumiu e ninguém avisou: Dirty=%v changed=%v", Dirty, changed)
	}
	// Segunda falha seguida: continua falhando, mas NÃO é notícia nova.
	if _, changed := s.boards.Persist(ctx, sid, defaultTab); changed {
		t.Error("a mesa levou um aviso a cada gravação, e não só na transição")
	}

	if _, err := s.db.Exec(`CREATE TABLE open_boards (
		sessionId INTEGER NOT NULL, boardId TEXT NOT NULL, state TEXT NOT NULL,
		openSeq INTEGER NOT NULL, updatedAt TEXT NOT NULL,
		PRIMARY KEY (sessionId, boardId))`); err != nil {
		t.Fatalf("recriar a tabela: %v", err)
	}
	if Dirty, changed := s.boards.Persist(ctx, sid, defaultTab); Dirty || !changed {
		t.Errorf("a recuperação não foi anunciada: Dirty=%v changed=%v", Dirty, changed)
	}
}

// Um erro TRANSIENTE de leitura não pode virar "esta sessão não tem tabuleiro"
// até o processo reiniciar (ALE-155).
//
// Este é o gêmeo do defeito da gravação, e o mecanismo é o mesmo: o
// `hydrateLocked` marcava a sessão como já consultada ANTES da query, então uma
// falha de banco na primeira leitura ficava cacheada. A mesa via um tabuleiro
// vazio, o mestre reabria, e o de verdade continuava no disco.
func TestATransientReadFailureIsRetried(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))

	abre(t, s, sid, "Cripta", "pedra")
	if _, err := s.boards.AddToken(ctx, sid, defaultTab, tabuleiro.BoardToken{Label: "Ogro", X: 1, Y: 1}, true); err != nil {
		t.Fatalf("adicionar peça: %v", err)
	}
	s.boards.Persist(ctx, sid, defaultTab)

	// Um servidor frio sobre o mesmo banco, e a leitura falha: é o disco
	// piscando no primeiro acesso à sessão.
	frio := tabuleiro.NewBoardStore(s.queries, aovivo.NewUUID, &events.Bus{})
	if _, err := s.db.Exec("ALTER TABLE open_boards RENAME TO open_boards_escondida"); err != nil {
		t.Fatalf("esconder a tabela: %v", err)
	}
	if vazio := frio.Get(ctx, sid, defaultTab); vazio != nil {
		t.Fatalf("leitura falhou e mesmo assim devolveu tabuleiro: %+v", vazio)
	}

	// O disco volta. A próxima leitura tem de ACHAR o tabuleiro — se a falha
	// tivesse sido cacheada, esta sessão ficaria sem tabuleiro para sempre.
	if _, err := s.db.Exec("ALTER TABLE open_boards_escondida RENAME TO open_boards"); err != nil {
		t.Fatalf("devolver a tabela: %v", err)
	}
	voltou := frio.Get(ctx, sid, defaultTab)

	if voltou == nil {
		t.Fatal("a falha transiente ficou cacheada: a sessão perdeu o tabuleiro até o próximo reinício")
	}
	if len(voltou.Tokens) != 1 {
		t.Errorf("o tabuleiro voltou incompleto: %+v", voltou.Tokens)
	}
}

// "Sessão sem tabuleiro" é resposta DEFINITIVA e continua sendo cacheada: sem
// isso, toda mensagem de uma sessão sem tabuleiro iria ao disco.
func TestNoBoardIsStillCached(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))

	if b := s.boards.Get(ctx, sid, defaultTab); b != nil {
		t.Fatalf("sessão nova veio com tabuleiro: %+v", b)
	}
	// Com a tabela fora do ar, uma segunda leitura só pode responder se estiver
	// respondendo de memória.
	if _, err := s.db.Exec("DROP TABLE open_boards"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}
	if b := s.boards.Get(ctx, sid, defaultTab); b != nil {
		t.Errorf("a segunda leitura foi ao disco em vez de lembrar: %+v", b)
	}
}

// Encerrar o tabuleiro também avisa quando a gravação falha: sem isso a memória
// diz "fechado", o banco mantém a linha, e no próximo boot o tabuleiro fantasma
// volta com as peças de uma cena que a mesa já encerrou (ALE-155).
func TestClosingReportsAFailedDelete(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	abre(t, s, sid, "Cripta", "pedra")
	s.boards.Persist(ctx, sid, defaultTab)

	if _, err := s.db.Exec("DROP TABLE open_boards"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}
	Dirty, changed := s.boards.Close(ctx, sid, defaultTab)

	if !Dirty || !changed {
		t.Fatalf("o encerramento falhou e ninguém soube: Dirty=%v changed=%v", Dirty, changed)
	}
}

// O `/health` conta a degradação em vez de dizer "ok" sempre (ALE-155).
//
// O boot é best-effort de propósito — sem catálogo, autenticação, leitura e
// vitais continuam de pé —, mas a degradação só existia numa linha de log
// enquanto os handlers que precisam do catálogo devolviam 503 no meio de uma
// jogada.
func TestHealthReportsADegradedBoot(t *testing.T) {
	// O servidor de teste sobe SEM catálogo — que é exatamente o estado
	// degradado que o boot de produção assume quando o arquivo falta.
	s := newTestServer(t)

	degradado := healthBody(t, s)

	if degradado["status"] != "degraded" {
		t.Fatalf("sem catálogo o health disse %v", degradado)
	}
	lista, _ := degradado["degraded"].([]any)
	if len(lista) == 0 || lista[0] != "catalogs" {
		t.Errorf("o health não disse O QUE está degradado: %v", degradado)
	}

	// Com catálogo, volta a "ok". Isto prova o ANÚNCIO, não que o catálogo
	// esteja correto — quem prova isso é a validação de schema do `catalog`.
	s.catalogs = &engine.Catalogs{}
	if saudavel := healthBody(t, s); saudavel["status"] != "ok" {
		t.Fatalf("servidor inteiro respondeu %v", saudavel)
	}
}

func healthBody(t *testing.T, s *Server) map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("health respondeu %d — reiniciar não conserta arquivo faltando", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("corpo do health: %v", err)
	}
	return body
}

// O descanso do grupo CONTA quem não descansou (ALE-155).
//
// Best-effort por personagem continua certo — uma ficha que falha não pode
// impedir o descanso das outras quatro. O que estava errado era o silêncio: o
// encerrar-cena era `_, _ =` e nem entrava na conta, então o mestre lia
// "descansou" enquanto duas de cinco fichas não tinham descansado.
func TestPartyRestCountsWhoActuallyRested(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "gm@t.com")
	campaignID := seedCampaign(t, s, gm)
	sid := seedSession(t, s, campaignID)
	heroi := seedCharacter(t, s, gm, "Tanque", 10, 20, 2, 5)
	seedMember(t, s, campaignID, heroi, "player")
	user := AuthUser{ID: gm, Email: "gm@t.com"}

	done, total, err := s.restParty(user, campaignID, sid, "scene", "normal")
	if err != nil || total != 1 || done != 1 {
		t.Fatalf("descanso saudável deu done=%d total=%d err=%v", done, total, err)
	}

	// O disco pisca no meio do descanso do grupo.
	if _, err := s.db.Exec("DROP TABLE active_effects"); err != nil {
		t.Fatalf("derrubar a tabela: %v", err)
	}
	done, total, err = s.restParty(user, campaignID, sid, "scene", "normal")

	if err != nil {
		t.Fatalf("uma ficha que falha não pode derrubar o descanso inteiro: %v", err)
	}
	if total != 1 {
		t.Errorf("o total deixou de contar o grupo: %d", total)
	}
	if done != 0 {
		t.Errorf("contou %d como descansados, e nenhum descansou — é o mestre lendo uma verdade pela metade", done)
	}
}

// O backup automático guarda os N últimos e apaga o resto (ALE-157).
//
// O backup manual já fazia a coisa certa; o que faltava era ele não depender de
// alguém lembrar. E a retenção é parte do recurso: sem ela, backup diário enche
// o disco do dono em silêncio, o que é uma forma nova de perder a mesa.
func TestBackupPruningKeepsTheNewest(t *testing.T) {
	s := newTestServer(t)
	s.cfg.BackupDir = t.TempDir()
	s.cfg.BackupKeep = 2
	ctx := context.Background()

	// Três snapshots com carimbo de hora distinto — o nome do arquivo carrega a
	// data, então precisam ser horas diferentes para não colidir.
	base := time.Date(2026, 8, 18, 3, 0, 0, 0, time.UTC)
	nomes := []string{}
	for i := 0; i < 3; i++ {
		nome, err := s.backupDatabase(ctx, base.Add(time.Duration(i)*time.Hour))
		if err != nil {
			t.Fatalf("backup %d: %v", i, err)
		}
		nomes = append(nomes, nome)
	}

	s.pruneBackups()

	restantes := s.listBackups()
	if len(restantes) != 2 {
		t.Fatalf("sobraram %d backups, esperava 2: %+v", len(restantes), restantes)
	}
	// O mais ANTIGO é quem sai.
	for _, b := range restantes {
		if b.Name == nomes[0] {
			t.Errorf("o backup mais antigo sobreviveu à poda: %s", b.Name)
		}
	}
}

// Um arquivo que não é backup não vira candidato a ser apagado: a poda só
// alcança o que a própria listagem reconhece.
func TestBackupPruningIgnoresStrangers(t *testing.T) {
	s := newTestServer(t)
	s.cfg.BackupDir = t.TempDir()
	s.cfg.BackupKeep = 1
	intruso := filepath.Join(s.cfg.BackupDir, "anotacoes-do-mestre.txt")
	if err := os.WriteFile(intruso, []byte("a taverna pega fogo"), 0o644); err != nil {
		t.Fatalf("escrever intruso: %v", err)
	}

	base := time.Date(2026, 8, 18, 3, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		if _, err := s.backupDatabase(context.Background(), base.Add(time.Duration(i)*time.Hour)); err != nil {
			t.Fatalf("backup %d: %v", i, err)
		}
	}
	s.pruneBackups()

	if _, err := os.Stat(intruso); err != nil {
		t.Errorf("a poda apagou um arquivo que não é backup: %v", err)
	}
}

// Intervalo (ou teto) zero DESLIGA o automático, sem laço nenhum girando: a
// mesa é do dono, e ele pode não querer backup automático.
func TestBackupSchedulerStaysOffWhenDisabled(t *testing.T) {
	s := newTestServer(t)
	s.cfg.BackupDir = t.TempDir()
	s.cfg.BackupEvery = 0

	pronto := make(chan struct{})
	go func() { s.ScheduleBackups(context.Background()); close(pronto) }()

	select {
	case <-pronto:
	case <-time.After(2 * time.Second):
		t.Fatal("o agendador ficou girando com o backup desligado")
	}
	if n := len(s.listBackups()); n != 0 {
		t.Errorf("fez %d backups com o automático desligado", n)
	}
}
