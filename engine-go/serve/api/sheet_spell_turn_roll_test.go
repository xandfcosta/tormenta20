package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"t20engine/domain/live"
	"t20engine/domain/sheet"
)

// O DADO DA VEZ (ver GLOSSARY, **dado da vez**): a magia em vigor rola de novo, ao apertar, na linha dela da aba
// Efeitos, com os aprimoramentos escolhidos ao conjurar, e a faixa da magia o informa dizendo o momento.

// turnCasterAtTheTable é uma mesa ao vivo cuja ficha do jogador é um conjurador de 20º nível da classe pedida: as
// magias com dado da vez são do 3º e do 4º círculos.
func turnCasterAtTheTable(t *testing.T, class string) sceneFixture {
	t.Helper()
	f := umaMesaAoVivo(t)
	f.charID = seedCharacterAtLevel(t, f.s, f.player, "Conjurador da Vez", class, 20, 0, 0)
	seedMember(t, f.s, f.campaignID, f.charID)
	return f
}

// effectOf é a linha em vigor da magia na ficha carregada, com os aprimoramentos da conjuração.
func effectOf(t *testing.T, f sceneFixture, spellID string) (sheet.EffectDTO, bool) {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("ler a ficha: %v", err)
	}
	dto, err := f.s.sheetRules().LoadCharacter(context.Background(), row)
	if err != nil {
		t.Fatalf("carregar a ficha: %v", err)
	}
	for _, e := range dto.ActiveEffects {
		if e.CatalogID == spellID {
			return e, true
		}
	}
	return sheet.EffectDTO{}, false
}

// rollTurn aperta o dado da vez da linha em vigor da magia, e devolve a recusa.
func rollTurn(t *testing.T, f sceneFixture, spellID, version, signals string) string {
	t.Helper()
	effect, open := effectOf(t, f, spellID)
	if !open {
		t.Fatalf("%s não está em vigor na ficha", spellID)
	}
	target := fmt.Sprintf("/personagens/%d/efeitos/vez/%d?tab=conditionals", f.charID, effect.ID)
	if version != "" {
		target += "&versao=" + url.QueryEscape(version)
	}
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, signals).Body.String())
}

func TestTheTurnRollRollsWithTheAugmentsOfTheCast(t *testing.T) {
	f := turnCasterAtTheTable(t, "Arcanista")
	if refusal, _ := castWith(t, f, "ferver-sangue", `{"augment0":2}`); refusal != "" {
		t.Fatalf("conjurar o Ferver Sangue com dois aumentos foi recusado: %q", refusal)
	}
	effect, open := effectOf(t, f, "ferver-sangue")
	if !open || len(effect.CastAugments) != 1 || effect.CastAugments[0] != (sheet.AugmentPick{AugmentIndex: 0, Stacks: 2}) {
		t.Fatalf("a linha em vigor guardou %+v (aberta %v); want o aumento ×2 da conjuração", effect.CastAugments, open)
	}

	if refusal := rollTurn(t, f, "ferver-sangue", "", `{}`); refusal != "" {
		t.Fatalf("o dado da vez foi recusado: %q", refusal)
	}
	cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast
	if cast == nil || cast.Spell != "Ferver Sangue" || cast.Kind != live.RollOfTheTurn || cast.Moment != "inicio" {
		t.Fatalf("a faixa da magia diz %+v; want o Ferver Sangue no início do turno", cast)
	}
	if len(cast.Parts) != 1 || cast.Parts[0].Dice != "6d8" || cast.Parts[0].Kind != "fogo" || len(cast.Parts[0].Rolls) != 6 {
		t.Errorf("o dado da vez rolou %+v; want 6d8 de fogo, com os dois aumentos da conjuração (p193)", cast.Parts)
	}
	page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(page, "Ferver Sangue · início do turno") {
		t.Error("a faixa da mesa não diz que o Ferver Sangue rolou no início do turno")
	}

	t.Run("o total rolado na mão chega à faixa, e o que os dados não tiram é recusado", func(t *testing.T) {
		if refusal := rollTurn(t, f, "ferver-sangue", "", `{"spell_hand_1":30}`); refusal != "" {
			t.Fatalf("os 30 da mesa foram recusados: %q", refusal)
		}
		if part := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast.Parts[0]; part.Total != 30 || !part.ByHand {
			t.Errorf("a faixa diz %d, à mão %v; want os 30 da mesa", part.Total, part.ByHand)
		}
		if refusal := rollTurn(t, f, "ferver-sangue", "", `{"spell_hand_1":60}`); !strings.Contains(refusal, "de 6 a 48") {
			t.Errorf("os 60 na mão saíram com a recusa %q", refusal)
		}
	})

	t.Run("o dado da vez não cobra PM", func(t *testing.T) {
		before := mpOf(t, f.s, f.charID)
		if refusal := rollTurn(t, f, "ferver-sangue", "", `{}`); refusal != "" {
			t.Fatalf("o dado da vez foi recusado: %q", refusal)
		}
		if after := mpOf(t, f.s, f.charID); after != before {
			t.Errorf("o dado da vez cobrou: PM %d → %d", before, after)
		}
	})

	t.Run("encerrar a magia leva junto os aprimoramentos da conjuração", func(t *testing.T) {
		target := fmt.Sprintf("/personagens/%d/efeitos/encerra/%d?tab=conditionals", f.charID, effect.ID)
		if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, `{}`).Body.String()); refusal != "" {
			t.Fatalf("encerrar o Ferver Sangue foi recusado: %q", refusal)
		}
		var left int
		if err := f.s.db.QueryRow(`SELECT COUNT(*) FROM effect_cast_augments`).Scan(&left); err != nil || left != 0 {
			t.Errorf("sobraram %d aprimoramentos de conjuração depois de encerrar (%v)", left, err)
		}
		gone := fmt.Sprintf("/personagens/%d/efeitos/vez/%d?tab=conditionals", f.charID, effect.ID)
		if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, gone, `{}`).Body.String()); refusal == "" {
			t.Error("o dado da vez da magia encerrada rolou")
		}
	})
}

