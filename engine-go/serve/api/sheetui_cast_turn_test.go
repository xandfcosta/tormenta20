package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// CONJURAR CUSTA AÇÃO, E A REAÇÃO NÃO ESPERA A SUA VEZ (T20 p233).
//
// O caso é de INTEGRAÇÃO porque o que ele prende é a COSTURA, e ela atravessa
// três lugares que não se conhecem: a regra do instante mora no
// `engine.UsableNow`, o que sobrou do turno mora no `live.Scene`, e quem
// conjura é a FICHA — que até aqui não enxergava a mesa.

// learnAndCast ensina a magia e conjura, pelas rotas da aba Magias.
func learnAndCast(t *testing.T, f sceneFixture, spellID string) *responseRecorderLike {
	t.Helper()
	learnURL := fmt.Sprintf("/personagens/%d/magias/aprende/%s?tab=spells", f.charID, spellID)
	if rec := f.pede(t, f.player, http.MethodPost, learnURL, ""); rec.Code != http.StatusOK {
		t.Fatalf("aprender %q respondeu %d: %s", spellID, rec.Code, rec.Body.String())
	}
	castURL := fmt.Sprintf("/personagens/%d/magias/conjura/%s?tab=spells", f.charID, spellID)
	rec := f.pede(t, f.player, http.MethodPost, castURL, "")
	return &responseRecorderLike{Code: rec.Code, Body: rec.Body.String()}
}

// startCombatWithSomeoneElseOnTurn põe a cena de ação no ar com a vez de OUTRO.
//
// O outro é um NPC sem ficha, e ele existe para que a vez possa ser de alguém:
// com um combatente só, "não é a sua vez" não tem como acontecer, e o caso
// mediria a metade em que o defeito é invisível.
func startCombatWithSomeoneElseOnTurn(t *testing.T, f sceneFixture) {
	t.Helper()
	store := f.s.sessions
	if _, err := store.Load(t.Context(), f.sessionID); err != nil {
		t.Fatalf("carregar a sessão: %v", err)
	}
	if _, err := store.StartScene(f.sessionID, live.SceneAction); err != nil {
		t.Fatalf("começar a cena de ação: %v", err)
	}
	if _, err := store.AddInitiativeEntry(f.sessionID, live.InitiativeEntry{
		ID: "npc", Label: "Goblin", Initiative: 20, Type: "npc",
	}); err != nil {
		t.Fatalf("pôr o NPC na fila: %v", err)
	}
	if _, err := store.AddInitiativeEntry(f.sessionID,
		sheetCombatant("Arcanista", 5, f.charID)); err != nil {
		t.Fatalf("pôr o personagem na fila: %v", err)
	}
	if _, err := store.NextTurn(f.sessionID); err != nil {
		t.Fatalf("girar para a primeira vez: %v", err)
	}
	state := store.GetState(f.sessionID)
	if who := state.Initiative[state.TurnIndex]; who.CharacterID != nil {
		t.Fatalf("o controle falhou: a vez é do personagem (%s), e o caso precisa dela ser de outro", who.Label)
	}
}

// FORA DA SUA VEZ, A PADRÃO RECUSA E A REAÇÃO PASSA.
//
// "Uma ação livre é uma escolha consciente, feita no seu turno. Já uma reação é
// uma resposta automática, que pode ocorrer mesmo fora do seu turno" (p233).
func TestCastingOutOfTurnRefusesTheStandardAndAllowsTheReaction(t *testing.T) {
	f := newSceneFixture(t)
	startCombatWithSomeoneElseOnTurn(t, f)

	// `luz` é execução PADRÃO (p197).
	refusal := sceneRefusal(learnAndCast(t, f, "luz").Body)
	if refusal == "" {
		t.Fatal("conjurar uma magia de ação padrão fora da vez passou sem uma palavra na tela")
	}
	if !strings.Contains(refusal, "não é a sua vez") {
		t.Errorf("a recusa diz %q, e quem lê precisa saber que é questão de VEZ", refusal)
	}

	// `queda-suave` é execução REAÇÃO (p203), e o mesmo instante a permite.
	if passed := learnAndCast(t, f, "queda-suave"); sceneRefusal(passed.Body) != "" {
		t.Errorf("a reação foi recusada fora da vez: %q", sceneRefusal(passed.Body))
	}
}

