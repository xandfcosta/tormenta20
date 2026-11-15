package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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

// O MESTRE LÊ COMO O DANO ATINGE A FICHA onde ele aplica o dano: na folha do combatente, ao lado do editor de
// PV (ALE-650). Quase todo dano chega pela mão dele, sem tipo; o app não tem como aplicar ali o +1 por dado, e
// por isso o diz. O JOGADOR não lê a folha de outra entidade.
func TestTheGMReadsHowTheDamageHitsTheKliren(t *testing.T) {
	f := newSceneFixture(t)
	f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 2)
	kliren := seedCharacter(t, f.s, f.gm, "Engrenagem")
	seedRaca(t, f.s, kliren, "kliren")
	umPersonagemNaFila(t, f, kliren, "Engrenagem")

	gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if want := "+1 por dado de impacto (Ossos Frágeis)"; !strings.Contains(gm, want) {
		t.Errorf("a folha do combatente não diz ao mestre %q", want)
	}
	player := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(player, "Ossos Frágeis") {
		t.Error("o jogador leu como o dano atinge outra ficha")
	}
}

// A BOLA DE FOGO NO GOLEM DO FOGO: a faixa do mestre diz que ele é imune e que o dano mágico de fogo o cura
// (Fonte Elemental, p27). A bola de fogo da Flamejante é mágica, e quem aplica é o mestre.
func TestTheBlastBandTellsTheGMTheFireGolemIsImmune(t *testing.T) {
	f, barbaro, _ := barbarianOnTurn(t)
	sword := itemSemeia(t, f, barbaro, "espada-longa", "Espada longa", "wielded")
	if err := f.s.characterPlays().EnchantItem(context.Background(), sword, []string{"encanto-flamejante"}, nil); err != nil {
		t.Fatalf("encantar a espada: %v", err)
	}
	golem := seedCharacter(t, f.s, f.gm, "Fornalha")
	seedRaca(t, f.s, golem, "golem")
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET raceAbilityChoices = '["golem-fonte-fogo"]' WHERE id = ?`, golem); err != nil {
		t.Fatalf("escolher a fonte do fogo: %v", err)
	}
	alvo := umPersonagemNaFila(t, f, golem, "Fornalha")
	if refusal := sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/bola-de-fogo/%d?embutida=1", barbaro, sword),
		fmt.Sprintf(`{"turn_target":%q,"blast_total":21}`, alvo))); refusal != "" {
		t.Fatalf("a bola de fogo no golem foi recusada: %q", refusal)
	}

	gm := entre(t, f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(), `id="table-blast"`, "</section>")
	for _, want := range []string{"imune a fogo (Fonte Elemental)", "dano mágico de fogo cura metade"} {
		if !strings.Contains(gm, want) {
			t.Errorf("a faixa do mestre não diz %q:\n%s", want, gm)
		}
	}
	player := entre(t, f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(), `id="table-blast"`, "</section>")
	if strings.Contains(player, "Fonte Elemental") {
		t.Errorf("a faixa do jogador contou a ficha do alvo:\n%s", player)
	}
}
