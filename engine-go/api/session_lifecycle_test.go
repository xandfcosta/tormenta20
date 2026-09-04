package api

import (
	"context"
	"strings"
	"testing"
)

// O CICLO DE VIDA DA SESSÃO (ALE-269).
//
// Estes guardas nasceram junto com a EXTRAÇÃO da regra para fora do handler
// HTTP, e a razão de nascerem é que ela não tinha nenhum: uma varredura por
// `start`, `end`, `ReopenSession` e a frase da recusa achou só testes de
// AUTORIZAÇÃO. A regra — três estados, cada um querendo dizer outra coisa —
// atravessou o repositório inteiro sem rede, e eu a movi de arquivo antes de
// perceber isso.

// sessaoDoBanco relê a linha, que é onde o estado mora de verdade.
func sessaoDoBanco(t *testing.T, f pilotoFixture) (status string, comeco bool) {
	t.Helper()
	s, err := f.s.queries.GetSession(context.Background(), f.sessionID)
	if err != nil {
		t.Fatalf("reler a sessão: %v", err)
	}
	return s.Status, s.Startedat.Valid
}

// TestStartingMeansThreeThings — é por isto que a regra merece função própria.
func TestStartingMeansThreeThings(t *testing.T) {
	f := novoPiloto(t)
	ctx := context.Background()

	// 1. PLANEJADA começa do zero, e carimba o início.
	sess, err := f.s.queries.GetSession(ctx, f.sessionID)
	if err != nil {
		t.Fatalf("ler a sessão: %v", err)
	}
	if sess.Status != "planned" {
		t.Fatalf("a bancada não nasce planejada (%q) — o resto do caso não mede o que diz", sess.Status)
	}
	if _, err := f.s.tableRules().StartSession(ctx, sess); err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	status, temComeco := sessaoDoBanco(t, f)
	if status != "active" || !temComeco {
		t.Fatalf("planejada → iniciar deu status %q, carimbo %v", status, temComeco)
	}

	// 2. JÁ ATIVA não faz nada, e NÃO é erro: clicar duas vezes é o gesto de
	//    quem não viu a tela mudar, e recusar seria punir a dúvida.
	ativa, _ := f.s.queries.GetSession(ctx, f.sessionID)
	if _, err := f.s.tableRules().StartSession(ctx, ativa); err != nil {
		t.Errorf("iniciar uma sessão já ativa deu erro: %v", err)
	}
	if status, _ := sessaoDoBanco(t, f); status != "active" {
		t.Errorf("iniciar de novo mexeu no estado: %q", status)
	}

	// 3. ENCERRADA REABRE. A noite continuou, e obrigar a criar uma sessão nova
	//    perderia a fila e o tabuleiro dela.
	if _, err := f.s.tableRules().EndSession(ctx, ativa); err != nil {
		t.Fatalf("encerrar: %v", err)
	}
	encerrada, _ := f.s.queries.GetSession(ctx, f.sessionID)
	if encerrada.Status != "ended" {
		t.Fatalf("encerrar não encerrou: %q", encerrada.Status)
	}
	if _, err := f.s.tableRules().StartSession(ctx, encerrada); err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	if status, _ := sessaoDoBanco(t, f); status != "active" {
		t.Errorf("encerrada → iniciar não reabriu: %q", status)
	}
}

// TestEndingASessionThatNeverStartedIsRefused.
//
// A recusa é DIFERENTE do "já ativa" do caso acima, e a diferença é o ponto:
// encerrar uma planejada não é um clique repetido, é um gesto sobre a coisa
// errada. Carimbar um fim numa noite que não teve início deixaria o histórico
// dizendo que ela aconteceu.
func TestEndingASessionThatNeverStartedIsRefused(t *testing.T) {
	f := novoPiloto(t)
	ctx := context.Background()
	sess, err := f.s.queries.GetSession(ctx, f.sessionID)
	if err != nil {
		t.Fatalf("ler a sessão: %v", err)
	}

	_, err = f.s.tableRules().EndSession(ctx, sess)

	if err == nil {
		t.Fatal("encerrar uma sessão nunca iniciada passou")
	}
	// A mensagem nomeia QUAL sessão, que é a regra da casa para erro.
	if !strings.Contains(err.Error(), "nunca foi iniciada") {
		t.Errorf("a recusa não diz o que houve: %q", err)
	}
	if status, _ := sessaoDoBanco(t, f); status != "planned" {
		t.Errorf("a recusa mexeu no estado: %q", status)
	}

	// E ENCERRAR DUAS VEZES não é erro, pela mesma razão que iniciar duas vezes
	// não é: o segundo clique é de quem não viu a tela mudar.
	if _, err := f.s.tableRules().StartSession(ctx, sess); err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	ativa, _ := f.s.queries.GetSession(ctx, f.sessionID)
	if _, err := f.s.tableRules().EndSession(ctx, ativa); err != nil {
		t.Fatalf("encerrar: %v", err)
	}
	encerrada, _ := f.s.queries.GetSession(ctx, f.sessionID)
	if _, err := f.s.tableRules().EndSession(ctx, encerrada); err != nil {
		t.Errorf("encerrar de novo deu erro: %v", err)
	}
}

// TestRestartingCombatEmptiesTheTrackerAndNothingElse.
//
// Reiniciar NÃO é encerrar: a sessão continua ao vivo, o que some é a ordem e
// os turnos. Os dois verbos morando na mesma tela, um do lado do outro, é
// exatamente onde a confusão custaria a noite de alguém.
func TestRestartingCombatEmptiesTheTrackerAndNothingElse(t *testing.T) {
	f := novoPiloto(t)
	ctx := context.Background()
	f.scene(t)

	// O CONTROLE: há fila para esvaziar, e a sessão está ao vivo.
	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n < 2 {
		t.Fatalf("a cena montou %d combatentes — não há o que reiniciar", n)
	}

	if err := f.s.tableRules().RestartCombat(ctx, f.sessionID); err != nil {
		t.Fatalf("reiniciar: %v", err)
	}

	// A LEITURA É DA FILA AO VIVO, e esta escolha custou uma sabotagem: eu tinha
	// escrito a asserção contra o BANCO, e ela era verdadeira ANTES do reset —
	// a fila mora em memória e o banco ainda tinha o `initiative:[]` do início.
	// O guarda media um canal onde o evento nunca passa.
	//
	// Foi essa medição errada que escondeu o defeito de verdade: a extração da
	// regra tinha PERDIDO o `sessions.Forget`, sem o qual o cache continua
	// servindo a fila velha e o reinício não muda nada na tela.
	if n := len(f.s.sessions.GetState(f.sessionID).Initiative); n != 0 {
		t.Errorf("a fila ao vivo continua com %d combatentes depois do reinício", n)
	}
	// E o BANCO também, que é o que sobrevive a um reinício do processo.
	linha, err := f.s.queries.GetSession(ctx, f.sessionID)
	if err != nil {
		t.Fatalf("reler a sessão: %v", err)
	}
	if !strings.Contains(linha.Runtimestate, `"initiative":[]`) {
		t.Errorf("a fila não foi esvaziada no banco: %.80s", linha.Runtimestate)
	}
	if linha.Status != "planned" && linha.Status != "active" {
		t.Errorf("reiniciar mexeu no ciclo da sessão: %q", linha.Status)
	}
}
