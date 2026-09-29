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
	// A COMPARAÇÃO tem os dois lados: sem o total de quem se defende, a mesa lê
	// um número sozinho e não tem como julgar o veredicto.
	if pending.Total == 0 || pending.Maneuver.Opposed == 0 {
		t.Errorf("os dois lados do teste oposto vieram %d vs %d, e nenhum pode ser zero "+
			"num d20 somado ao Luta", pending.Total, pending.Maneuver.Opposed)
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
