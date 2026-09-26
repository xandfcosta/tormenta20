package catalog_test

import (
	"encoding/json"
	"testing"

	"t20engine/domain/catalog"
)

// TestEveryPowerAGodGrantsIsAValidSeedReference (ALE-397).
//
// O `GrantedPowerNames` é a lista contra a qual a seed valida `create.godPower`.
// Ele lia o `granted-powers.json`, que tinha 36 dos 72 nomes — e o resultado
// era uma seed de devoto de Marah em diante ser RECUSADA como referência
// quebrada de um catálogo que tem o poder.
//
// O defeito não estava na seed: estava na lista. O guarda prende a invariante
// de que a lista e o que os deuses de fato concedem são a MESMA coisa, e falha
// com o nome de cada poder que sobrar — a diferença entre "conserte isto" e
// "procure".
func TestEveryPowerAGodGrantsIsAValidSeedReference(t *testing.T) {
	raw, ok := catalog.Resource("gods")
	if !ok {
		t.Fatal("catálogo de deuses ausente")
	}
	var gods []struct {
		Name    string   `json:"name"`
		Granted []string `json:"poderesConcedidos"`
	}
	if err := json.Unmarshal(raw, &gods); err != nil {
		t.Fatalf("deuses: %v", err)
	}

	aceitos := map[string]bool{}
	for _, nome := range catalog.GrantedPowerNames() {
		aceitos[nome] = true
	}

	// O CONTROLE, nos dois lados: uma lista vazia de qualquer um deles faria o
	// laço abaixo não comparar nada e sair verde.
	if len(gods) != 20 {
		t.Fatalf("%d deuses — o livro tem 20, e o caso mediria quase nada", len(gods))
	}
	if len(aceitos) < 70 {
		t.Fatalf("o validador aceita %d nomes — os 20 deuses concedem 4 cada, "+
			"então a lista não pode ser tão curta", len(aceitos))
	}

	for _, deus := range gods {
		for _, poder := range deus.Granted {
			if !aceitos[poder] {
				t.Errorf("%s concede %q, e o validador da seed não aceita esse nome: "+
					"uma seed de devoto desse deus sai como referência quebrada",
					deus.Name, poder)
			}
		}
	}
}
