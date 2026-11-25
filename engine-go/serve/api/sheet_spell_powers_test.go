package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"t20engine/domain/sheet"
)

// OS PODERES NO DADO DA MAGIA, de ponta a ponta (ver GLOSSARY, **dado da magia**): o Arcanista com o Arcano de
// Batalha conjura a Bola de Fogo, e a faixa da mesa traz a parcela do poder, com o atributo-chave dele (p38).

// keyAttributeOf é o total computado do atributo-chave de uma classe que conjura.
func keyAttributeOf(t *testing.T, f sceneFixture, class string) int {
	t.Helper()
	row, err := f.s.queries.GetCharacter(context.Background(), f.charID)
	if err != nil {
		t.Fatalf("ler a ficha: %v", err)
	}
	dto, err := f.s.sheetRules().LoadCharacter(context.Background(), row)
	if err != nil {
		t.Fatalf("carregar a ficha: %v", err)
	}
	computed, err := sheet.Compute(dto)
	if err != nil {
		t.Fatalf("computar a ficha: %v", err)
	}
	return computed.Attributes[computed.SpellAttributeByClass[class]].Total
}

func TestTheBattleArcaneAddsItsPartToTheCast(t *testing.T) {
	f := turnCasterAtTheTable(t, "Arcanista")
	seedPowerChoices(t, f.s, f.charID, `["class.arcanista.arcano-de-batalha"]`, "{}")
	if _, err := f.s.db.Exec(`UPDATE characters SET intelligence = 4 WHERE id = ?`, f.charID); err != nil {
		t.Fatalf("dar Inteligência ao Arcanista: %v", err)
	}
	if keyAttributeOf(t, f, "Arcanista") == 0 {
		t.Fatal("o atributo-chave do Arcanista é 0, e o Arcano de Batalha somaria nada: o caso não mede o poder")
	}
	if refusal, _ := castWith(t, f, "bola-de-fogo", `{}`); refusal != "" {
		t.Fatalf("conjurar a Bola de Fogo foi recusado: %q", refusal)
	}
	cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast
	want := keyAttributeOf(t, f, "Arcanista")
	if cast == nil || len(cast.Bonuses) != 1 || cast.Bonuses[0].Name != "Arcano de Batalha" || cast.Bonuses[0].Amount != want {
		t.Fatalf("a faixa da Bola de Fogo traz %+v; want a parcela do Arcano de Batalha, +%d", cast, want)
	}
	page := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(page, "Arcano de Batalha") {
		t.Error("a faixa da mesa não escreve a parcela do Arcano de Batalha")
	}
}

// O ASPECTO DO INVERNO vale em todo dano de frio da magia, também no que ela causa depois de conjurada: a Nevasca
// da Fúria do Panteão, 10d6 de frio, soma +10 (p61, p194).
func TestTheWinterAspectAddsToTheFrostAfterTheCast(t *testing.T) {
	// O Aspecto do Inverno é do druida, e a Fúria pede o 5º círculo divino: um clérigo que também é druida.
	f := turnCasterAtTheTable(t, "Clérigo")
	seedClasse(t, f.s, f.charID, "Druida", 1)
	seedPowerChoices(t, f.s, f.charID, `["class.druida.aspecto-do-inverno"]`, "{}")
	if refusal, _ := castWith(t, f, "furia-do-panteao", `{}`); refusal != "" {
		t.Fatalf("conjurar a Fúria do Panteão foi recusado: %q", refusal)
	}
	onTurnInAnActionScene(t, f)
	if refusal := spellAct(t, f, "furia-do-panteao", "Nevasca", `{}`); refusal != "" {
		t.Fatalf("a Nevasca foi recusada: %q", refusal)
	}
	cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast
	if cast == nil || len(cast.Bonuses) != 1 || cast.Bonuses[0].Name != "Aspecto do Inverno" || cast.Bonuses[0].Amount != 10 {
		t.Errorf("a Nevasca traz %+v; want +10 do Aspecto do Inverno, um por dado de frio", cast)
	}
}

