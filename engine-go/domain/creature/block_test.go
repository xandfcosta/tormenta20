package creature

import (
	"strings"
	"testing"
)

// O BLOCO IMPOSSÍVEL É RECUSADO, e a mensagem nomeia o valor ofensor.
//
// Ele veio do `api/creatures_http_test.go` na ALE-277. Lá media o 400 de
// `POST /campanhas/{id}/creatures`, e a rota saiu com as outras sem consumidor;
// a garantia desceu para onde a regra MORA. Quem a chama hoje é o editor de NPC
// da Mesa (`creature.Validate`, em `web/table/npc_editor.go`), e ele é o único
// caminho por onde um bloco entra.
//
// **A varredura é por CAMPO e não por caso feliz**: cada linha aqui é uma das
// recusas do `Validate`, e o CONTROLE no fim prova que um bloco bom passa —
// sem ele, tudo isto seria verdade sobre uma função que recusa qualquer coisa.
func TestAnImpossibleBlockIsRefusedNamingTheOffender(t *testing.T) {
	good := func() Block { return Block{ND: 1, Kind: "monstro", Size: "medio", HP: 10} }

	cases := []struct {
		name  string
		who   string
		block Block
		waits string
	}{
		{"sem nome", "  ", good(), "nome"},
		{"nome comprido demais", strings.Repeat("a", 61), good(), "60"},
		{"tipo fora do livro", "X", Block{ND: 1, Kind: "dragao", Size: "medio", HP: 10}, "dragao"},
		{"tamanho fora do livro", "X", Block{ND: 1, Kind: "monstro", Size: "gigante", HP: 10}, "gigante"},
		{"sem vida", "X", Block{ND: 1, Kind: "monstro", Size: "medio", HP: 0}, "PV"},
		{"ND negativo", "X", Block{ND: -1, Kind: "monstro", Size: "medio", HP: 10}, "ND"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := c.block
			err := Validate(c.who, &b)
			if err == nil {
				t.Fatalf("o bloco %+v passou, e ele é impossível", b)
			}
			if !strings.Contains(err.Error(), c.waits) {
				t.Errorf("a recusa não nomeia %q: %v", c.waits, err)
			}
		})
	}

	// O CONTROLE: um bloco do livro passa.
	b := good()
	if err := Validate("Bandido", &b); err != nil {
		t.Fatalf("um bloco válido foi recusado: %v", err)
	}
}
