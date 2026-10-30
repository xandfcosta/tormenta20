package api

import (
	"strings"
	"testing"
)

// A ABA COMBATE DA FICHA (ALE-473): Defesa e os dois ataques como
// linhas da conta, uma linha por arma empunhada, e as resistências FORA — elas
// moram no topo das Perícias (decisão do dono). Sem "Atacar": na ficha solta
// não há alvo nem turno, e o gesto vem com a ficha embutida.
//
// Os números são os do combatente de nível 3 do `fighterFixture`, escritos à
// mão: Defesa 10 + DES 2 = 12; Luta ½ nível 1 + FOR 4 + treino 2 = +7;
// Pontaria 1 + DES 2 = +3; a espada longa rola 1d8 + FOR 4, 19-20/x2. O
// fixture não tem proficiência nenhuma, então a espada cobra -5 no ataque
// (p142), e o corpo a corpo, que lê a arma empunhada, também: +7 - 5 = +2.
func TestTheParallelCombatTabSumsDefenseAttacksAndWeapons(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	page := parallelSheet(t, f, id, "combat")

	for name, total := range map[string]string{
		"Defesa": "12", "Ataque corpo a corpo": "+2", "Ataque à distância": "+3", "Espada longa": "+2",
	} {
		if sum := sumLine(t, page, name); !strings.Contains(sum, `<span class="ds-row-value">`+total+`</span>`) {
			t.Errorf("a linha %q não fechou em %s:\n%s", name, total, sum)
		}
	}
	sword := sumLine(t, page, "Espada longa")
	if !strings.Contains(sword, "1d8+4") || !strings.Contains(sword, "19-20/x2") {
		t.Errorf("a linha da espada não diz o dano e o crítico:\n%s", sword)
	}
	if !strings.Contains(sword, "<dd data-penalty>-5</dd>") {
		t.Errorf("a conta da espada não pinta o -5 de quem não é proficiente:\n%s", sword)
	}

	section, _, _ := strings.Cut(page[strings.Index(page, `aria-label="Combate"`):], "</section>")
	for _, gone := range []string{"Fortitude", "Reflexos", "Vontade", "Atacar"} {
		if strings.Contains(section, gone) {
			t.Errorf("a aba Combate tem %q, e não devia", gone)
		}
	}
}

// A MAGIA SÓ APARECE PARA QUEM CONJURA, e o ramo é nomeado: o guerreiro não tem
// a lista, a arcanista tem.
func TestTheParallelCombatTabShowsMagicOnlyToWhoCasts(t *testing.T) {
	f := newSceneFixture(t)
	warrior := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)
	wizard := seedCharacterAtLevel(t, f.s, f.player, "Ana", "Arcanista", 3, 0, 0)

	if page := parallelSheet(t, f, warrior, "combat"); strings.Contains(page, ">CD das magias<") {
		t.Error("o guerreiro tem a lista de magia no Combate")
	}
	if page := parallelSheet(t, f, wizard, "combat"); !strings.Contains(page, ">CD das magias<") {
		t.Error("a arcanista não tem a CD das magias no Combate")
	}
}
