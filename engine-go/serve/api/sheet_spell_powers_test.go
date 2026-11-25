package api

import (
	"context"
	"net/http"
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
