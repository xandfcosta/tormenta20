package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
	"t20engine/domain/sheet"
)

// AS AÇÕES DA MAGIA (ver GLOSSARY, **ação da magia**): a magia em vigor dá a quem a conjurou opções que custam uma
// ação da vez; o gesto mora na aba Ações, no grupo da ação, rola o dado com os aprimoramentos da conjuração, e a
// faixa da magia o informa com o nome da opção.

// spellAct aperta a opção `option` da magia em vigor, pela rota da aba Ações, e devolve a recusa.
func spellAct(t *testing.T, f sceneFixture, spellID, option, signals string) string {
	t.Helper()
	effect, open := effectOf(t, f, spellID)
	if !open {
		t.Fatalf("%s não está em vigor na ficha", spellID)
	}
	actions := sheet.CasterOf(sheet.CharacterDTO{}).Spell(spellID).Actions(effect.CastAugments)
	index := -1
	for i, o := range actions.Options {
		if o.Name == option {
			index = i
		}
	}
	if index < 0 {
		t.Fatalf("%s não tem a ação %q", spellID, option)
	}
	target := fmt.Sprintf("/personagens/%d/acoes/magia/%d/%d?tab=combat", f.charID, effect.ID, index)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, signals).Body.String())
}

// onTurnInAnActionScene põe a cena de ação no ar com a vez do conjurador.
func onTurnInAnActionScene(t *testing.T, f sceneFixture) {
	t.Helper()
	f.actionSceneWithTheGoblinFirst(t, "Conjurador da Vez", f.charID, 2)
}

func TestTheMightyHandCrushesWithTheAugmentsOfTheCastAndSpendsTheMovement(t *testing.T) {
	f := turnCasterAtTheTable(t, "Arcanista")
	if refusal, _ := castWith(t, f, "mao-poderosa-de-talude", `{"augment0":1}`); refusal != "" {
		t.Fatalf("conjurar a Mão Poderosa com o aumento foi recusado: %q", refusal)
	}
	onTurnInAnActionScene(t, f)

	if refusal := spellAct(t, f, "mao-poderosa-de-talude", "Esmagar", `{}`); refusal != "" {
		t.Fatalf("o Esmagar foi recusado: %q", refusal)
	}
	state := stateOf(t, f.s.sessions, f.sessionID)
	cast := state.LastSpellCast
	if cast == nil || cast.Spell != "Mão Poderosa de Talude" || cast.Kind != live.RollOfTheAction || cast.Option != "Esmagar" {
		t.Fatalf("a faixa da magia diz %+v; want o Esmagar da Mão Poderosa", cast)
	}
	if len(cast.Parts) != 1 || cast.Parts[0].Dice != "3d6+15" || cast.Parts[0].Kind != "impacto" {
		t.Errorf("o Esmagar rolou %+v; want 3d6+15 de impacto, com o aumento da conjuração (p198)", cast.Parts)
	}
	if state.Scene.MovementLeft {
		t.Error("o Esmagar não gastou a ação de movimento da vez")
	}

	t.Run("o total rolado na mão chega à faixa", func(t *testing.T) {
		f.s.sessions.NextTurn(t.Context(), f.sessionID) //nolint:errcheck // o controle abaixo mede
		f.s.sessions.NextTurn(t.Context(), f.sessionID) //nolint:errcheck
		if refusal := spellAct(t, f, "mao-poderosa-de-talude", "Esmagar", `{"spell_hand_1":30}`); refusal != "" {
			t.Fatalf("os 30 da mesa foram recusados: %q", refusal)
		}
		if part := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast.Parts[0]; part.Total != 30 || !part.ByHand {
			t.Errorf("a faixa diz %d, à mão %v; want os 30 da mesa", part.Total, part.ByHand)
		}
	})
}

// "Uma vez por turno" (p194): a segunda tempestade da mesma vez é recusada, e a recusa não gasta a ação.
func TestTheFuryOfThePantheonIsOncePerTurn(t *testing.T) {
	f := turnCasterAtTheTable(t, "Clérigo")
	if refusal, _ := castWith(t, f, "furia-do-panteao", `{}`); refusal != "" {
		t.Fatalf("conjurar a Fúria do Panteão foi recusado: %q", refusal)
	}
	onTurnInAnActionScene(t, f)
	if refusal := spellAct(t, f, "furia-do-panteao", "Raios", `{}`); refusal != "" {
		t.Fatalf("os Raios foram recusados: %q", refusal)
	}
	if cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast; cast.Parts[0].Dice != "10d8" || cast.Save != "Reflexos reduz à metade" {
		t.Errorf("os Raios rolaram %+v com a resistência %q; want 10d8, Reflexos reduz à metade", cast.Parts, cast.Save)
	}
	standard := stateOf(t, f.s.sessions, f.sessionID).Scene.StandardLeft
	refusal := spellAct(t, f, "furia-do-panteao", "Nevasca", `{}`)
	if !strings.Contains(refusal, "uma vez por turno") {
		t.Errorf("a segunda tempestade da vez saiu com a recusa %q", refusal)
	}
	if after := stateOf(t, f.s.sessions, f.sessionID).Scene.StandardLeft; after != standard {
		t.Error("a recusa do uma vez por turno gastou ação")
	}
}