// O ESPÍRITO DOS SOLSTÍCIOS põe o dado no máximo, sem rolar (p62): a Curar Ferimentos, 2d8+2, cura 18.
func TestTheSolsticeSpiritMaximizesTheCast(t *testing.T) {
	f := turnCasterAtTheTable(t, "Druida")
	seedPowerChoices(t, f.s, f.charID,
		`["class.druida.aspecto-do-inverno","class.druida.aspecto-do-verao","class.druida.espirito-dos-solsticios"]`, "{}")
	if refusal, _ := castWith(t, f, "curar-ferimentos", `{"augment_solstice":1}`); refusal != "" {
		t.Fatalf("conjurar a Curar Ferimentos com o Espírito dos Solstícios foi recusado: %q", refusal)
	}
	cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast
	if cast == nil || len(cast.Parts) != 1 || cast.Parts[0].Total != 18 || !cast.Parts[0].Maximized || len(cast.Parts[0].Rolls) != 0 {
		t.Fatalf("a Curar Ferimentos maximizada diz %+v; want 18, no máximo, sem dados", cast)
	}
}

// A COMUNHÃO VITAL faz a faixa dizer a metade da cura que outra criatura recupera (p58).
func TestTheVitalCommunionSharesHalfTheHealing(t *testing.T) {
	f := turnCasterAtTheTable(t, "Clérigo")
	seedPowerChoices(t, f.s, f.charID, `["class.clerigo.comunhao-vital"]`, "{}")
	if refusal, _ := castWith(t, f, "curar-ferimentos", `{"augment_vital_communion":1,"spell_hand_1":15}`); refusal != "" {
		t.Fatalf("conjurar a Curar Ferimentos com a Comunhão Vital foi recusado: %q", refusal)
	}
	if cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast; cast == nil || cast.VitalShare != 7 {
		t.Fatalf("a Comunhão Vital diz %+v; want 7, a metade dos 15 da cura", cast)
	}
	page := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(page, "Comunhão Vital: outra criatura em alcance curto recupera 7 PV") {
		t.Error("a faixa da mesa não diz a cura da Comunhão Vital")
	}
}

// A MAGIA SAGRADA/PROFANA troca o tipo do dano pela energia do deus (p58): o de Khalmyr vira luz, e o de Nimb, que
// canaliza qualquer uma, pede a escolha no diálogo.
func TestTheHolySpellTurnsTheDamageByTheGod(t *testing.T) {
	f := turnCasterAtTheTable(t, "Clérigo")
	seedPowerChoices(t, f.s, f.charID, `["class.clerigo.magia-sagrada-profana"]`, "{}")
	godOf := func(god string) {
		t.Helper()
		if _, err := f.s.db.Exec(`UPDATE characters SET god = ? WHERE id = ?`, god, f.charID); err != nil {
			t.Fatalf("pôr o deus %s: %v", god, err)
		}
	}
	kinds := func() []string {
		t.Helper()
		var out []string
		for _, part := range stateOf(t, f.s.sessions, f.sessionID).LastSpellCast.Parts {
			out = append(out, part.Kind)
		}
		return out
	}

	godOf("khalmyr")
	if refusal, _ := castWith(t, f, "coluna-de-chamas", `{"augment_holy":1}`); refusal != "" {
		t.Fatalf("a Coluna de Chamas sagrada foi recusada: %q", refusal)
	}
	if got := strings.Join(kinds(), ","); got != "luz,luz" {
		t.Errorf("a Coluna de Chamas do devoto de Khalmyr rola %s; want luz,luz", got)
	}

	godOf("nimb")
	if screen := spellScreen(t, f, f.charID); !strings.Contains(screen, `aria-label="Luz ou trevas da Magia Sagrada/Profana"`) {
		t.Error("o diálogo de conjurar do devoto de Nimb não pergunta luz ou trevas")
	}
	if refusal, _ := castWith(t, f, "coluna-de-chamas", `{"augment_holy":1}`); !strings.Contains(refusal, "luz ou trevas") {
		t.Errorf("a Coluna de Chamas sagrada do devoto de Nimb, sem escolha, devolveu %q; want a recusa que pede luz ou trevas", refusal)
	}
	if refusal, _ := castWith(t, f, "coluna-de-chamas", `{"augment_holy":1,"augment_holy_energy":"trevas"}`); refusal != "" {
		t.Fatalf("a Coluna de Chamas de trevas foi recusada: %q", refusal)
	}
	if got := strings.Join(kinds(), ","); got != "trevas,trevas" {
		t.Errorf("a Coluna de Chamas do devoto de Nimb, com trevas, rola %s; want trevas,trevas", got)
	}
}

