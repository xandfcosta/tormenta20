package book

import (
	"slices"
	"testing"
)

// TODA DECLARAÇÃO LIGA UMA FLAG QUE O PRÓPRIO PODER LÊ, e que acaba sozinha. Com a flag escrita
// errada, o Usar gravava um interruptor que nenhum modificador casa: o botão funcionava e o -2 e o
// segundo ataque não vinham. E sem o `Ends`, o uso valeria até alguém encerrá-lo à mão.
func TestEveryDeclarationLightsAFlagItsPowerReadsAndThatEnds(t *testing.T) {
	ends := map[string][]string{} // id do poder -> flags dos modificadores com fim
	for _, p := range slices.Concat(ClassPowersWithModifiers(), GeneralPowersWithModifiers()) {
		for _, m := range p.Modifiers {
			if m.Condition != nil && m.Condition.C == "flagOn" && m.Condition.Ends != "" {
				ends[p.ID] = append(ends[p.ID], m.Condition.Flag)
			}
		}
	}
	measured := 0
	for _, d := range declarations() {
		measured++
		if !slices.Contains(ends[d.PowerID], d.Flag) {
			t.Errorf("%s (%s) declara a flag %q, e nenhum modificador do poder a lê com fim (`Ends`): "+
				"o gesto ligaria uma flag inerte, ou que nunca desliga. Os do poder: %v", d.Name, d.PowerID, d.Flag, ends[d.PowerID])
		}
	}
	if measured == 0 {
		t.Fatal("nenhuma declaração no catálogo — o guarda não mediu nada")
	}
}