// A energia mista "termina a magia imediatamente" (p204), e custa a ação padrão.
func TestTheMixedEnergyEndsTheSpell(t *testing.T) {
	f := turnCasterAtTheTable(t, "Arcanista")
	if refusal, _ := castWith(t, f, "relampago-flamejante-de-reynard", `{}`); refusal != "" {
		t.Fatalf("conjurar o Relâmpago Flamejante foi recusado: %q", refusal)
	}
	onTurnInAnActionScene(t, f)
	if refusal := spellAct(t, f, "relampago-flamejante-de-reynard", "Energia mista", `{}`); refusal != "" {
		t.Fatalf("a energia mista foi recusada: %q", refusal)
	}
	state := stateOf(t, f.s.sessions, f.sessionID)
	if cast := state.LastSpellCast; cast.Parts[0].Dice != "20d12" || cast.Option != "Energia mista" {
		t.Errorf("a energia mista rolou %+v (%q); want 20d12", cast.Parts, cast.Option)
	}
	if state.Scene.StandardLeft {
		t.Error("a energia mista não gastou a ação padrão")
	}
	if _, open := effectOf(t, f, "relampago-flamejante-de-reynard"); open {
		t.Error("a energia mista não terminou a magia")
	}
}

// OS DARDOS SÃO CONTADOS (p196): a conjuração com o aprimoramento não rola nada e abre quatro, cada dardo rola o
// efeito normal da magia, e o último encerra.
func TestTheDartsOfTheIgneousSpearAreCountedUntilTheLast(t *testing.T) {
	f := turnCasterAtTheTable(t, "Arcanista")
	if refusal, _ := castWith(t, f, "lanca-ignea-de-aleph", `{"augment1":1}`); refusal != "" {
		t.Fatalf("conjurar a Lança Ígnea com os dardos foi recusado: %q", refusal)
	}
	if cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast; cast == nil || len(cast.Parts) != 0 {
		t.Fatalf("a conjuração com os dardos rolou %+v; want nada rolado", cast)
	}
	effect, open := effectOf(t, f, "lanca-ignea-de-aleph")
	if !open || effect.Scope != "scene" || effect.Charges != 4 {
		t.Fatalf("a Lança com os dardos abriu %+v (aberta %v); want uma linha de cena com 4 dardos", effect, open)
	}
	for dart := 1; dart <= 4; dart++ {
		if refusal := spellAct(t, f, "lanca-ignea-de-aleph", "Dardo de lava", `{}`); refusal != "" {
			t.Fatalf("o dardo %d foi recusado: %q", dart, refusal)
		}
		if cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast; len(cast.Parts) != 2 || cast.Parts[0].Dice != "4d6" {
			t.Errorf("o dardo %d rolou %+v; want 4d6 de fogo e 4d6 de perfuração", dart, cast.Parts)
		}
		left, open := effectOf(t, f, "lanca-ignea-de-aleph")
		if dart < 4 && (!open || left.Charges != 4-dart) {
			t.Errorf("depois do dardo %d sobraram %d (aberta %v); want %d", dart, left.Charges, open, 4-dart)
		}
		if dart == 4 && open {
			t.Error("o último dardo não encerrou a magia")
		}
	}
}

// A ABA AÇÕES mostra a opção no grupo da ação dela, enquanto a magia está em vigor.
func TestTheActionsSurfaceOffersTheSpellAction(t *testing.T) {
	f := turnCasterAtTheTable(t, "Arcanista")
	if refusal, _ := castWith(t, f, "mao-poderosa-de-talude", `{"augment0":1}`); refusal != "" {
		t.Fatalf("conjurar a Mão Poderosa foi recusado: %q", refusal)
	}
	// A mesa desenha as Ações da PRIMEIRA ficha do jogador na campanha: o conjurador fica sendo a única.
	if _, err := f.s.db.Exec(`DELETE FROM campaign_members WHERE campaignId = ? AND characterId != ?`, f.campaignID, f.charID); err != nil {
		t.Fatalf("tirar a outra ficha do jogador da mesa: %v", err)
	}
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	movement := entre(t, surface, "Ação de movimento", `</section>`)
	for _, want := range []string{"Esmagar · Mão Poderosa de Talude", "3d6+15 de impacto"} {
		if !strings.Contains(movement, want) {
			t.Errorf("o grupo da ação de movimento não tem %q", want)
		}
	}
}
