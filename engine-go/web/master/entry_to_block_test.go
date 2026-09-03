package master

import (
	"encoding/json"
	"os"
	"reflect"
	"t20engine/book"
	"t20engine/creature"
	"testing"
)

// O ORÁCULO DA CÓPIA DO VERBETE (ALE-269, superfície 6b).
//
// Ele nasceu como PARIDADE: enquanto as duas telas existiam, o mestre podia
// copiar o Ogro pela SPA numa noite e pelo piloto na outra, e o bloco ia para a
// MESMA coluna do MESMO banco. Duas cópias diferentes do mesmo verbete é a
// divergência mais cara possível aqui, porque ela não aparece — os dois blocos
// são JSON válido. O esperado era MEDIDO rodando o `creature-from-monster.ts`.
//
// Com a SPA apagada (ALE-272, fatia 10c) sobrou uma tela só, e o oráculo virou
// LINHA DE BASE: ele acusa qualquer campo que mude sem ter sido pedido, e não
// prova mais que dois lados concordam. O script que o gerava saiu junto.

type oraculoDaCopia struct {
	Casos []struct {
		ID    string         `json:"id"`
		Nota  string         `json:"nota"`
		Bloco creature.Block `json:"bloco"`
	} `json:"casos"`
}

func TestTheEntryCopyMatchesTheJs(t *testing.T) {
	bruto, err := os.ReadFile("testdata/entry-to-block-from-the-js.json")
	if err != nil {
		t.Fatalf("oráculo ausente — ele é versionado e não se regenera mais: %v", err)
	}
	var oraculo oraculoDaCopia
	if err := json.Unmarshal(bruto, &oraculo); err != nil {
		t.Fatalf("oráculo ilegível: %v", err)
	}
	// O CONTROLE: um oráculo vazio faria o laço passar verde sem comparar nada.
	if len(oraculo.Casos) == 0 {
		t.Fatal("o oráculo está vazio — o laço abaixo não compararia nada")
	}
	livro := book.Creatures()
	if len(livro) == 0 {
		t.Fatal("o bestiário do Go veio vazio — não há de onde copiar")
	}

	for _, caso := range oraculo.Casos {
		t.Run(caso.ID+" — "+caso.Nota, func(t *testing.T) {
			v := chosenOrFirst(livro, caso.ID)
			if v == nil || v.ID != caso.ID {
				t.Fatalf("o verbete %q não existe no bestiário do Go", caso.ID)
			}
			meu := CopyOfEntry(*v)
			if reflect.DeepEqual(meu, caso.Bloco) {
				return
			}
			doJS, _ := json.Marshal(caso.Bloco)
			doGo, _ := json.Marshal(meu)
			t.Errorf("a cópia divergiu\n  o JS dá: %s\n  o Go dá: %s", doJS, doGo)
		})
	}
}

// TestTheCopyDoesNotShareASliceWithTheCatalog.
//
// O catálogo é EMBUTIDO e servido à mesa inteira; o bloco nasce para ser
// editado. Se os dois dividissem o mesmo array, o mestre mexer no ataque do seu
// "Ogro Capitão" mexeria no verbete que o bestiário desenha para todo mundo — e
// a fonte do livro passaria a mentir para quem a consultasse depois, sem nada
// na tela dizendo por quê.
//
// `reflect.DeepEqual` não pega isto: duas fatias que compartilham memória são
// profundamente iguais. Só a MUTAÇÃO revela.
func TestTheCopyDoesNotShareASliceWithTheCatalog(t *testing.T) {
	livro := book.Creatures()
	v := chosenOrFirst(livro, "ogro")
	if v == nil || len(v.Attacks) == 0 {
		t.Skip("o ogro do catálogo não tem ataque para mexer")
	}
	original := v.Attacks[0].Name

	bloco := CopyOfEntry(*v)
	bloco.Attacks[0].Name = "MEXIDO PELO MESTRE"

	if v.Attacks[0].Name != original {
		t.Errorf("editar o bloco do mestre reescreveu o VERBETE do livro: %q virou %q",
			original, v.Attacks[0].Name)
	}
}
