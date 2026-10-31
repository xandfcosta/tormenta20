package api

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O ATAQUE NO TURNO: agredir é ação PADRÃO (p233), e quem não pode agir não
// ataca. Estes casos prendem a ligação entre o gesto de atacar e a economia de
// ação que a ALE-365 pôs no turno — a regra da troca mora no
// `engine.TurnBudget` e a do instante no `engine.MomentFor`.

// attackOnTurn monta a mesa com um Goblin do bestiário na frente e o
// personagem, de espada na mão, com a vez. Devolve a linha do Goblin, que é o
// alvo — do BESTIÁRIO porque é ele que traz a Defesa que o ataque mede.
func attackOnTurn(t *testing.T) (sceneFixture, string) {
	t.Helper()
	f := newSceneFixture(t)
	itemSemeia(t, f, f.charID, "espada-longa", "Espada longa", "wielded")
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
	if _, err := store.AddInitiativeEntry(context.Background(), f.sessionID, sheetCombatant("Arcanista", 5, f.charID)); err != nil {
		t.Fatalf("pôr o personagem na fila: %v", err)
	}
	for range 2 { // a primeira vez é do Goblin, a segunda do personagem
		if _, err := store.NextTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}
	state := stateOf(t, store, f.sessionID)
	if state.Initiative[state.TurnIndex].CharacterID == nil {
		t.Fatalf("a vez tinha de ser do personagem, e é de %q", state.Initiative[state.TurnIndex].Label)
	}
	for _, e := range state.Initiative {
		if e.Label == "Goblin" {
			return f, e.ID
		}
	}
	t.Fatal("o Goblin não está na fila")
	return f, ""
}

var commandErrorSignal = regexp.MustCompile(`"command_error":("(?:[^"\\]|\\.)*")`)

// tableRefusal lê a frase que a mesa acendeu no rodapé. Falha alto quando o
// sinal não veio: sem ele, "não recusou" e "o canal não existe" são a mesma
// string vazia.
func tableRefusal(t *testing.T, body string) string {
	t.Helper()
	found := commandErrorSignal.FindStringSubmatch(body)
	if found == nil {
		t.Fatalf("a resposta não trouxe o sinal command_error: %q", body[:min(220, len(body))])
	}
	var sentence string
	if err := json.Unmarshal([]byte(found[1]), &sentence); err != nil {
		t.Fatalf("o command_error não é uma string JSON: %v", err)
	}
	return sentence
}

func TestAttackingSpendsTheStandardAction(t *testing.T) {
	f, goblin := attackOnTurn(t)
	attack := f.tableUrl() + "/iniciativa/" + goblin + "/atacar"

	rec := f.requests(t, f.player, http.MethodPost, attack, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor o ataque deu %d", rec.Code)
	}
	// O CONTROLE: de pé, na vez e armado, o provisório NASCE. Sem isto, um
	// ataque recusado por qualquer outro motivo passaria pelos casos abaixo.
	if stateOf(t, f.s.sessions, f.sessionID).PendingAttack == nil {
		t.Fatalf("o ataque permitido não virou provisório: %q", tableRefusal(t, rec.Body.String()))
	}
	// PROPOR NÃO COBRA: o provisório ainda pode ser cancelado, e um ataque
	// cancelado não aconteceu.
	if scene := stateOf(t, f.s.sessions, f.sessionID).Scene; !scene.StandardLeft {
		t.Fatal("propor o ataque já gastou a ação padrão, antes de o mestre confirmar")
	}
	if rec := settlesAsTheBookSuggests(t, f); rec.Code != http.StatusOK {
		t.Fatalf("encerrar o ataque deu %d", rec.Code)
	}
	scene := stateOf(t, f.s.sessions, f.sessionID).Scene
	if scene.StandardLeft {
		t.Error("o ataque confirmado não gastou a ação padrão")
	}
	if !scene.MovementLeft {
		t.Error("atacar não toca na ação de movimento")
	}

	// O SEGUNDO ATAQUE NÃO SAI: a padrão acabou e a troca da p233 é de mão
	// única — movimento não vira padrão. O que se prende é o PROVISÓRIO, que
	// não pode nascer.
	rec = f.requests(t, f.player, http.MethodPost, attack, "")
	// O ERRO DECIDIDO fica na mesa para o jogador ler; um segundo ataque seria
	// um provisório NOVO, ainda sem decisão.
	if pa := stateOf(t, f.s.sessions, f.sessionID).PendingAttack; pa != nil && pa.Ruling == "" {
		t.Error("sem ação padrão no turno, um segundo ataque foi rolado")
	}
	if refusal := tableRefusal(t, rec.Body.String()); !strings.Contains(refusal, "não sobrou ação neste turno") {
		t.Errorf("o segundo ataque tinha de ser recusado por falta de ação, e veio %q", refusal)
	}
}

