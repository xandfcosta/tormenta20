package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O ALVO DA VEZ E O ROLAR (ALE-423, p231 e p234).
//
// É a fatia que fecha o A→B da superfície Ações: até ela, o único gesto que
// PROPUNHA um ataque era o menu da peça do tabuleiro — e sem mapa aberto o
// jogador não tinha gesto nenhum, por mais números que a lista mostrasse.
//
// INTEGRAÇÃO, e pela razão de sempre: o que esta fatia monta é uma LIGAÇÃO
// entre duas cenas que não se conhecem. O alvo é escrito pela barra da MESA,
// num sinal; o botão é desenhado pela FICHA; e quem junta os dois é a porta do
// hospedeiro, que resolve a sessão e a vez. Nenhum teste de unidade de nenhuma
// das três provaria que a ligação existe.
//
// O QUE ESTE ARQUIVO NÃO MEDE é a conta do ataque nem a da manobra: as duas
// moram no `app/combat` e no `domain/engine`, e lá elas já têm caso. Uma regra
// é presa UMA vez, onde ela mora.

// barbarianOnTurn é a bancada: o bárbaro de machado na vez, e um Goblin do
// bestiário na fila para ser alvo.
//
// DO BESTIÁRIO porque é ele que traz a Defesa de verdade — e porque é a mesma
// escolha do `attackOnTurn`, que mede o gesto irmão vindo do tabuleiro.
func barbarianOnTurn(t *testing.T) (sceneFixture, int64, string) {
	t.Helper()
	f, barbaro := barbarianAtTheTable(t, 6)
	// A SESSÃO TEM DE ESTAR EM CURSO, e não é arrumação: o hospedeiro acha a
	// mesa deste personagem varrendo as campanhas dele atrás de sessão
	// `active` — é a mesma varredura que leva o teste de perícia à faixa. Com a
	// sessão só planejada, o gesto sai da ficha e não encontra mesa nenhuma.
	if rec := f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"status":"active"}`); rec.Code != http.StatusOK {
		t.Fatalf("iniciar a sessão deu %d", rec.Code)
	}
	store := f.s.sessions
	if _, err := store.State(t.Context(), f.sessionID); err != nil {
		t.Fatalf("carregar a sessão: %v", err)
	}
	if _, err := store.StartScene(context.Background(), f.sessionID, live.SceneAction); err != nil {
		t.Fatalf("começar a cena de ação: %v", err)
	}
	goblin := "goblin-salteador"
	if _, err := store.AddInitiativeEntry(context.Background(), f.sessionID, live.InitiativeEntry{
		Label: "Goblin", Initiative: 20, Type: "npc", MonsterID: &goblin,
	}); err != nil {
		t.Fatalf("pôr o Goblin na fila: %v", err)
	}
	if _, err := store.AddInitiativeEntry(context.Background(), f.sessionID,
		sheetCombatant("Furioso", 5, barbaro)); err != nil {
		t.Fatalf("pôr o bárbaro na fila: %v", err)
	}
	for range 2 { // a primeira vez é do Goblin, a segunda do bárbaro
		if _, err := store.NextTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}
	state := stateOf(t, store, f.sessionID)
	if state.Initiative[state.TurnIndex].CharacterID == nil {
		t.Fatalf("a vez tinha de ser do bárbaro, e é de %q", state.Initiative[state.TurnIndex].Label)
	}
	return f, barbaro, entryLabeled(t, state, "Goblin")
}

// entryLabeled acha a linha da fila pelo rótulo, e FALHA quando não acha: uma
// string vazia devolvida daqui viraria um alvo vazio lá embaixo, e o caso
// reprovaria pela recusa de "escolha um alvo" em vez de pelo que mede.
func entryLabeled(t *testing.T, state *live.SessionRuntimeState, label string) string {
	t.Helper()
	for _, entry := range state.Initiative {
		if entry.Label == label {
			return entry.ID
		}
	}
	t.Fatalf("%q não está na fila", label)
	return ""
}

// rollsFromTheActions manda o gesto COM o alvo no sinal, por um servidor HTTP
// de verdade.
//
// Pelo `posts` e não pelo `requests`: o alvo viaja como SINAL, e o
// `ReadSignals` só se exercita de verdade num ciclo de pedido completo — o par
// `httptest.NewRequest` + recorder não reproduz o fechamento do corpo que o SDK
// do Datastar faz.
func rollsFromTheActions(t *testing.T, f sceneFixture, id int64, path, target string) string {
	t.Helper()
	body := ""
	if target != "" {
		body = fmt.Sprintf(`{"turn_target":%q}`, target)
	}
	return sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/%s?embutida=1", id, path), body))
}

// O GESTO SAI DA FICHA E CHEGA NA MESA, que é a fatia inteira numa frase.
//
// O que se prende é o PROVISÓRIO com o ALVO CERTO, e não "a resposta deu 200":
// a cena da ficha responde 200 para recusa também — ela redesenha a tela com a
// frase, porque o Datastar descarta remendo que não é 2xx.
func TestTheActionsSurfaceRollsTheAttackOnTheChosenTarget(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)

	if refusal := rollsFromTheActions(t, f, barbaro, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o ataque da superfície Ações foi recusado: %s", refusal)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil {
		t.Fatal("rolar pela superfície Ações não deixou provisório na mesa")
	}
	// O ALVO é o que esta fatia acrescenta, e é por isso que ele é a asserção:
	// com o sinal ignorado, o gesto ainda acharia a vez e ainda rolaria — contra
	// a primeira linha da fila, em silêncio.
	if pending.TargetEntryID != goblin {
		t.Errorf("o provisório mirou %q, e o alvo escolhido era o Goblin (%s)",
			pending.TargetEntryID, goblin)
	}
	if pending.Maneuver != nil {
		t.Error("um golpe virou manobra")
	}
}

// A MANOBRA ATRAVESSA O MESMO CANO, e o caso existe porque o que muda entre os
// dois é um parâmetro de caminho que ninguém mais lê: um erro de grafia ali —
// `{kind}` de um lado, `{manobra}` do outro — faria o gesto chegar ao motor com
// a manobra VAZIA, que é exatamente um golpe. Verde, e medindo o caso de cima
// de novo.
func TestTheManeuverFromTheActionsSurfaceReachesTheTable(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)

	if refusal := rollsFromTheActions(t, f, barbaro, "manobra/derrubar", goblin); refusal != "" {
		t.Fatalf("a manobra da superfície Ações foi recusada: %s", refusal)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil || pending.Maneuver == nil {
		t.Fatal("derrubar pela superfície Ações não deixou manobra na mesa")
	}
	if pending.Maneuver.Kind != "derrubar" {
		t.Errorf("a manobra chegou como %q", pending.Maneuver.Kind)
	}
}

// SEM ALVO NÃO SE ROLA, e a recusa é da TELA: ela nomeia a barra que a pessoa
// não usou, porque o `app/combat` não sabe que essa barra existe.
//
// O botão já nasce desabilitado sem alvo — isto é o que sobra para quem chegou
// pelo endereço, e é a fronteira de segurança: travar na UI é UX.
func TestNoRollLeavesTheActionsSurfaceWithoutATarget(t *testing.T) {
	f, barbaro, _ := barbarianOnTurn(t)

	refusal := rollsFromTheActions(t, f, barbaro, "atacar/0", "")
	if !strings.Contains(refusal, "alvo da vez") {
		t.Errorf("a recusa tinha de mandar escolher o alvo, e veio %q", refusal)
	}
	if stateOf(t, f.s.sessions, f.sessionID).PendingAttack != nil {
		t.Error("sem alvo escolhido, o ataque foi rolado assim mesmo")
	}
}

// A RECUSA TEM DE CHEGAR NA SUPERFÍCIE DE ONDE O GESTO SAIU.
//
// Medido no navegador, e é o defeito que OLHAR a tela pegou: a frase da regra é
// desenhada pelo `ruleRefusal`, que mora dentro do `#sheet-scene` — e esse nó
// vive na superfície FICHA, escondido por `data-show` enquanto a pessoa está
// nas Ações. O jogador apertava Rolar, o servidor recusava com a frase certa, e
// a tela dele não mudava NADA. Um gesto que não faz nada e não diz por quê é
// pior do que um botão que não existe.
//
// O RECORTE é o fragmento do `#actions-scene` e não a resposta inteira: a
// resposta traz os DOIS remendos, e `Contains` no corpo todo passaria verde
// encontrando a frase no remendo da ficha — que é exatamente o estado defeituoso.
func TestTheRefusalReachesTheSurfaceTheGestureCameFrom(t *testing.T) {
	f, barbaro, _ := barbarianOnTurn(t)

	body := f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", barbaro), "")
	// O CONTROLE: a recusa existe na resposta. Sem ele, um recorte que errasse o
	// id passaria verde dizendo que a frase não chegou por outro motivo.
	if !strings.Contains(body, "alvo da vez na barra acima antes de rolar") {
		t.Fatalf("a resposta não trouxe recusa nenhuma — o caso não mede nada:\n%s", recorte(body))
	}
	at := strings.Index(body, `id="actions-scene"`)
	if at < 0 {
		t.Fatalf("a resposta não remendou o #actions-scene:\n%s", recorte(body))
	}
	if !strings.Contains(body[at:], "alvo da vez na barra acima antes de rolar") {
		t.Errorf("a recusa não chegou à superfície Ações, só à Ficha:\n%s", recorte(body[at:]))
	}
}

