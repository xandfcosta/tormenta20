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
// tipo do dano. Um teste do `RollAttackDamage` prova a regra e nada sobre o
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

// O Estilo Desarmado: "podem causar dano letal ou não letal (sem penalidades)" (p128). Vale para o
// ataque desarmado, e a arma na mão de quem tem o poder segue pagando os −5.
func TestUnarmedStyleSwitchesTheDamageTypeWithoutPenalty(t *testing.T) {
	f, goblin := attackOnTurn(t)
	seedPowerChoices(t, f.s, f.charID, `["estilo-desarmado"]`, "{}")

	// O CONTROLE: com a espada na mão, o poder não muda nada.
	rollNu, totalNu, _, _ := propoeGolpe(t, f, goblin, "")
	rollTrocado, totalTrocado, _, _ := propoeGolpe(t, f, goblin, "/trocando")
	if totalTrocado-rollTrocado != totalNu-rollNu-5 {
		t.Fatalf("o controle já estava errado: com a espada, o golpe trocado somou %d e o nu, %d",
			totalTrocado-rollTrocado, totalNu-rollNu)
	}

	if _, err := f.s.db.ExecContext(t.Context(), "UPDATE character_items SET equipped = NULL WHERE characterId = ?", f.charID); err != nil {
		t.Fatalf("guardar a espada: %v", err)
	}
	rollNu, totalNu, _, _ = propoeGolpe(t, f, goblin, "")
	rollTrocado, totalTrocado, _, _ = propoeGolpe(t, f, goblin, "/trocando")
	if totalTrocado-rollTrocado != totalNu-rollNu {
		t.Errorf("desarmado, o golpe trocado somou %d e o nu, %d: a p128 diz sem penalidades",
			totalTrocado-rollTrocado, totalNu-rollNu)
	}

	// A TROCA SEGUE VALENDO, agora sem os −5: o dano sai do outro tipo. O nu do Estilo é letal.
	rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/atacar/trocando", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("propor o golpe trocado deu %d", rec.Code)
	}
	gmRules(t, f, "acertou")
	if refusal := gmRefusal(t, f, "/ataque/dano"); refusal != "" {
		t.Fatalf("o dano foi recusado: %q", refusal)
	}
	pa := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pa == nil || pa.Damage == 0 || pa.NonLethal != pa.Damage {
		t.Errorf("o golpe trocado do Estilo Desarmado devia ser não letal: %+v", pa)
	}
}