func TestAStunnedCharacterDoesNotAttack(t *testing.T) {
	f, goblin := attackOnTurn(t)
	var pc string
	for _, e := range stateOf(t, f.s.sessions, f.sessionID).Initiative {
		if e.CharacterID != nil {
			pc = e.ID
		}
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/"+pc+"/condicao/atordoado", ""); rec.Code != http.StatusOK {
		t.Fatalf("marcar Atordoado deu %d", rec.Code)
	}
	rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/atacar", "")
	if stateOf(t, f.s.sessions, f.sessionID).PendingAttack != nil {
		t.Error("atordoado, o personagem rolou um ataque")
	}
	if refusal := tableRefusal(t, rec.Body.String()); !strings.Contains(refusal, "não dá para agir") {
		t.Errorf("a recusa tinha de dizer que não dá para agir, e veio %q", refusal)
	}
}

// A FAIXA DIZ O QUE O LIVRO RESERVA AO GOLPE, e nada sobre o desfecho (ALE-441).
//
// "Uma criatura viva atingida fica sangrando" (p336). O app não aplica nada: ele
// põe a condição PREVISTA na frente do mestre, que é quem a marca na fila — o
// mesmo desenho da manobra (ALE-423).
//
// O caso é de INTEGRAÇÃO porque o trajeto é o que importa: catálogo → carta da
// arma → resultado do ataque → provisório → faixa. Um teste sobre a frase
// sozinha passaria verde com o encanto inerte.
func TestTheAttackBannerSaysTheConditionTheBookForesees(t *testing.T) {
	f, goblin := attackOnTurn(t)
	itens, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("ler os itens: %v", err)
	}
	espada := int64(0)
	for _, item := range itens {
		if item.Name == "Espada longa" {
			espada = item.ID
		}
	}
	if espada == 0 {
		t.Fatal("o personagem da bancada não está com a espada longa")
	}
	if err := f.s.characterPlays().EnchantItem(
		context.Background(), espada, []string{"encanto-sanguinaria"},
	); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}

	rec := f.requests(t, f.player, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+goblin+"/atacar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor o ataque deu %d", rec.Code)
	}
	pendente := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pendente == nil {
		t.Fatalf("o ataque não virou provisório: %q", tableRefusal(t, rec.Body.String()))
	}
	// O ANÚNCIO SAI COM O DANO, depois de o mestre dizer acertou — e é ele quem
	// diz, qualquer que seja o d20. Decidir aqui é o que mantém o caso
	// determinístico.
	if len(pendente.Announces) != 0 {
		t.Errorf("o teste do ataque já anuncia %+v, antes da decisão do mestre", pendente.Announces)
	}
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/julgar/acertou", "")
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/dano", "")
	pendente = stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if len(pendente.Announces) != 1 || pendente.Announces[0].Condition != "sangrando" {
		t.Fatalf("o acerto com a Sanguinária anuncia %+v, e a p336 reserva sangrando",
			pendente.Announces)
	}
	// A TELA DO MESTRE: a consequência que o livro prevê é leitura dele, e a
	// faixa do jogador não a traz (ALE-532).
	tela := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(tela, "o livro prevê") || !strings.Contains(tela, "Sangrando") {
		t.Errorf("a faixa não diz a condição prevista:\n%s", recorte(tela))
	}
	// E NENHUM VEREDICTO: a faixa não pode dizer que a criatura FICOU sangrando.
	if strings.Contains(tela, "fica sangrando") || strings.Contains(tela, "ficou sangrando") {
		t.Error("a faixa declarou o desfecho, e quem decide é o mestre")
	}
}