// NA SUA VEZ, A PADRÃO CABE UMA VEZ SÓ: "no seu turno, você pode fazer uma ação
// padrão e uma ação de movimento" (p233). A segunda recusa por CUSTO, e a frase
// é outra — "espere a sua vez" e "não sobrou ação" mandam fazer coisas
// diferentes.
func TestTheSecondStandardSpellOfATurnIsRefusedForLackOfAction(t *testing.T) {
	f := newSceneFixture(t)
	startCombatWithSomeoneElseOnTurn(t, f)
	if _, err := f.s.sessions.NextTurn(f.sessionID); err != nil {
		t.Fatalf("passar a vez ao personagem: %v", err)
	}
	state := f.s.sessions.GetState(f.sessionID)
	who := state.Initiative[state.TurnIndex].CharacterID
	if who == nil || *who != f.charID {
		t.Fatalf("o controle falhou: a vez não é do personagem")
	}

	if rec := learnAndCast(t, f, "luz"); sceneRefusal(rec.Body) != "" {
		t.Fatalf("a primeira padrão da vez foi recusada: %q", sceneRefusal(rec.Body))
	}
	second := sceneRefusal(learnAndCast(t, f, "luz").Body)
	if second == "" {
		t.Fatal("a segunda ação padrão do mesmo turno passou: o turno tem UMA")
	}
	if !strings.Contains(second, "não sobrou ação") {
		t.Errorf("a segunda recusa diz %q, e o motivo é falta de AÇÃO e não de vez", second)
	}
}

// FORA DE UMA CENA DE AÇÃO NADA CUSTA, e por isso nada recusa: numa conversa na
// corte não há turno, e cobrar ali seria aplicar uma regra que o livro não
// aplica (p252). É o caminho de quem abre a ficha sozinho, sem mesa nenhuma.
func TestCastingOutsideAnActionSceneCostsNoAction(t *testing.T) {
	f := newSceneFixture(t)
	if rec := learnAndCast(t, f, "luz"); sceneRefusal(rec.Body) != "" {
		t.Errorf("conjurar fora de cena foi recusado: %q", sceneRefusal(rec.Body))
	}
}

// A CONJURAÇÃO RECUSADA NÃO COBRA A AÇÃO.
//
// A ação é cobrada DEPOIS de a magia sair, e não antes: entre o instante e a
// conjuração ainda há o grimório, a preparação e o PM, e qualquer um deles
// recusa. Cobrar antes tiraria a padrão de alguém por uma magia que nunca saiu
// — e o sintoma seria uma vez perdida sem nada na tela explicando.
func TestARefusedCastDoesNotSpendTheTurnAction(t *testing.T) {
	f := newSceneFixture(t)
	startCombatWithSomeoneElseOnTurn(t, f)
	if _, err := f.s.sessions.NextTurn(f.sessionID); err != nil {
		t.Fatalf("passar a vez ao personagem: %v", err)
	}

	// `luz` sem estar no grimório: o instante permite e a castURLção recusa.
	castURL := fmt.Sprintf("/personagens/%d/magias/conjura/luz?tab=spells", f.charID)
	refusal := sceneRefusal(f.pede(t, f.player, http.MethodPost, castURL, "").Body.String())
	if refusal == "" {
		t.Fatal("o controle falhou: conjurar uma magia fora do grimório passou")
	}
	if strings.Contains(refusal, "sua vez") || strings.Contains(refusal, "sobrou ação") {
		t.Fatalf("a recusa foi do TURNO e não do grimório: %q", refusal)
	}
	if left := f.s.sessions.GetState(f.sessionID).Scene; !left.StandardLeft {
		t.Error("a conjuração recusada cobrou a ação padrão mesmo assim")
	}

	// E a padrão continua inteira: a mesma magia, agora aprendida, sai.
	if rec := learnAndCast(t, f, "luz"); sceneRefusal(rec.Body) != "" {
		t.Errorf("a padrão tinha sido gasta pela recusa: %q", sceneRefusal(rec.Body))
	}
}

// QUEM CAIU NÃO REAGE (decisão do dono, ALE-366): a isenção da p233 — "você
// pode reagir mesmo se não puder realizar ações, como por estar atordoado" — é
// do atordoado. O inconsciente está "sem ações (incluindo reações)" (p395).
func TestAnUnconsciousCharacterCannotReact(t *testing.T) {
	f := newSceneFixture(t)
	startCombatWithSomeoneElseOnTurn(t, f)
	// O CONTROLE: de pé, a mesma reação passa — sem ele, a recusa abaixo
	// poderia vir de qualquer outra coisa.
	if rec := learnAndCast(t, f, "queda-suave"); sceneRefusal(rec.Body) != "" {
		t.Fatalf("o controle falhou: a reação de quem está de pé foi recusada: %q", sceneRefusal(rec.Body))
	}
	if err := f.s.queries.UpdateConditions(t.Context(), sqlcgen.UpdateConditionsParams{
		ActiveConditions: `["inconsciente","sangrando"]`, UpdatedAt: dbvalue.NowISO(), ID: f.charID,
	}); err != nil {
		t.Fatalf("derrubar o personagem: %v", err)
	}
	cast := fmt.Sprintf("/personagens/%d/magias/conjura/queda-suave?tab=spells", f.charID)
	refusal := sceneRefusal(f.pede(t, f.player, http.MethodPost, cast, "").Body.String())
	if refusal == "" {
		t.Fatal("o inconsciente conjurou uma reação: a p395 diz sem ações, incluindo reações")
	}
	if !strings.Contains(refusal, "reação") {
		t.Errorf("a recusa diz %q, e quem lê precisa saber que é a inconsciência tirando a reação", refusal)
	}
}
