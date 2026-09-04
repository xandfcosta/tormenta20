package table

import (
	"strings"
	"t20engine/aovivo"
	"t20engine/web/ui"
	"testing"
)

// O guarda que justifica o piloto reusar `stateForRole` em vez de montar a
// própria leitura: a PÁGINA obedece à mesma redação que o socket.
//
// Provado VERMELHO trocando `aovivo.StateForRole(role, ...)` por `s.deps.Sessions().GetState(...)`
// no `LoadView` — o HTML passou a carregar "12/130", os PV que o mestre
// escondeu, para dentro da tela do jogador.
// A vez é MINHA quando a linha na vez é de um personagem meu — e é "de outro"
// quando não é. Tradução literal do `playerTurnState` da SPA; duas escadas
// divergiriam em silêncio.
func TestTableTurnOf(t *testing.T) {
	meu, alheio := int64(7), int64(9)
	fila := []aovivo.InitiativeEntry{
		{Label: "Ogro", Initiative: 19, Type: "npc"},
		{Label: "Arcanista", Initiative: 12, Type: "character", CharacterID: &meu},
	}
	meus := map[int64]bool{meu: true}

	casos := []struct {
		nome      string
		turnIndex int
		kind      string
		label     string
	}{
		{"fora de combate ninguém está na vez", -1, "idle", ""},
		{"a vez do ogro é de outro", 0, "other", "Ogro"},
		{"a vez do meu personagem é minha", 1, "mine", ""},
		{"índice além da fila não inventa uma vez", 5, "idle", ""},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			got := tableTurnOf(&aovivo.SessionRuntimeState{Initiative: fila, TurnIndex: c.turnIndex}, meus)
			if got.Kind != c.kind || got.Label != c.label {
				t.Errorf("veio {%s %q}, queria {%s %q}", got.Kind, got.Label, c.kind, c.label)
			}
		})
	}
	// O personagem alheio não acende a faixa de ninguém.
	outro := tableTurnOf(&aovivo.SessionRuntimeState{
		Initiative: []aovivo.InitiativeEntry{{Label: "Colega", Type: "character", CharacterID: &alheio}},
		TurnIndex:  0,
	}, meus)
	if outro.Kind != "other" {
		t.Errorf("a vez de um PC alheio virou %q", outro.Kind)
	}
}

// Os LIMIARES da cor, e só eles: a tabela inteira de porcentagens seria a
// implementação reescrita. 25 e 50 são os mesmos do `hpFillVar` da SPA, e é a
// divergência entre os dois que este teste existe para tornar barulhenta.
func TestHpToneAtTheThresholds(t *testing.T) {
	casos := []struct {
		pct int
		tom string
	}{
		{0, "bg-hp-critical"},
		{25, "bg-hp-critical"},
		{26, "bg-hp-hurt"},
		{50, "bg-hp-hurt"},
		{51, "bg-hp-full"},
		{100, "bg-hp-full"},
	}
	for _, c := range casos {
		if got := hpTomDe(c.pct); got != c.tom {
			t.Errorf("hpTomDe(%d) = %q, queria %q", c.pct, got, c.tom)
		}
	}
}

// O campo vazio não pode virar um total (ALE-236).
//
// MEDIDO no navegador: o `data-bind` do Datastar escreve ZERO no sinal quando
// um `<input type=number>` esvazia — digitar 7, apagar, e o sinal vai a 0. Sem
// guarda, apagar para redigitar mostra "Total previsto 8" com bônus 8 e dado
// nenhum: um total que não existe, lido no instante da decisão. Mesma família
// da ALE-224, onde a prévia era o que impedia o erro silencioso.
//
// Este guarda pina a EXPRESSÃO, e digo isso em vez de fingir que ele pina o
// comportamento. O teste comportamental exigiria a cena EM JOGO com o jogador
// tendo personagem nela, e montar esse estado no e2e é entrar exatamente na
// armadilha que a ALE-238 documenta: asserção que depende do estado do combate
// mede o banco, não o app. A prova do comportamento foi a medição no navegador,
// que está descrita acima e é reproduzível em três linhas.
func TestTheD20PreviewDoesNotLieWithAnEmptyField(t *testing.T) {
	bonus := int64(8)
	html, err := ui.RenderFragment(t.Context(), tableScene(View{
		CampaignID: 7, SessionID: 42, SceneActive: true,
		Turn: tableTurn{Kind: "idle"},
		Eu:   &tableMe{CharacterID: 1, Nome: "Samira", Bonus: bonus},
	}))
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	// LITERAL e não escapado: o atributo é CONSTANTE, e o templ só escapa os
	// dinâmicos — foi o que a ALE-227 mediu ao comparar as duas saídas byte a
	// byte. Escrevi a forma escapada primeiro e o guarda nasceu vermelho por
	// isso, o que ao menos provou que ele lê o HTML de verdade.
	const faixa = "$d20 >= 1 && $d20 <= 20"
	if !strings.Contains(html, faixa) {
		t.Errorf("a prévia não é condicionada à faixa do dado — campo vazio vira um total inventado")
	}
	if !strings.Contains(html, "informe o dado") {
		t.Error("sem dado, a linha não diz o que falta")
	}
	if !strings.Contains(html, "$registrando || !(") {
		t.Error("o botão continua oferecendo uma ação que o servidor vai recusar")
	}
}
