package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// A FORMA ÁGIL ATACA COM AS DUAS ARMAS NATURAIS (p63): "quando usa a ação agredir, pode atacar com ambas,
// mas sofre –2 em todos os testes de ataque até o seu próximo turno". É uma declaração, como o Estilo de
// Duas Armas: um Usar antes do golpe, e não uma chave. As duas armas são iguais, e o adicional sai da
// mesma carta, como a outra ponta da arma dupla.

const agileBoth = "ataque-forma-agil"

// druidShapedOnTurn é o druida na vez, já na forma pedida — gravada direto, porque transformar pela ficha
// gasta a ação completa, e aí não sobraria o agredir.
func druidShapedOnTurn(t *testing.T, form string) (sceneFixture, int64, string) {
	t.Helper()
	f, id, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, id, "Druida", 1)
	seedPowerChoices(t, f.s, id, `["class.druida.forma-selvagem"]`, "{}")
	if _, err := f.s.db.ExecContext(t.Context(), "DELETE FROM character_items WHERE characterId = ?", id); err != nil {
		t.Fatalf("limpar as mãos: %v", err)
	}
	if form != "" {
		if _, err := f.s.db.ExecContext(t.Context(),
			"INSERT INTO character_wild_shape (characterId, form, tier, damageType, movement, pmPaid) VALUES (?, ?, 'normal', 'corte', '', 3)",
			id, form); err != nil {
			t.Fatalf("pôr o druida na forma %s: %v", form, err)
		}
	}
	return f, id, goblin
}

func declares(t *testing.T, f sceneFixture, id int64, flag string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/declara/%s", id, flag)
	return sceneRefusal(f.requests(t, f.player, http.MethodPost, target, `{}`).Body.String())
}

// agileCard é o índice da carta da forma nas Ações: sem arma na mão, o desarmado vem antes dela.
const agileCard = 1

func TestTheAgileFormAttacksWithBothNaturalWeapons(t *testing.T) {
	f, id, goblin := druidShapedOnTurn(t, "agil")
	if !strings.Contains(acoesDaMesa(t, f), "/declara/"+agileBoth) {
		t.Error("as Ações da mesa não têm o gesto de atacar com as duas armas naturais")
	}
	if strings.Contains(effectScreen(t, f, id), "flag:"+agileBoth) {
		t.Error("a aba Efeitos tem uma chave para a declaração da Forma Ágil")
	}
	if refusal := switchRefusal(t, f, id, agileBoth); refusal == "" {
		t.Error("a declaração da Forma Ágil ligou pela rota do interruptor")
	}

	if refusal := rollsFromTheActions(t, f, id, fmt.Sprintf("atacar/%d", agileCard), goblin); refusal != "" {
		t.Fatalf("o golpe sem declarar: %q", refusal)
	}
	off := pendingOf(t, f)
	if off == nil || !strings.Contains(off.Weapon, "Ágil") {
		t.Fatalf("o golpe devia ser com a arma da Forma Ágil: %+v", off)
	}
	gmCancels(t, f)
	if refusal := declares(t, f, id, agileBoth); refusal != "" {
		t.Fatalf("declarar foi recusado: %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, id, fmt.Sprintf("atacar/%d", agileCard), goblin); refusal != "" {
		t.Fatalf("o primeiro golpe: %q", refusal)
	}
	if on := pendingOf(t, f); (on.Total-on.Roll)-(off.Total-off.Roll) != -2 {
		t.Errorf("declarar mudou o ataque em %+d, e o livro dá -2", (on.Total-on.Roll)-(off.Total-off.Roll))
	}
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, agileCard, goblin); refusal != "" {
		t.Fatalf("o ataque com a outra arma natural foi recusado: %q", refusal)
	}
	gmRules(t, f, "errou")
	if refusal := extraStrike(t, f, id, agileCard, goblin); refusal == "" {
		t.Error("um terceiro ataque passou: são as duas armas, e não três")
	}
}

// Só a Forma Ágil tem duas armas naturais: a Feroz tem uma, e fora da forma não há nenhuma.
func TestOnlyTheAgileFormDeclaresBothNaturalWeapons(t *testing.T) {
	for _, form := range []string{"feroz", ""} {
		t.Run("forma "+form, func(t *testing.T) {
			f, id, _ := druidShapedOnTurn(t, form)
			if strings.Contains(acoesDaMesa(t, f), "/declara/"+agileBoth) {
				t.Error("as Ações oferecem o gesto fora da Forma Ágil")
			}
			if refusal := declares(t, f, id, agileBoth); refusal == "" {
				t.Error("declarar fora da Forma Ágil foi aceito")
			}
		})
	}
}

// "Até o seu próximo turno": quando a vez do druida volta, a declaração sai sozinha.
func TestTheAgileDeclarationEndsWhenTheTurnComesBack(t *testing.T) {
	f, id, _ := druidShapedOnTurn(t, "agil")
	if refusal := declares(t, f, id, agileBoth); refusal != "" {
		t.Fatalf("declarar: %q", refusal)
	}
	end := "/efeitos/declaracao/" + agileBoth
	if !strings.Contains(effectScreen(t, f, id), end) {
		t.Fatal("a aba Efeitos não mostra a declaração em vigor, com o Encerrar")
	}
	for range 2 {
		if _, err := f.s.sessions.NextTurn(t.Context(), f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}
	if strings.Contains(effectScreen(t, f, id), end) {
		t.Error("a declaração sobreviveu à volta da vez do druida")
	}
}
