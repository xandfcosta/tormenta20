package api

import (
	"net/http"
	"testing"
)

// O GOLPE NUMA FICHA OBEDECE AO CORPO DELA (ALE-650). O kliren "sofre 1 ponto de dano adicional por dado de
// dano de impacto" (p28): a clava (1d6 de impacto) que acerta traz +1, e o mestre o lê na conta.
//
// INTEGRAÇÃO porque o que quebra é a COMPOSIÇÃO: a ficha do alvo vira combatente, o corpo dela vai para o
// alvo do motor, e o número chega ao provisório que o ✓ do mestre pousa no PV.
func TestTheClubHitsTheKlirenOnePointHarderPerDie(t *testing.T) {
	f := newSceneFixture(t)
	itemSemeia(t, f, f.charID, "clava", "Clava", "wielded")
	f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 2)
	kliren := seedCharacter(t, f.s, f.gm, "Engrenagem")
	seedRaca(t, f.s, kliren, "kliren")
	alvo := umPersonagemNaFila(t, f, kliren, "Engrenagem")

	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+alvo+"/atacar", "")
	if pendingOf(t, f) == nil {
		t.Fatal("o golpe no kliren não virou provisório")
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/julgar/acertou", ""); rec.Code != http.StatusOK {
		t.Fatalf("decidir acertou deu %d", rec.Code)
	}
	if rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/dano/na-mao", `{"damage_die_1":4}`); rec.Code != http.StatusOK {
		t.Fatalf("rolar o dano deu %d", rec.Code)
	}
	pa := pendingOf(t, f)
	if pa.Damage != pa.RawDamage+1 {
		t.Errorf("a clava no kliren causou %d com o cru em %d: os Ossos Frágeis somam +1 pelo dado de impacto", pa.Damage, pa.RawDamage)
	}
	if len(pa.Taken) != 1 || pa.Taken[0].Label != "Ossos Frágeis" || pa.Taken[0].Amount != 1 {
		t.Errorf("a conta não diz de onde veio o ponto a mais: %+v", pa.Taken)
	}
}
