package api

import (
	"context"
	"regexp"
	"strings"
	"testing"
)

// A RECUSA DE CONJURAR É UMA SÓ: o servidor, a aba Magias e a aba Ações leem a do conjurador. A magia do mago
// não memorizada (p37) e a acima do círculo (p170) trancam o Conjurar nas duas abas, com a frase, em vez de
// deixá-lo de pé para o servidor recusar ao tocar.
func TestTheCastLockIsTheSameOnBothTabs(t *testing.T) {
	f := newSceneFixture(t)
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE characters SET classChoices = '{"Arcanista":{"caminho":"mago"}}' WHERE id = ?`, f.charID); err != nil {
		t.Fatalf("pôr a arcanista no caminho mago: %v", err)
	}
	// Explosão de Chamas memorizada, Luz não; o Desintegrar, gravado antes da regra, é do 4º círculo, e a
	// arcanista de 8º alcança o 2º.
	for _, spell := range []string{"explosao-de-chamas", "luz", "desintegrar"} {
		learn(t, f, f.charID, spell)
	}
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE character_spells SET prepared = 1 WHERE characterId = ? AND catalogSpellId = 'explosao-de-chamas'`, f.charID); err != nil {
		t.Fatalf("memorizar a Explosão de Chamas: %v", err)
	}

	actions := entre(t, acoesDaMesa(t, f), "Ação padrão", "Ação de movimento")
	for _, c := range []struct{ name, why string }{
		{"Luz", "memorize a magia antes de conjurá-la"},
		{"Desintegrar", "Desintegrar é do 4º círculo"},
	} {
		line := castLineOf(t, actions, c.name)
		if !strings.Contains(line, c.why) {
			t.Errorf("na aba Ações, o Conjurar · %s não diz %q:\n%s", c.name, c.why, recorte(line))
		}
	}

	grimoire := spellScreen(t, f, f.charID)
	for _, c := range []struct {
		name   string
		locked bool
	}{{"Luz", true}, {"Desintegrar", true}, {"Explosão de Chamas", false}} {
		button := regexp.MustCompile(`<button[^>]*aria-label="Conjurar ` + c.name + `"[^>]*>`).FindString(grimoire)
		if button == "" {
			t.Fatalf("a aba Magias não desenhou o Conjurar %s", c.name)
		}
		if got := strings.Contains(button, "disabled"); got != c.locked {
			t.Errorf("na aba Magias, o Conjurar %s trancado = %v; want %v", c.name, got, c.locked)
		}
	}
}

// castLineOf é o trecho da linha "Conjurar · <magia>" na aba Ações, até a próxima linha de conjurar.
func castLineOf(t *testing.T, actions, name string) string {
	t.Helper()
	start := strings.Index(actions, "Conjurar · "+name)
	if start < 0 {
		t.Fatalf("a aba Ações não desenhou o Conjurar · %s:\n%s", name, recorte(actions))
	}
	line := actions[start+len("Conjurar · "):]
	if next := strings.Index(line, "Conjurar · "); next >= 0 {
		line = line[:next]
	}
	return line
}
