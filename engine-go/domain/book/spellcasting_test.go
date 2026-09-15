package book

import "testing"

// A PROGRESSÃO SAI DO CATÁLOGO, com as cinco classes que conjuram.
//
// Os números são os que estavam no front e foram MOVIDOS na ALE-272 — nenhum foi
// lido do livro ali, e o que garantiu a fidelidade da mudança foi um teste do
// front que comparava as duas cópias. Ele morreu com a SPA (ALE-272, fatia 10c),
// então esta tabela não tem mais segunda opinião: uma auditoria contra o livro
// continua sendo outro trabalho.
//
// O que este caso prende é que a LEITURA funciona: um `classes.json` sem o
// campo, ou um `Resource` que parasse de casar, daria um mapa vazio e o portão
// de círculo aceitaria tudo em silêncio.
func TestTheCircleProgressionComesFromTheCatalog(t *testing.T) {
	prog := SpellProgressions()
	if len(prog) != 5 {
		t.Fatalf("o catálogo ofereceu %d classes conjuradoras, quer 5", len(prog))
	}
	for _, nome := range []string{"Arcanista", "Bardo", "Clérigo", "Druida", "Paladino"} {
		if _, tem := prog[nome]; !tem {
			t.Errorf("a classe %q não trouxe progressão", nome)
		}
	}
	// O Bardo PARA no 4º círculo, e é o caso que separa "não alcançou ainda" de
	// "nunca alcança": o 5º dele é nulo, não um nível alto.
	bardo := prog["Bardo"]
	if bardo.MaxCircle != 4 {
		t.Errorf("o Bardo vai até o %dº círculo, quer o 4º", bardo.MaxCircle)
	}
	if bardo.UnlockLevel["5"] != nil {
		t.Error("o 5º círculo do Bardo veio com nível: ele nunca chega lá")
	}
}
