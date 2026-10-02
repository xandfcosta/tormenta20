package api

import (
	"net/http"
	"testing"
)

// TROCAR O TIPO DE DANO, pela mesa (ALE-423, p236).
//
//	"Você pode usar uma arma para causar dano não letal [...], mas sofre uma
//	 penalidade de –5 no teste de ataque. [...] Você pode usar esses ataques e
//	 armas para causar dano letal, mas sofre a mesma penalidade."
//
// INTEGRAÇÃO porque o que pode quebrar é a COMPOSIÇÃO: o botão do menu bate numa
// rota irmã, a cena põe uma flag no pedido, o `Strike` a junta às situações que
// vêm do TABULEIRO, e o motor lê a lista para duas coisas diferentes — o −5 e o
// tipo do dano. Um teste do `ResolveAttackUnder` prova a regra e nada sobre o
// gesto chegar nela.

// propoeGolpe rola um golpe pela rota pedida e devolve o provisório, cancelando
// em seguida para a mesa ficar limpa para a próxima rolagem.
func propoeGolpe(t *testing.T, f sceneFixture, alvo, sufixo string) (roll, total, naoLetal, dano int) {
	t.Helper()
	rec := f.requests(t, f.player, http.MethodPost,
		f.tableUrl()+"/iniciativa/"+alvo+"/atacar"+sufixo, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor o golpe %q deu %d", sufixo, rec.Code)
	}
	pendente := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pendente == nil {
		t.Fatalf("o golpe %q não virou provisório: %q", sufixo, tableRefusal(t, rec.Body.String()))
	}
	out := *pendente
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("cancelar deu %d", rec.Code)
	}
	return out.Roll, out.Total, out.NonLethal, out.Damage
}

// O −5 E A TROCA, medidos no MESMO d20.
//
// O bônus de ataque é `total − roll`, e é ele que a comparação usa: o d20 é
// rolado pelo servidor e muda entre as duas chamadas, então comparar os totais
// crus mediria o dado e não a regra.
func TestSwitchingTheDamageTypeCostsFiveAndFlipsTheDamage(t *testing.T) {
	f, goblin := attackOnTurn(t)

	rollNu, totalNu, naoLetalNu, _ := propoeGolpe(t, f, goblin, "")
	// O CONTROLE: a espada longa nua causa dano letal e não paga nada.
	if naoLetalNu != 0 {
		t.Fatalf("o controle já estava errado: a espada nua causou %d de dano não letal",
			naoLetalNu)
	}

	rollTrocado, totalTrocado, naoLetalTrocado, danoTrocado := propoeGolpe(t, f, goblin, "/trocando")
	bonusNu, bonusTrocado := totalNu-rollNu, totalTrocado-rollTrocado
	if bonusTrocado != bonusNu-5 {
		t.Errorf("o golpe trocado somou bônus %d e o nu somou %d — a p236 tira 5",
			bonusTrocado, bonusNu)
	}
	if danoTrocado > 0 && naoLetalTrocado != danoTrocado {
		t.Errorf("o golpe trocado causou %d de dano e só %d dele é não letal — pagar os "+
			"−5 e matar do mesmo jeito é o pior dos dois mundos",
			danoTrocado, naoLetalTrocado)
	}
}
