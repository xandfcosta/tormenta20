package book

import "testing"

// A FLAG DECLARADA E A DOS MODIFICADORES SÃO A MESMA.
//
// A `Activation.Flag` passou a ser declarada porque três das seis Posturas de
// Combate (p54) não têm modificador de onde derivá-la. O preço disso é que
// agora existem DOIS lugares dizendo qual flag a postura acende — e duas
// grafias de um conceito é o que o GLOSSARY proíbe.
//
// O defeito que elas produzem é calado nos DOIS sentidos: com a declarada
// errada, a postura acende uma flag que modificador nenhum escuta e o
// personagem paga 2 PM por nada; com os modificadores errados, o número muda
// sozinho e ninguém consegue desligá-lo.
//
// DENOMINADOR: ele diz quantas olhou. Com zero posturas declaradas, "nenhuma
// divergiu" e "não mediu nada" são a mesma cor.
func TestEveryDeclaredStanceFlagMatchesItsModifiers(t *testing.T) {
	flags := ClassPowerFlags()
	measured := 0
	for _, a := range Activations() {
		if a.Kind != "stance" || a.Flag == "" {
			continue
		}
		measured++
		doPoder, temModificador := flags[a.ID]
		if !temModificador {
			// SEM MODIFICADOR NÃO HÁ O QUE CASAR, e é o caso que fez a
			// declaração existir: a Castigo de Ferro e a Provocação Petulante
			// não mexem em número nenhum — o que elas fazem é do mestre.
			continue
		}
		if doPoder != a.Flag {
			t.Errorf("%s declara a flag %q e os modificadores dele usam %q",
				a.ID, a.Flag, doPoder)
		}
	}
	if measured == 0 {
		t.Fatal("nenhuma postura declara flag — o guarda não mediu nada")
	}
}

// TODA POSTURA DE UM GRUPO CONHECE AS IRMÃS, e o grupo tem mais de uma.
//
// Um grupo com uma postura só é um grupo que não exclui ninguém — ele passaria
// por regra de exclusividade sem nunca excluir nada, que é a forma mais cara
// de um erro de digitação no nome do grupo aparecer.
func TestEveryStanceGroupHasMoreThanOneStance(t *testing.T) {
	size := map[string]int{}
	for _, s := range StancesFromCatalog() {
		if s.Group != "" {
			size[s.Group]++
		}
	}
	if len(size) == 0 {
		t.Fatal("nenhum grupo de posturas — o guarda não mediu nada")
	}
	for group, n := range size {
		if n < 2 {
			t.Errorf("o grupo %q tem %d postura: ou o nome está errado numa delas, "+
				"ou o grupo não precisa existir", group, n)
		}
	}
}
