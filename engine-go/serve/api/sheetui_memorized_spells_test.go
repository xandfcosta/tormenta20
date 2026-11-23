package api

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

// O MAGO MEMORIZA (p37): "você só pode lançar magias memorizadas", e memoriza metade das que conhece,
// arredondando para baixo — "se conhece 7 magias, escolhe 3". Só ele: o clérigo e o druida conjuram o que
// conhecem.

// mago é a arcanista de 9º nível do caminho mago, com estas magias no grimório, nenhuma memorizada.
func mago(t *testing.T, spells ...string) (sceneFixture, int64) {
	t.Helper()
	f, id := arcanistaDeNivel(t, 9)
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET classChoices = '{"Arcanista":{"caminho":"mago"}}' WHERE id = ?`, id); err != nil {
		t.Fatalf("pôr a arcanista no caminho mago: %v", err)
	}
	for _, spell := range spells {
		learn(t, f, id, spell)
	}
	return f, id
}

func TestTheMagoMemorizesHalfOfWhatSheKnows(t *testing.T) {
	// Cinco magias: memoriza duas.
	f, id := mago(t, "armadura-arcana", "explosao-de-chamas", "seta-infalivel-de-talude", "bola-de-fogo", "luz")
	if refusal := spellRefusal(t, f, id, "conjura/luz"); !strings.Contains(refusal, "memoriz") {
		t.Errorf("conjurar a Luz não memorizada foi %q; want a recusa do mago", refusal)
	}
	for _, spell := range []string{"luz", "explosao-de-chamas"} {
		if refusal := spellRefusal(t, f, id, "memoriza/"+spell); refusal != "" {
			t.Fatalf("memorizar %s foi recusado: %q", spell, refusal)
		}
	}
	if refusal := spellRefusal(t, f, id, "memoriza/bola-de-fogo"); !strings.Contains(refusal, "metade") {
		t.Errorf("memorizar a terceira de cinco foi %q; want a recusa pela metade", refusal)
	}
	if spellbook(t, f, id)["bola-de-fogo"] {
		t.Error("a terceira magia ficou memorizada mesmo recusada")
	}
	if refusal := spellRefusal(t, f, id, "conjura/luz"); refusal != "" {
		t.Errorf("conjurar a Luz memorizada foi recusado: %q", refusal)
	}
	full := spellScreen(t, f, id)
	if !strings.Contains(full, "2 de 2 memorizadas") {
		t.Error("a aba não diz quantas o mago memorizou do limite")
	}
	// No limite, o Memorizada das outras tranca: o servidor recusaria de todo jeito.
	if toggle := regexp.MustCompile(`<input type="checkbox" role="switch"[^>]*>\s*Memorizada`).FindString(spellSheetOf(t, full, "spell-bola-de-fogo")); !strings.Contains(toggle, "disabled") {
		t.Errorf("no limite, o Memorizada da Bola de Fogo está de pé: %q", toggle)
	}
	// Desmemorizar abre a vaga.
	spellRefusal(t, f, id, "memoriza/luz")
	if refusal := spellRefusal(t, f, id, "memoriza/bola-de-fogo"); refusal != "" {
		t.Errorf("memorizar depois de liberar a vaga foi recusado: %q", refusal)
	}
}

// A MAGIA MEMORIZADA ACIMA DO LIMITE, gravada antes da regra, fica e se conjura; memorizar outra é recusado.
func TestTheMagoKeepsTheMemorizedSpellsAboveTheLimit(t *testing.T) {
	f, id := mago(t, "armadura-arcana", "explosao-de-chamas", "luz", "seta-infalivel-de-talude")
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE character_spells SET prepared = 1 WHERE characterId = ? AND catalogSpellId != 'seta-infalivel-de-talude'`, id); err != nil {
		t.Fatalf("memorizar três de quatro à mão: %v", err)
	}
	if refusal := spellRefusal(t, f, id, "conjura/luz"); refusal != "" {
		t.Errorf("conjurar a Luz memorizada acima do limite foi recusado: %q", refusal)
	}
	if refusal := spellRefusal(t, f, id, "memoriza/seta-infalivel-de-talude"); !strings.Contains(refusal, "metade") {
		t.Errorf("memorizar a quarta com três de dois foi %q; want a recusa pela metade", refusal)
	}
	if !strings.Contains(spellScreen(t, f, id), "3 de 2 memorizadas") {
		t.Error("a aba não diz que o mago passou do limite")
	}
}

// O CLÉRIGO CONJURA O QUE CONHECE: não há o que memorizar, nem interruptor na folha.
func TestTheClericCastsWhatSheKnows(t *testing.T) {
	f := clericOf(t, 5, "")
	learnSpell(t, f, "curar-ferimentos")
	if refusal := castFromTheSheet(t, f, "curar-ferimentos"); refusal != "" {
		t.Errorf("a clériga conjurar o Curar Ferimentos foi recusado: %q", refusal)
	}
	if refusal := spellRefusal(t, f, f.charID, "memoriza/curar-ferimentos"); !strings.Contains(refusal, "mago") {
		t.Errorf("a clériga memorizar foi %q; want a recusa de quem não é mago", refusal)
	}
	if regexp.MustCompile(`Memorizad`).MatchString(spellScreen(t, f, f.charID)) {
		t.Error("a aba da clériga fala em memorizar")
	}
}

// NO MULTICLASSE, memoriza-se só a magia que a ficha lança APENAS pelo mago: a Luz, que o clérigo também lança,
// se conjura sem memorizar.
func TestTheMultiClassMagoMemorizesOnlyTheMagosSpells(t *testing.T) {
	f, id := mago(t, "luz", "bola-de-fogo")
	seedClasse(t, f.s, id, "Clérigo", 1)
	if refusal := spellRefusal(t, f, id, "conjura/luz"); refusal != "" {
		t.Errorf("conjurar a Luz, que o clérigo lança, foi recusado: %q", refusal)
	}
	if refusal := spellRefusal(t, f, id, "conjura/bola-de-fogo"); !strings.Contains(refusal, "memoriz") {
		t.Errorf("conjurar a Bola de Fogo não memorizada foi %q; want a recusa do mago", refusal)
	}
}
