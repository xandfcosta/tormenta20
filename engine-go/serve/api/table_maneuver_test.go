package api

import (
	"net/http"
	"strings"
	"testing"
)

// A MANOBRA PELA ROTA, e ela divide o provisório com o golpe (ALE-420).
//
// Uma manobra É um ataque corpo a corpo (p234), e o gesto atravessa o mesmo
// caminho: a vez, a posse conferida contra o banco, a ação padrão, e o mesmo
// `PendingAttack` que o mestre confirma. O que muda é a REGRA no meio.
//
// É INTEGRAÇÃO e não unitário de propósito: o que esta fatia acrescentou foi
// COMPOSIÇÃO — uma rota, um ramo no caso de uso e um campo no fio —, e a
// aritmética do teste oposto já está prendida no `maneuver_rules_test.go`.
func TestTheManeuverRidesTheSamePendingAsTheStrike(t *testing.T) {
	f, goblin := attackOnTurn(t)
	derrubar := f.tableUrl() + "/iniciativa/" + goblin + "/manobra/derrubar"

	rec := f.requests(t, f.player, http.MethodPost, derrubar, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor a manobra deu %d", rec.Code)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil {
		t.Fatalf("a manobra permitida não virou provisório: %q", tableRefusal(t, rec.Body.String()))
	}
	if pending.Maneuver == nil {
		t.Fatal("o provisório nasceu sem a conta da manobra — a faixa não teria o que " +
			"escrever, e a mesa leria um golpe que não houve")
	}
	if pending.Maneuver.Kind != "derrubar" {
		t.Errorf("a manobra do provisório é %q e o caminho pediu derrubar", pending.Maneuver.Kind)
	}
	// O DANO NÃO EXISTE numa manobra: ela é "um ataque corpo a corpo para fazer
	// algo diferente de causar dano" (p234). Um dano aqui faria a confirmação
	// tirar PV de quem só foi derrubado.
	if pending.Damage != 0 || len(pending.Dice) != 0 {
		t.Errorf("a manobra veio com %d de dano e %d dados, e a p234 diz que ela não "+
			"causa dano", pending.Damage, len(pending.Dice))
	}
	// A COMPARAÇÃO tem os dois lados: sem o de quem se defende, a mesa lê um
	// número sozinho e não tem como julgar o veredicto.
	//
	// O sentinela é o d20 NATURAL e não o total, e isso é conserto: o total
	// pode ser zero DE VERDADE — o arcanista do caso não tem proficiência com
	// espada longa e ataca com −5 (p142), então um d20 de 5 dá exatamente zero.
	// Com o total como sentinela, este caso reprovava uma vez a cada vinte
	// corridas dizendo que o campo não fora preenchido, e ele fora.
	if pending.Roll < 1 || pending.Roll > 20 {
		t.Errorf("o d20 de quem tentou a manobra veio %d, e um d20 vai de 1 a 20",
			pending.Roll)
	}
	if r := pending.Maneuver.OpposedRoll; r < 1 || r > 20 {
		t.Errorf("o d20 de quem se defendeu veio %d, e um d20 vai de 1 a 20", r)
	}
	// E os totais são a CONTA dos naturais: é isto que prende os dois lados sem
	// depender de nenhum deles ser diferente de zero.
	if pending.Total-pending.Roll == 0 && pending.Maneuver.Opposed-pending.Maneuver.OpposedRoll == 0 {
		t.Error("os dois lados somaram bônus ZERO ao d20, e os dois têm Luta na ficha — " +
			"o `ManeuverSide.Bonus` não chegou")
	}

	// E ELA GASTA A PADRÃO pelo mesmo verbo do golpe: manobra é ação de agredir.
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar a manobra deu %d", rec.Code)
	}
	if stateOf(t, f.s.sessions, f.sessionID).Scene.StandardLeft {
		t.Error("a manobra confirmada não gastou a ação padrão")
	}
}