// O RAIO ARCANO na aba Ações (p39; ver GLOSSARY, **raio arcano**): a linha no grupo da ação padrão, e o disparo com
// o Raio Elemental de fogo, rolado na mão, vai pela faixa da magia com a condição de quem falha e a parcela do
// Arcano de Batalha, porque o raio conta como magia; ele cobra 1 PM e a ação padrão.
func TestTheArcaneBoltFiresFromTheActionsTab(t *testing.T) {
	f := turnCasterAtTheTable(t, "Arcanista")
	seedPowerChoices(t, f.s, f.charID,
		`["class.arcanista.raio-arcano","class.arcanista.raio-elemental","class.arcanista.arcano-de-batalha"]`, "{}")
	if _, err := f.s.db.Exec(`UPDATE characters SET intelligence = 4 WHERE id = ?`, f.charID); err != nil {
		t.Fatalf("dar Inteligência ao Arcanista: %v", err)
	}
	// A mesa desenha as Ações da PRIMEIRA ficha do jogador na campanha: o conjurador fica sendo a única.
	if _, err := f.s.db.Exec(`DELETE FROM campaign_members WHERE campaignId = ? AND characterId != ?`, f.campaignID, f.charID); err != nil {
		t.Fatalf("tirar a outra ficha do jogador da mesa: %v", err)
	}
	onTurnInAnActionScene(t, f)

	standard := entre(t, superficieDeAcoes(t, acoesDaMesa(t, f)), "Ação padrão", `</section>`)
	if !strings.Contains(standard, "Raio Arcano") || !strings.Contains(standard, "5d8 de essência") {
		t.Errorf("o grupo da ação padrão não tem o Raio Arcano de 5d8 do arcanista de 20º (5º círculo)")
	}
	// O gesto próprio SUBSTITUI o Usar da ativação, que só gastava a ação padrão sem rolar nada.
	if rows := regexp.MustCompile(`ds-row-title">\s*Raio Arcano\s*<`).FindAllString(standard, -1); len(rows) != 1 {
		t.Errorf("o grupo da ação padrão tem %d linhas do Raio Arcano; want 1, a do gesto", len(rows))
	}

	// A LINHA DO DADO no diálogo acompanha o tipo que o seletor do Raio Elemental escolhe.
	if !strings.Contains(standard, `data-text="&#39;5d8 de &#39; + ($arcane_bolt_element || &#39;essência&#39;)"`) {
		t.Error("a linha do dado no diálogo do Raio Arcano não acompanha o tipo do Raio Elemental")
	}

	mpBefore := mpOf(t, f.s, f.charID)
	target := fmt.Sprintf("/personagens/%d/acoes/raio-arcano?tab=combat", f.charID)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, `{"arcane_bolt_element":"fogo","spell_hand_1":20}`).Body.String()); refusal != "" {
		t.Fatalf("o Raio Arcano de fogo foi recusado: %q", refusal)
	}
	state := stateOf(t, f.s.sessions, f.sessionID)
	cast := state.LastSpellCast
	if cast == nil || cast.Spell != "Raio Arcano" || len(cast.Parts) != 1 || cast.Parts[0].Kind != "fogo" || cast.Parts[0].Total != 20 ||
		!strings.Contains(cast.Parts[0].Note, "em chamas") || cast.Save != "Reflexos reduz à metade" {
		t.Fatalf("a faixa diz %+v; want o Raio Arcano de fogo, 20 da mesa, Reflexos reduz à metade, em chamas para quem falha", cast)
	}
	if len(cast.Bonuses) != 1 || cast.Bonuses[0].Name != "Arcano de Batalha" {
		t.Errorf("o Raio Arcano traz as parcelas %+v; want a do Arcano de Batalha, porque ele conta como magia", cast.Bonuses)
	}
	if spent := mpBefore - mpOf(t, f.s, f.charID); spent != 1 {
		t.Errorf("o Raio Elemental cobrou %d PM; want 1", spent)
	}
	if state.Scene.StandardLeft {
		t.Error("o Raio Arcano não gastou a ação padrão")
	}
}
