package book

import (
	"slices"
	"strings"
	"testing"
)

// TODO PODER QUE UM ITEM MUDA EXISTE. Um id de ativação errado no verbete não estoura: o item fica na
// mochila e simplesmente não muda nada, e só quem lê a linha do poder com o item na mão percebe.
func TestEveryPowerAnItemChangesExists(t *testing.T) {
	measured := 0
	for _, item := range Catalogs().Items {
		for _, change := range item.PowerChanges {
			for _, id := range change.Powers {
				measured++
				if ActivationOf(id, "") == nil {
					t.Errorf("%s muda o poder %q, que não é ativação do catálogo", item.ID, id)
				}
			}
		}
	}
	// O DENOMINADOR: uma lista de reprovados vazia e um catálogo sem o campo se parecem no terminal.
	if measured == 0 {
		t.Fatal("nenhum item muda poder nenhum — o campo powerChanges não chegou do catálogo")
	}
}

// O TAMBOR DOBRA O ALCANCE "de qualquer Música de Bardo" (p158), e a lista do verbete é escrita à mão:
// uma Música que entre no catálogo depois precisa entrar nela também, e é aqui que ela reprova.
func TestTheDeepDrumReachesEveryBardMusic(t *testing.T) {
	drum := ItemByID("tambor-das-profundezas")
	if drum == nil || len(drum.PowerChanges) != 1 {
		t.Fatalf("o Tambor das Profundezas não está no catálogo com uma mudança de poder: %+v", drum)
	}
	reached := drum.PowerChanges[0].Powers
	musics := 0
	for _, spec := range Activations() {
		if !strings.HasPrefix(spec.Name, "Música:") {
			continue
		}
		musics++
		if !slices.Contains(reached, spec.ID) {
			t.Errorf("o tambor não alcança %s (%s)", spec.Name, spec.ID)
		}
	}
	if musics == 0 {
		t.Fatal("nenhuma ativação se chama Música: — o prefixo mudou, e este guarda não mede nada")
	}
}