// FORA DA VEZ NÃO SE ROLA, e quem o diz é o servidor.
//
// Isto é o que separa a superfície Ações do menu da peça: o menu só existe
// desenhado sobre o tabuleiro aberto, e esta lista fica na tela o turno
// inteiro, inclusive quando a vez virou — o botão que estava bom há um segundo
// continua lá.
func TestTheActionsSurfaceRefusesToRollOutOfTurn(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	if _, err := f.s.sessions.NextTurn(context.Background(), f.sessionID); err != nil {
		t.Fatalf("girar a vez: %v", err)
	}

	refusal := rollsFromTheActions(t, f, barbaro, "atacar/0", goblin)
	if !strings.Contains(refusal, "não é a vez") {
		t.Errorf("fora da vez a recusa tinha de dizer de quem é a vez, e veio %q", refusal)
	}
	if stateOf(t, f.s.sessions, f.sessionID).PendingAttack != nil {
		t.Error("fora da vez, o ataque foi rolado assim mesmo")
	}
}

var targetOption = regexp.MustCompile(`<option value="([^"]*)">([^<]*)</option>`)

// A BARRA DO ALVO DIZ O NOME E NADA MAIS (decisão do dono, ALE-423).
//
//	"o jogador nunca vê stats de alvo; tecnicamente num role play um personagem
//	 não sabe essa informação, logo o player também não deve saber"
//
// O que o tipo `turnTarget` já garante por construção, este caso garante na
// TELA — que é onde a regressão apareceria: acrescentar " · Def 13" ao rótulo é
// uma linha de template, e o tipo não a barra.
//
// DENOMINADOR: ele afirma QUANTAS opções olhou antes de afirmar que nenhuma
// vazou. Sem isso, um recorte que não casasse com nada passaria verde dizendo
// que nada vazou.
func TestTheTargetBarOffersTheNameAndNothingElse(t *testing.T) {
	f, _, _ := barbarianOnTurn(t)

	bar := entre(t, acoesDaMesa(t, f), `id="table-turn-target"`, "</section>")
	named := map[string]bool{}
	for _, option := range targetOption.FindAllStringSubmatch(bar, -1) {
		if option[1] == "" { // o convite, que não é alvo nenhum
			continue
		}
		named[option[2]] = true
	}
	if len(named) != 1 {
		t.Fatalf("a barra tinha de oferecer um alvo — o Goblin — e ofereceu %d: %v", len(named), named)
	}
	if !named["Goblin"] {
		t.Errorf("a opção não é o nome cru do Goblin: %v", named)
	}
}

