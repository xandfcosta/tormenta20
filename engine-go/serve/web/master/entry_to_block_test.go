package master

import (
	"encoding/json"
	"os"
	"reflect"
	"t20engine/domain/book"
	"t20engine/domain/creature"
	"testing"
)

// O ORÁCULO DA CÓPIA DO VERBETE, e hoje ele é LINHA DE BASE: acusa qualquer
// campo que mude sem ter sido pedido.
//
// Uma divergência aqui não aparece na tela — os dois blocos são JSON válido —, e
// o bloco copiado vai para a mesma coluna do mesmo banco.

type oraculoDaCopia struct {
	Cases []struct {
		ID    string         `json:"id"`
		Note  string         `json:"nota"`
		Block creature.Block `json:"bloco"`
	} `json:"casos"`
}

func TestTheEntryCopyMatchesTheJs(t *testing.T) {
	raw, err := os.ReadFile("testdata/entry-to-block-from-the-js.json")
	if err != nil {
		t.Fatalf("oráculo ausente — ele é versionado e não se regenera mais: %v", err)
	}
	var oracle oraculoDaCopia
	if err := json.Unmarshal(raw, &oracle); err != nil {
		t.Fatalf("oráculo ilegível: %v", err)
	}
	// O CONTROLE: um oráculo vazio faria o laço passar verde sem comparar nada.
	if len(oracle.Cases) == 0 {
		t.Fatal("o oráculo está vazio — o laço abaixo não compararia nada")
	}
	bookRef := book.Creatures()
	if len(bookRef) == 0 {
		t.Fatal("o bestiário do Go veio vazio — não há de onde copiar")
	}

	for _, tc := range oracle.Cases {
		t.Run(tc.ID+" — "+tc.Note, func(t *testing.T) {
			v := chosenOrFirst(bookRef, tc.ID)
			if v == nil || v.ID != tc.ID {
				t.Fatalf("o verbete %q não existe no bestiário do Go", tc.ID)
			}
			mine := CopyOfEntry(*v)
			if reflect.DeepEqual(mine, tc.Block) {
				return
			}
			doJS, _ := json.Marshal(tc.Block)
			doGo, _ := json.Marshal(mine)
			t.Errorf("a cópia divergiu\n  o JS dá: %s\n  o Go dá: %s", doJS, doGo)
		})
	}
}

// O catálogo é EMBUTIDO e servido à mesa inteira; o bloco nasce para ser
// editado. Se os dois dividissem o mesmo array, o mestre mexer no ataque do seu
// "Ogro Capitão" mexeria no verbete que o bestiário desenha para todo mundo — e
// a fonte do livro passaria a mentir para quem a consultasse depois, sem nada
// na tela dizendo por quê.
//
// `reflect.DeepEqual` não pega isto: duas fatias que compartilham memória são
// profundamente iguais. Só a MUTAÇÃO revela.
func TestTheCopyDoesNotShareASliceWithTheCatalog(t *testing.T) {
	bookRef := book.Creatures()
	v := chosenOrFirst(bookRef, "ogro")
	if v == nil || len(v.Attacks) == 0 {
		t.Skip("o ogro do catálogo não tem ataque para mexer")
	}
	original := v.Attacks[0].Name

	block := CopyOfEntry(*v)
	block.Attacks[0].Name = "MEXIDO PELO MESTRE"

	if v.Attacks[0].Name != original {
		t.Errorf("editar o bloco do mestre reescreveu o VERBETE do livro: %q virou %q",
			original, v.Attacks[0].Name)
	}
}
