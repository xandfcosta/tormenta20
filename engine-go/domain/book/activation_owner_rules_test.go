package book

import (
	"strings"
	"testing"
)

// TODA ATIVAÇÃO TEM DONO (ALE-492): a ficha liga o gesto ao que o personagem
// tem pelo id ou pelo nome (`sheet.OwnsActivation`). Uma ativação cujo id e
// nome não casam com poder, habilidade ou benefício nenhum nunca aparece — é
// um gesto escrito para ninguém, e o erro de digitação nasce calado.
//
// O encanto de item (`item.*`) fica de fora: o dono dele é o item equipado.
var orphanActivations = map[string]string{
	// O catálogo modela a Herança Divina do suraggel como a ascendência, e
	// Luz Sagrada e Sombras Profanas como variantes dela; no livro (p31) são
	// habilidades. Registrado na ALE-504.
	"race.suraggel.luz-sagrada":      "Herança Divina modelada como ascendência",
	"race.suraggel.sombras-profanas": "Herança Divina modelada como ascendência",
}

func TestEveryActivationHasAnOwner(t *testing.T) {
	owners := map[string]bool{}
	own := func(id, name string) { owners[id], owners["name:"+name] = true, true }
	for _, p := range FlattenedPowers() {
		own(p.ID, p.Name)
	}
	for _, p := range GeneralPowers() {
		own(p.ID, p.Name)
	}
	for key, race := range RaceTraitsByKey() {
		if key != race.ID {
			continue
		}
		for _, a := range race.Abilities {
			own(a.ID, a.Name)
		}
	}
	for _, o := range Origins() {
		own(o.UniquePower.ID, o.UniquePower.Name)
		for _, b := range o.Benefits {
			own(b.ID, b.Name)
		}
	}
	measured := 0
	for _, a := range Activations() {
		if strings.HasPrefix(a.ID, "item.") {
			continue
		}
		measured++
		owned := owners[a.ID] || owners["name:"+a.Name]
		reason, known := orphanActivations[a.ID]
		switch {
		case !owned && !known:
			t.Errorf("a ativação %s (%q) não casa com poder, habilidade ou benefício nenhum: confira o id e o nome", a.ID, a.Name)
		case owned && known:
			t.Errorf("a ativação %s ganhou dono e continua em `orphanActivations` (%q): tire-a da lista", a.ID, reason)
		}
	}
	if measured < 400 {
		t.Fatalf("mediu %d ativações; o catálogo tem mais de 400", measured)
	}
}
