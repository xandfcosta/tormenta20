package engine

import (
	"testing"

	"t20engine/domain/ecs"
)

// TODO SISTEMA DE DERIVAÇÃO ESCREVE O COMPONENTE DELE (ALE-378, fatia 4).
//
// # Ele repõe o que o compilador dava
//
// Antes, `displacementBreakdown(base, e, load)` não COMPILAVA sem a carga na
// mão: a assinatura era o escalonador. Com sistemas, um que não rode deixa o
// componente ausente, e quem o lê recebe o valor ZERO — sem erro, sem log, com
// a ficha mostrando 0 de Defesa como se fosse a conta.
//
// Essa troca foi decisão do dono (o critério passou a ser aprender ECS), e este
// caso é o preço dela pago em dia.
//
// # O DENOMINADOR é o que impede a lista de apodrecer
//
// A lista abaixo é escrita à mão — não há como perguntar ao `ecs` quantos
// componentes uma entidade tem sem inventar API para isto. O que a mantém
// honesta é a contagem: ela tem de bater com o número de sistemas. Acrescentar
// um sistema sem acrescentar a conferência dele reprova aqui, com o número.
func TestEveryDerivationSystemWritesItsComponent(t *testing.T) {
	conferencias := []struct {
		nome string
		tem  func(*ecs.World, ecs.Entity) bool
	}{
		{"carga", func(w *ecs.World, e ecs.Entity) bool { _, ok := ecs.Get[derivedLoad](w, e); return ok }},
		{"atributos", func(w *ecs.World, e ecs.Entity) bool { _, ok := ecs.Get[derivedAttributes](w, e); return ok }},
		{"defesa", func(w *ecs.World, e ecs.Entity) bool { _, ok := ecs.Get[derivedDefense](w, e); return ok }},
		{"movimento", func(w *ecs.World, e ecs.Entity) bool { _, ok := ecs.Get[derivedMovement](w, e); return ok }},
		{"magia", func(w *ecs.World, e ecs.Entity) bool { _, ok := ecs.Get[derivedMagic](w, e); return ok }},
		{"combate", func(w *ecs.World, e ecs.Entity) bool { _, ok := ecs.Get[derivedCombat](w, e); return ok }},
		{"perícias", func(w *ecs.World, e ecs.Entity) bool { _, ok := ecs.Get[derivedExpertises](w, e); return ok }},
	}

	if len(conferencias) != len(derivationSystems()) {
		t.Fatalf("são %d sistemas de derivação e %d conferências.\n"+
			"Sistema novo entra nesta lista JUNTO — senão ele pode parar de rodar e "+
			"ninguém fica sabendo: o componente ausente vira valor zero em silêncio.",
			len(derivationSystems()), len(conferencias))
	}

	w := ecs.NewWorld()
	sheet := w.Spawn()
	ecs.Set(w, sheet, sheetInput{char: Character{}, effects: ItemEffects{}, raceSquares: 6})
	ecs.Run(w, derivationSystems()...)

	for _, c := range conferencias {
		if !c.tem(w, sheet) {
			t.Errorf("o componente %q não foi escrito — o sistema que o produz não rodou, e "+
				"quem o ler vai receber o valor ZERO como se fosse a conta", c.nome)
		}
	}
}

// A CARGA VEM ANTES, e o deslocamento é quem prova.
//
// É a única aresta do grafo de dependência: a carga alimenta o deslocamento e
// as perícias. Com os sistemas na ordem errada, o `derivedLoad` ainda não
// existe quando o movimento o lê — e o `ecs.Get` devolve o valor ZERO, que é
// uma carga sem sobrecarga nenhuma.
//
// O caso usa um personagem SOBRECARREGADO de propósito: sem a sobrecarga, a
// carga zerada e a carga de verdade dão o mesmo deslocamento, e o caso mediria
// a metade em que o defeito é invisível.
func TestTheLoadIsDerivedBeforeWhatReadsIt(t *testing.T) {
	// Espaços muito acima do limite: o barril de 40 sobrecarrega qualquer um.
	ch := Character{
		Level: 1,
		Items: []CharacterItem{{Name: "Barril", Quantity: 1, Slots: 40}},
	}
	in := sheetInput{char: ch, effects: ItemEffects{}, raceSquares: 6}

	sheet := deriveSheet(in)
	if !sheet.Load.Overloaded {
		t.Fatal("o controle já estava errado: o personagem não ficou sobrecarregado")
	}
	// 9m menos os 3m da sobrecarga (p141) são 6m — 4 quadrados.
	if sheet.Displacement.Total != 4 {
		t.Errorf("o deslocamento do sobrecarregado deu %d quadrados e esperava 4 (6m = 9m − 3m, p141).\n"+
			"6 quer dizer que o movimento leu uma carga ZERADA — o sistema da carga "+
			"rodou depois de quem a lê.", sheet.Displacement.Total)
	}
}
