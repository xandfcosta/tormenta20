package api

import (
	"net/http"
	"strings"
	"testing"
)

// A LINHA DO DANO NA MESA, PELO GESTO (ALE-655). Um lutador com a Chave ataca desarmado e o mestre diz acerto: o
// passo do dano oferece a linha "esmagando quem agarra", quem atacou a escolhe, e o "rolei na mão" passa a pedir
// o dado dela, um passo acima. A conta da linha é do motor e da camada de combate; aqui se prende a ROTA e a TELA.
func TestTheHoldDamageLineIsChosenOnTheTableAndRolledByHand(t *testing.T) {
	f := newSceneFixture(t)
	goblin := f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 2)
	if rec := f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"status":"active"}`); rec.Code != http.StatusOK {
		t.Fatalf("iniciar a sessão deu %d", rec.Code)
	}
	asClass(t, f, "Lutador")
	withPowers(t, f, `["class.lutador.chave"]`, "{}")
	if refusal := attacks(t, f, goblin, ""); refusal != "" {
		t.Fatalf("o golpe desarmado foi recusado: %q", refusal)
	}
	gmRules(t, f, "acertou")
	pa := pendingOf(t, f)
	if pa == nil || pa.DamageToRoll == nil || len(pa.DamageToRoll.Lines) != 2 {
		t.Fatalf("o acerto não ofereceu as linhas do desarmado: %+v", pa)
	}
	normal := pa.DamageToRoll.Dice[0].Faces

	player := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(player, "esmagando quem agarra") {
		t.Error("o passo do dano de quem atacou não mostra a linha da Chave")
	}

	rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/dano/linha/class.lutador.chave", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("escolher a linha da Chave deu %d", rec.Code)
	}
	pa = pendingOf(t, f)
	if pa.DamageToRoll.Line != "class.lutador.chave" || pa.DamageToRoll.Dice[0].Faces <= normal {
		t.Fatalf("depois da escolha, o dano pede %+v; want a Chave, um passo acima do d%d", pa.DamageToRoll, normal)
	}

	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/dano/na-mao", `{"damage_die_1":1}`)
	if pa = pendingOf(t, f); !pa.DamageRolled || pa.Faces != pa.DamageToRoll.Dice[0].Faces {
		t.Errorf("o dano na mão rolou d%d (rolado: %v); want o dado da Chave, d%d", pa.Faces, pa.DamageRolled, pa.DamageToRoll.Dice[0].Faces)
	}
}
