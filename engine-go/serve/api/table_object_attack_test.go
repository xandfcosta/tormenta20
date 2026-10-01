package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"t20engine/domain/board"
)

// O ATAQUE A UM OBJETO SOLTO (ALE-423, p239).
//
//	"Para objetos soltos, faça um ataque contra a Defesa do objeto, definida por
//	 sua categoria de tamanho. [...] Se você acerta o ataque, causa dano normal.
//	 Entretanto, objetos normalmente têm redução de dano, dependendo de seu
//	 material. Um objeto reduzido a 0 ou menos PV é destruído."
//
// INTEGRAÇÃO e não unitário, porque o que pode quebrar aqui é a COMPOSIÇÃO, e
// ela atravessa DOIS AGREGADOS: o provisório mora na fila e o PV do objeto mora
// no tabuleiro, cada store com trava própria. Quem escolhe onde o dano pousa é a
// CENA, acima dos dois — um teste do `Strike` provaria a rolagem e nada sobre
// onde o PV foi parar.

// umObjetoNoMapa põe uma porta de madeira Grande (Defesa 8, RD 5, 20 PV) e
// devolve o id da peça.
func umObjetoNoMapa(t *testing.T, f sceneFixture, x, y int) string {
	t.Helper()
	rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/pecas/nova",
		`{"from":{"X":`+strconv.Itoa(x)+`,"Y":`+strconv.Itoa(y)+`},"new_token_name":"Porta da cripta",`+
			`"new_token_size":"Grande","new_token_look":"object",`+
			`"new_token_material":"madeira","new_token_hp":20}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("pôr a porta deu %d", rec.Code)
	}
	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	for _, peca := range b.Tokens {
		if peca.Label == "Porta da cripta" {
			return peca.ID
		}
	}
	t.Fatalf("a porta não entrou no mapa: %d peças", len(b.Tokens))
	return ""
}

func portaAgora(t *testing.T, f sceneFixture, tokenID string) board.BoardToken {
	t.Helper()
	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	peca := board.FindToken(b, tokenID)
	if peca == nil {
		t.Fatalf("a peça %q sumiu do tabuleiro", tokenID)
	}
	return *peca
}

// O TRAJETO INTEIRO: a porta é atacada, o golpe enfrenta a Defesa DELA, e o PV
// que ela perde é o dano JÁ DESCONTADA a RD.
//
// A asserção que mais paga é a Defesa 8: ela não está em lugar nenhum da peça —
// vem da escada do tamanho (p239), e é a prova de que a regra do commit do motor
// chegou até o d20 por este caminho.
func TestAttackingALooseObjectSpendsItsHitPointsAfterTheDamageReduction(t *testing.T) {
	f, _ := attackOnTurn(t)
	f.seedOpenBoard(t, "stone")
	porta := umObjetoNoMapa(t, f, 5, 0)
	putTokenForEntry(t, f, entryOnTurn(t, f), 1, 0)

	antes := portaAgora(t, f, porta)
	if antes.HpCurrent != 20 {
		t.Fatalf("o controle já estava errado: a porta começou com %d PV", antes.HpCurrent)
	}

	rec := f.requests(t, f.player, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+porta+"/atacar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor o ataque à porta deu %d", rec.Code)
	}
	pendente := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pendente == nil {
		t.Fatalf("o ataque à porta não virou provisório: %q", tableRefusal(t, rec.Body.String()))
	}
	if pendente.Defense != 8 {
		t.Errorf("a porta enfrentou Defesa %d, e a escada do tamanho dá 8 a um objeto Grande",
			pendente.Defense)
	}
	if pendente.TargetTokenID != porta {
		t.Errorf("o provisório aponta para a linha %q e não para a peça %q",
			pendente.TargetEntryID, porta)
	}
	// A FAIXA PRECISA DO NOME, e a porta não tem linha de onde tirá-lo.
	if pendente.TargetLabel != "Porta da cripta" {
		t.Errorf("o provisório não levou o nome do alvo: %q", pendente.TargetLabel)
	}

	dano := pendente.Damage
	// O CÓDIGO NÃO BASTA: o comando da mesa responde 200 com a recusa no corpo,
	// então confirmar o código seria o mesmo guarda cego do `placed.ok()` do e2e.
	confirma := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", "")
	if confirma.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", confirma.Code)
	}
	if frase := tableRefusal(t, confirma.Body.String()); frase != "" {
		t.Fatalf("confirmar foi recusado: %q", frase)
	}
	depois := portaAgora(t, f, porta)
	if depois.HpCurrent != 20-dano {
		t.Errorf("a porta ficou com %d PV; o provisório dizia %d de dano sobre 20",
			depois.HpCurrent, dano)
	}
	// E O PROVISÓRIO SAIU DA MESA: sem isto o mestre confirmaria duas vezes.
	if p := stateOf(t, f.s.sessions, f.sessionID).PendingAttack; p != nil {
		t.Errorf("o provisório continuou pendurado depois de confirmado")
	}
}

// A 0 PV ELA É DESTRUÍDA, e CONTINUA NO MAPA.
//
// As duas metades juntas: apagar a peça sozinha faria a porta arrombada sumir da
// cena, e a mesa precisa continuar vendo o buraco onde ela estava. Quem a tira é
// o mestre.
func TestAnObjectAtZeroHitPointsIsDestroyedAndStaysOnTheMap(t *testing.T) {
	f, _ := attackOnTurn(t)
	f.seedOpenBoard(t, "stone")
	porta := umObjetoNoMapa(t, f, 5, 0)

	// O dano vem do GESTO DO MESTRE e não de rolagens até cair: um laço de
	// ataques esperando o PV zerar seria um teste que depende do dado.
	if _, err := f.s.tableHost().Boards().DamageObject(
		context.Background(), f.sessionID, defaultTab, porta, 20); err != nil {
		t.Fatalf("aplicar o dano: %v", err)
	}
	depois := portaAgora(t, f, porta)
	if !depois.IsDestroyed() {
		t.Errorf("a porta a %d PV não se diz destruída, e a p239 diz que ela é", depois.HpCurrent)
	}
	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	if board.FindToken(b, porta) == nil {
		t.Errorf("a porta destruída sumiu do mapa — quem a tira é o mestre")
	}
	// E ELA NÃO APANHA MAIS: o gesto é recusado com o nome dela.
	rec := f.requests(t, f.player, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+porta+"/atacar", "")
	if frase := tableRefusal(t, rec.Body.String()); frase == "" {
		t.Errorf("atacar a porta já destruída não foi recusado")
	}
}

// CENÁRIO SEM PV É RECUSADO PELO NOME, e não com um erro genérico.
//
// A mancha de musgo é estado legítimo, e quem tenta atacá-la precisa saber que o
// que falta é dar PV à peça — não que o gesto está quebrado.
func TestAttackingSceneryWithoutHitPointsIsRefusedByName(t *testing.T) {
	f, _ := attackOnTurn(t)
	f.seedOpenBoard(t, "stone")
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/pecas/nova",
		`{"from":{"X":4,"Y":0},"new_token_name":"Musgo","new_token_size":"Médio",`+
			`"new_token_look":"object","new_token_material":"","new_token_hp":0}`)
	b := boardRead(f.s.tableHost().Boards().Get(context.Background(), f.sessionID, defaultTab))
	var musgo string
	for _, peca := range b.Tokens {
		if peca.Label == "Musgo" {
			musgo = peca.ID
		}
	}
	if musgo == "" {
		t.Fatalf("o musgo não entrou no mapa")
	}
	rec := f.requests(t, f.player, http.MethodPost,
		f.tableUrl()+"/tabuleiro/pecas/"+musgo+"/atacar", "")
	frase := tableRefusal(t, rec.Body.String())
	if frase == "" {
		t.Fatalf("atacar o musgo não foi recusado")
	}
	for _, pedaco := range []string{"Musgo", "p239"} {
		if !strings.Contains(frase, pedaco) {
			t.Errorf("a recusa não citou %q: %q", pedaco, frase)
		}
	}
}
