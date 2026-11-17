package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"t20engine/domain/book"
	"t20engine/infra/db/sqlcgen"
)

// A PÁGINA DE PERÍCIAS NÃO REFAZ OS PODERES POR PERÍCIA (ALE-520). Os gastos de
// PM de cada perícia perguntavam a posse de cada ativação, e cada pergunta
// remontava a lista inteira de poderes e rodava o motor dos itens — ~46% da
// página nos personagens da semente, e o custo crescia com o personagem: 41 mil
// alocações por página num guerreiro 10, 61 mil num paladino 20.
//
// O teto é de ALOCAÇÕES e não de tempo: tempo varia com a máquina, e a
// contagem de alocações é a mesma em toda execução. Ele mede um paladino 20
// com as 29 perícias do livro, que é onde o custo antigo era maior.
//
// A CONTAGEM DEPENDE DA VERSÃO DO GO, e é por isso que a CI e a máquina usam a
// mesma (a do `go.mod`): com o Go 1.26 na CI e o 1.27 aqui, a mesma página contava
// ~37 mil lá e ~29,5 mil aqui, e a CI ficou vermelha por dias sem o portão local
// ver. O teto fica LONGE do uso de hoje e ainda abaixo do defeito que ele pega (61
// mil): cada fatia acrescenta algumas centenas, e um teto colado no uso reprovava
// por crescimento normal em vez de pela volta do refazer por perícia.
func TestTheExpertisesPageDoesNotRebuildThePowersPerSkill(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Paladino", 20, 0, 0)
	for _, e := range book.Expertises() {
		if _, err := f.s.queries.CreateExpertise(context.Background(), sqlcgen.CreateExpertiseParams{
			Characterid: id, Name: e.Name, Attribute: e.Attribute,
		}); err != nil {
			t.Fatalf("semear a perícia %s: %v", e.Name, err)
		}
	}
	path := fmt.Sprintf("/personagens/%d?tab=expertises", id)
	f.requests(t, f.player, http.MethodGet, path, "")

	const ceiling = 45000
	if allocs := testing.AllocsPerRun(3, func() { f.requests(t, f.player, http.MethodGet, path, "") }); allocs > ceiling {
		t.Errorf("a página de Perícias alocou %.0f vezes, e o teto é %d: os gastos de PM voltaram a refazer os poderes por perícia", allocs, ceiling)
	}
}