// E O PRÓPRIO PERSONAGEM NÃO ESTÁ LÁ: o `Propose` recusa atacar a si mesmo, e
// uma opção que existe para levar recusa é um erro desenhado.
//
// O CONTROLE está no caso acima, e é por isso que este pode ser uma linha: lá
// se prova que a barra foi encontrada e que ela oferece alguém.
func TestTheTargetBarDoesNotOfferTheCharacterItself(t *testing.T) {
	f, _, _ := barbarianOnTurn(t)

	if bar := entre(t, acoesDaMesa(t, f), `id="table-turn-target"`, "</section>"); strings.Contains(bar, "Furioso") {
		t.Errorf("a barra oferece o próprio personagem como alvo:\n%s", recorte(bar))
	}
}

// O GESTO FICA NA LINHA DO NÚMERO QUE ELE USA, e é isso que a fatia desenha.
//
// Prende a fiação inteira numa asserção: o painel escreveu o endereço com o
// ÍNDICE da arma, o template o montou com `sheetPost` (que carrega a aba e a
// marca de embutida que a URL da página não leva), e o botão nasce preso ao
// sinal que a barra da Mesa escreve. Qualquer um dos três quebrado sai daqui.
//
// RECORTADO AO GRUPO DA AÇÃO PADRÃO: `Contains` na cena inteira acharia o
// endereço em qualquer lugar da página, inclusive num grupo errado.
func TestTheRollSitsOnTheLineOfTheNumberItUses(t *testing.T) {
	f, _, _ := barbarianOnTurn(t)

	cena := acoesDaMesa(t, f)
	padrao := entre(t, superficieDeAcoes(t, cena), "Ação padrão", "Ação de movimento")
	if !strings.Contains(padrao, "/acoes/atacar/0?") {
		t.Errorf("a linha de Atacar não oferece o Rolar:\n%s", recorte(padrao))
	}
	// A MANOBRA É POR NOME e não um "Rolar" genérico: cinco botões iguais
	// obrigariam a contar a posição para saber qual é qual.
	if !strings.Contains(padrao, "/acoes/manobra/derrubar?") {
		t.Errorf("a Manobra não oferece o Derrubar:\n%s", recorte(padrao))
	}
	// O BOTÃO NASCE PRESO AO ALVO. Sem isto ele apareceria clicável sem alvo
	// escolhido, existindo só para levar a recusa do servidor.
	if !strings.Contains(padrao, "$turn_target === &#39;&#39;") {
		t.Errorf("o Rolar não espera o alvo da vez:\n%s", recorte(padrao))
	}
	// O CONTROLE: a FINTA é teste de perícia e NÃO pede alvo, então ela prova
	// que a tela distingue os dois — sem ela, prender tudo ao sinal também
	// passaria verde aqui.
	if !strings.Contains(padrao, "/pericias/rolar/Engana") {
		t.Errorf("a Finta não oferece o teste de Enganação:\n%s", recorte(padrao))
	}
}
