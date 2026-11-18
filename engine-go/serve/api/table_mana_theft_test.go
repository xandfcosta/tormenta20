package api

import (
	"fmt"
	"net/http"
	"testing"
)

// O ROUBO DE MANA (p74): "Quando você causa dano com um ataque furtivo, para cada 1d6 de
// dano de seu ataque furtivo, você recebe 1 PM temporário e a criatura perde 1 ponto de
// mana (se tiver). Você só pode usar este poder uma vez por cena contra uma mesma
// criatura."
//
// Ele sai na CONFIRMAÇÃO do mestre, como o Panache: é ela que diz que o golpe causou dano.
// Os dados de furtivo são os que a PROPOSTA guardou, com o interruptor ligado. Roubos
// contra criaturas diferentes na mesma cena SOMAM (decisão do dono).

// thiefAtTheTable é a arcanista da bancada com sete níveis de ladino — 4d6 de furtivo —,
// o Truque Mágico e, se pedido, o Roubo de Mana, numa cena de ação com o Goblin.
func thiefAtTheTable(t *testing.T, withThePower bool) (sceneFixture, string) {
	t.Helper()
	f := newSceneFixture(t)
	seedClasse(t, f.s, f.charID, "Ladino", 7)
	powers := `["class.ladino.truque-magico"]`
	if withThePower {
		powers = `["class.ladino.truque-magico","class.ladino.roubo-de-mana"]`
	}
	choiceCom(t, f, f.charID, powers, `[]`)
	if rec := f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"status":"active"}`); rec.Code != http.StatusOK {
		t.Fatalf("iniciar a sessão deu %d", rec.Code)
	}
	return f, f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 2)
}

func TestTheProposalKeepsTheSneakAttackDice(t *testing.T) {
	f, goblin := thiefAtTheTable(t, true)
	target := fmt.Sprintf("/personagens/%d/efeitos/situacao", f.charID)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, `{"conditional":"flag:ataque-furtivo"}`).Body.String()); refusal != "" {
		t.Fatalf("ligar o Ataque Furtivo: %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, f.charID, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o golpe: %q", refusal)
	}
	pa := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pa == nil {
		t.Fatal("o golpe não deixou provisório")
	}
	if pa.SneakDice != 4 {
		t.Errorf("o golpe com o furtivo ligado guardou %d d6 de furtivo, e o ladino 7 rola 4d6 (p73)", pa.SneakDice)
	}
}