// A MANOBRA QUE O LIVRO NÃO TEM é recusada pelo SERVIDOR.
//
// O menu da peça oferece as cinco da p234, mas a fronteira não é o menu: o
// caminho aceita qualquer texto, e um `POST .../manobra/voar` chegaria. Travar
// só na tela é UX; a recusa mora aqui.
func TestAManeuverTheBookDoesNotHaveIsRefused(t *testing.T) {
	f, goblin := attackOnTurn(t)

	rec := f.requests(t, f.player, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+goblin+"/manobra/voar", "")
	if stateOf(t, f.s.sessions, f.sessionID).PendingAttack != nil {
		t.Fatal("\"voar\" não é manobra do livro e virou provisório")
	}
	if refusal := tableRefusal(t, rec.Body.String()); !strings.Contains(refusal, "manobra de combate") {
		t.Errorf("a recusa tinha de nomear a lista da p234, e veio %q", refusal)
	}
}

// A MANOBRA CONFIRMADA DEIXA A CONDIÇÃO NO ALVO (T20 p234).
//
// O Goblin do caso é um NPC do bestiário, então a fonte da condição é a LINHA —
// ficha ele não tem. O caminho do personagem tem a ficha como fonte, e é o
// espelho dela que a linha mostra; quem os separa é o `ImposeCondition`.
func TestAConfirmedTakedownLeavesTheTargetProne(t *testing.T) {
	f, goblin := attackOnTurn(t)

	// A MANOBRA PODE PERDER, e o caso não pode depender do dado: ele repete até
	// uma vitória, porque o que se mede é o que acontece DEPOIS dela. Um caso
	// que rolasse uma vez mediria a metade errada em metade das corridas.
	venceu := false
	for range 40 {
		rec := f.requests(t, f.player, http.MethodPost,
			f.tableUrl()+"/iniciativa/"+goblin+"/manobra/derrubar", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("propor a manobra deu %d", rec.Code)
		}
		pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
		if pending == nil || pending.Maneuver == nil {
			t.Fatalf("a manobra não virou provisório: %q", tableRefusal(t, rec.Body.String()))
		}
		if pending.Maneuver.Won {
			if pending.Maneuver.Imposes != "caido" {
				t.Fatalf("o derrubar vencido diz impor %q e a p234 diz caído",
					pending.Maneuver.Imposes)
			}
			venceu = true
			break
		}
		// A ação padrão volta com o cancelamento; sem isso o segundo giro seria
		// recusado por falta de ação e o laço mediria a recusa.
		f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	}
	if !venceu {
		t.Fatal("quarenta tentativas sem uma vitória: o caso não chegou a medir nada")
	}

	// ANTES DA CONFIRMAÇÃO o alvo NÃO está caído — a proposta é um rascunho, e
	// uma condição aplicada nela ficaria no alvo de um gesto que pode ser
	// cancelado.
	if temCondicao(t, f, goblin, "caido") {
		t.Error("o alvo ficou caído com a manobra ainda por confirmar")
	}

	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar a manobra deu %d", rec.Code)
	}
	if !temCondicao(t, f, goblin, "caido") {
		t.Error("a manobra confirmada não deixou o alvo caído")
	}
}

// A MANOBRA QUE NÃO IMPÕE NADA não inventa condição.
//
// O desarmar derruba um ITEM, e o livro não lhe dá condição. Sem este caso, um
// `Imposes` preenchido para as cinco passaria — e o alvo de um desarmar
// apareceria caído na mesa.
func TestADisarmLeavesNoConditionBehind(t *testing.T) {
	f, goblin := attackOnTurn(t)

	rec := f.requests(t, f.player, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+goblin+"/manobra/desarmar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor o desarmar deu %d", rec.Code)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending.Maneuver.Imposes != "" {
		t.Errorf("o desarmar diz impor %q, e a p234 lhe dá efeito de ITEM — o item cai, "+
			"a criatura não ganha condição", pending.Maneuver.Imposes)
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if len(condicoesDe(t, f, goblin)) != 0 {
		t.Errorf("o desarmar confirmado deixou %v no alvo", condicoesDe(t, f, goblin))
	}
}

func condicoesDe(t *testing.T, f sceneFixture, entryID string) []string {
	t.Helper()
	state := stateOf(t, f.s.sessions, f.sessionID)
	for _, e := range state.Initiative {
		if e.ID == entryID {
			return e.Conditions
		}
	}
	t.Fatalf("a linha %q sumiu da fila", entryID)
	return nil
}

func temCondicao(t *testing.T, f sceneFixture, entryID, condition string) bool {
	t.Helper()
	for _, c := range condicoesDe(t, f, entryID) {
		if c == condition {
			return true
		}
	}
	return false
}