// A POEIRA DA PODRIDÃO dura a cena e não é sustentada (p201): conjurá-la abre uma linha em vigor de cena, e é dela
// que o dado da vez sai.
func TestTheSceneSpellOpensAnEffectThatRollsOnTheTurn(t *testing.T) {
	f := turnCasterAtTheTable(t, "Clérigo")
	if refusal, _ := castWith(t, f, "poeira-da-podridao", `{"augment0":1}`); refusal != "" {
		t.Fatalf("conjurar a Poeira da Podridão foi recusado: %q", refusal)
	}
	effect, open := effectOf(t, f, "poeira-da-podridao")
	if !open || effect.Scope != "scene" {
		t.Fatalf("a Poeira da Podridão abriu %+v (aberta %v); want uma linha em vigor de cena", effect, open)
	}
	if refusal := rollTurn(t, f, "poeira-da-podridao", "", `{}`); refusal != "" {
		t.Fatalf("o dado da vez foi recusado: %q", refusal)
	}
	if cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast; cast.Parts[0].Dice != "3d8+12" || cast.Parts[0].Kind != "trevas" {
		t.Errorf("a Poeira rolou %+v; want 3d8+12 de trevas, com o aumento da conjuração", cast.Parts)
	}
}

// O MANTO DO CRUZADO rola a versão que quem joga aperta: a ficha não guarda qual ele aprendeu (decisão do dono).
func TestTheMantleRollsTheVersionPressed(t *testing.T) {
	f := turnCasterAtTheTable(t, "Clérigo")
	if refusal, _ := castWith(t, f, "manto-do-cruzado", `{}`); refusal != "" {
		t.Fatalf("conjurar o Manto do Cruzado foi recusado: %q", refusal)
	}
	if refusal := rollTurn(t, f, "manto-do-cruzado", "Manto de Trevas", `{}`); refusal != "" {
		t.Fatalf("o Manto de Trevas foi recusado: %q", refusal)
	}
	if cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast; cast.Spell != "Manto do Cruzado (Manto de Trevas)" || cast.Parts[0].Dice != "4d8" {
		t.Errorf("o Manto rolou %q %+v; want o de Trevas, 4d8", cast.Spell, cast.Parts)
	}
	if refusal := rollTurn(t, f, "manto-do-cruzado", "", `{}`); !strings.Contains(refusal, "diga qual") {
		t.Errorf("o Manto sem versão saiu com a recusa %q", refusal)
	}
}

// A LINHA EM VIGOR diz o dado da vez e quando ele vale, e o campo do total rolado na mão.
func TestTheEffectsTabOffersTheTurnRoll(t *testing.T) {
	f := turnCasterAtTheTable(t, "Arcanista")
	if refusal, _ := castWith(t, f, "ferver-sangue", `{"augment0":1}`); refusal != "" {
		t.Fatalf("conjurar o Ferver Sangue foi recusado: %q", refusal)
	}
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=conditionals", f.charID), "").Body.String()
	for _, want := range []string{"5d8 de fogo", "início do turno", `aria-label="Rolar Ferver Sangue"`, `data-bind="spell_hand_1"`} {
		if !strings.Contains(page, want) {
			t.Errorf("a aba Efeitos não tem %q na linha do Ferver Sangue", want)
		}
	}
}
