package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// "Na forma selvagem você não pode falar, empunhar itens ou lançar magias" (p62), e a Magia Natural (p62):
// "em forma selvagem, você pode lançar magias e empunhar catalisadores e esotéricos".

// shapedCaster é o druida com uma magia preparada, já na Forma Feroz, com os poderes pedidos.
func shapedCaster(t *testing.T, powers string, items ...string) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Druida que conjura", "Druida", 6, 0, 0)
	choiceCom(t, f, id, powers, `[]`)
	for _, item := range items {
		itemSemeia(t, f, id, item, item, "wielded")
	}
	for _, step := range []string{"aprende", "prepara"} {
		target := fmt.Sprintf("/personagens/%d/magias/%s/curar-ferimentos?tab=spells", id, step)
		if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, "").Body.String()); refusal != "" {
			t.Fatalf("%s Curar Ferimentos: %q", step, refusal)
		}
	}
	if _, err := f.s.db.ExecContext(t.Context(),
		"INSERT INTO character_wild_shape (characterId, form, tier, damageType, movement, pmPaid) VALUES (?, 'feroz', 'normal', 'corte', '', 3)", id); err != nil {
		t.Fatalf("pôr o druida na forma: %v", err)
	}
	return f, id
}

func castsCure(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/magias/conjura/curar-ferimentos?tab=spells", id)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, "").Body.String())
}

func TestTheShapeDoesNotCastWithoutNaturalMagic(t *testing.T) {
	f, id := shapedCaster(t, `["class.druida.forma-selvagem"]`)
	if refusal := castsCure(t, f, id); !strings.Contains(refusal, "Forma Selvagem") {
		t.Errorf("na forma, conjurar devia ser recusado dizendo por quê, e veio %q", refusal)
	}
	if !strings.Contains(spellScreen(t, f, id), "não lança magias") {
		t.Error("a aba Magias não diz que a forma não lança magias")
	}
}

func TestNaturalMagicCastsInTheShape(t *testing.T) {
	f, id := shapedCaster(t, `["class.druida.forma-selvagem","class.druida.magia-natural"]`)
	if refusal := castsCure(t, f, id); refusal != "" {
		t.Errorf("com a Magia Natural, conjurar na forma foi recusado: %q", refusal)
	}
	if strings.Contains(spellScreen(t, f, id), "não lança magias") {
		t.Error("com a Magia Natural, a aba Magias ainda diz que a forma não lança magias")
	}
}

// O esotérico fica na mão com a Magia Natural; a adaga, não.
func TestNaturalMagicKeepsTheEsotericInHand(t *testing.T) {
	for _, c := range []struct {
		name, powers string
		kept         bool
	}{
		{"sem a Magia Natural", `["class.druida.forma-selvagem"]`, false},
		{"com a Magia Natural", `["class.druida.forma-selvagem","class.druida.magia-natural"]`, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id := shapedCaster(t, c.powers, "varinha-arcana", "adaga")
			bag := bagScreen(t, f, id)
			if !strings.Contains(bag, "adaga") {
				t.Fatal("o controle falhou: a Mochila não mostra a adaga")
			}
			guarded := strings.Count(bag, "guardada pela Forma Selvagem")
			if want := map[bool]int{true: 1, false: 2}[c.kept]; guarded != want {
				t.Errorf("%d itens guardados pela forma, want %d (a adaga sempre; a varinha só sem a Magia Natural)", guarded, want)
			}
			if says := strings.Contains(bag, "nada se empunha"); says == c.kept {
				t.Errorf("a linha das mãos diz \"nada se empunha\" = %v, e a varinha está na mão = %v", says, c.kept)
			}
		})
	}
}
