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

// A MANOBRA CONFIRMADA **NÃO** APLICA A CONDIÇÃO — ela a ANUNCIA.
//
// Este caso nasceu afirmando o contrário: o motor decidia quem vencia o embate e
// a confirmação punha "caído" no alvo sozinha. A regra que o derrubou está na
// seção "O sistema INFORMA; o mestre DECIDE" do `CLAUDE.md` da raiz — e é por
// isso que o caso ficou e teve o sinal invertido em vez de ser apagado: o que ele
// protege agora é a AUSÊNCIA do automatismo, que é mais fácil de perder de vista
// que a presença dele.
//
// O Goblin é um NPC do bestiário, e portanto a condição dele moraria na LINHA da
// iniciativa — o caminho mais curto que o automatismo tinha. Se alguma coisa
// voltar a aplicar sozinha, é aqui que aparece.
func TestAConfirmedManeuverAnnouncesTheConditionAndLeavesItToTheGameMaster(t *testing.T) {
	f, goblin := attackOnTurn(t)

	// UM GIRO SÓ, e isto é a medida da mudança: o caso antigo rolava até QUARENTA
	// vezes procurando uma vitória, porque só a vitória produzia a condição. Hoje
	// o dado não decide o que a faixa diz, então não há o que procurar.
	rec := f.requests(t, f.player, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+goblin+"/manobra/derrubar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor a manobra deu %d", rec.Code)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil || pending.Maneuver == nil {
		t.Fatalf("a manobra não virou provisório: %q", tableRefusal(t, rec.Body.String()))
	}
	if pending.Maneuver.ConditionOnAWin != "caido" {
		t.Fatalf("o derrubar anuncia %q e a p234 diz caído — a faixa existe para "+
			"poupar o mestre de ir ao livro", pending.Maneuver.ConditionOnAWin)
	}
	if temCondicao(t, f, goblin, "caido") {
		t.Error("o alvo ficou caído com a manobra ainda por confirmar")
	}

	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar a manobra deu %d", rec.Code)
	}
	// A CONFIRMAÇÃO GASTA A AÇÃO E REGISTRA A ROLAGEM, e para aí. Quem decide se
	// o goblin caiu é quem está narrando — e ele tem o gesto de condição para
	// dizê-lo, que é por onde isto passa a ser uma escolha e não um efeito.
	if caidas := condicoesDe(t, f, goblin); len(caidas) != 0 {
		t.Errorf("a manobra confirmada aplicou %v sozinha: o desfecho do embate é do "+
			"mestre, e a faixa só lhe diz o que o livro prevê", caidas)
	}
}

// AS MANOBRAS QUE O LIVRO NÃO PREMIA COM CONDIÇÃO não inventam uma.
//
// O desarmar derruba um ITEM e o empurrar move, e a p234 não lhes dá condição.
// Sem este caso, um `ConditionOnAWin` preenchido para as cinco mandaria o mestre
// deitar o alvo de um desarmar.
func TestTheManeuversTheBookGivesNoConditionAnnounceNone(t *testing.T) {
	for _, manobra := range []string{"desarmar", "empurrar"} {
		// UMA BANCADA POR MANOBRA: confirmar gasta a ação padrão do turno, e um
		// segundo giro na mesma mesa mediria a recusa por falta de ação.
		f, goblin := attackOnTurn(t)
		rec := f.requests(t, f.player, http.MethodPost,
			f.tableUrl()+"/iniciativa/"+goblin+"/manobra/"+manobra, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("propor o %s deu %d", manobra, rec.Code)
		}
		pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
		if pending.Maneuver.ConditionOnAWin != "" {
			t.Errorf("o %s anuncia a condição %q, e a p234 lhe dá efeito de ITEM ou de "+
				"MOVIMENTO", manobra, pending.Maneuver.ConditionOnAWin)
		}
		if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
			t.Fatalf("confirmar o %s deu %d", manobra, rec.Code)
		}
		if len(condicoesDe(t, f, goblin)) != 0 {
			t.Errorf("o %s confirmado deixou %v no alvo", manobra, condicoesDe(t, f, goblin))
		}
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
