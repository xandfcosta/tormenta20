package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"t20engine/infra/db/sqlcgen"
)

// addToBag adiciona do livro. O sinal da quantidade vai junto porque ele continua
// vivo na página, por causa do formulário do Editar, e não pode valer aqui.
func addToBag(t *testing.T, f sceneFixture, id int64, catalogID string) string {
	t.Helper()
	target := fmt.Sprintf("/personagens/%d/itens/adiciona/%s?tab=bag", id, catalogID)
	return bagRefusal(t, f.requests(t, f.player, http.MethodPost, target, `{"item_qty":7}`).Body.String())
}

// bagLinesOf são as linhas da mochila com este nome.
func bagLinesOf(t *testing.T, f sceneFixture, id int64, name string) []sqlcgen.ListItemsByCharacterRow {
	t.Helper()
	items, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("listar os itens: %v", err)
	}
	var lines []sqlcgen.ListItemsByCharacterRow
	for _, item := range items {
		if item.Name == name {
			lines = append(lines, item)
		}
	}
	return lines
}

func TestAddingTheSameItemAgainSumsOnItsLine(t *testing.T) {
	f, id := fighterFixture(t)

	for range 2 {
		if refusal := addToBag(t, f, id, "fogo-alquimico"); refusal != "" {
			t.Fatalf("adicionar foi recusado: %q", refusal)
		}
	}

	lines := bagLinesOf(t, f, id, "Fogo alquímico")
	if len(lines) != 1 {
		t.Fatalf("dois gestos criaram %d linhas, e o item é um só", len(lines))
	}
	if lines[0].Quantity != 2 {
		t.Errorf("dois gestos deram quantidade %d", lines[0].Quantity)
	}
}

// Oito pedidos porque um par é pouco para expor uma corrida entre ler a mochila e
// gravar a soma. Sem transação, um gesto se perde ou vira uma segunda linha.
func TestSimultaneousAddsCountEveryGesture(t *testing.T) {
	f, id := fighterFixture(t)

	const gestures = 8
	refusals := make([]string, gestures)
	var adding sync.WaitGroup
	for n := range gestures {
		adding.Add(1)
		go func() {
			defer adding.Done()
			refusals[n] = addToBag(t, f, id, "fogo-alquimico")
		}()
	}
	adding.Wait()

	for _, refusal := range refusals {
		if refusal != "" {
			t.Fatalf("um gesto simultâneo foi recusado: %q", refusal)
		}
	}
	lines := bagLinesOf(t, f, id, "Fogo alquímico")
	if len(lines) != 1 || lines[0].Quantity != gestures {
		t.Errorf("%d gestos deram %+v, e esperava uma linha com %d", gestures, lines, gestures)
	}
}

// A linha guarda o lugar no corpo, as melhorias e o material, e uma pilha não se
// divide: duas espadas curtas numa linha só não iriam para as duas mãos.
func TestEachWeaponArmorAndShieldKeepsItsOwnLine(t *testing.T) {
	cases := map[string]string{
		"espada-curta":   "Espada curta",
		"armadura-couro": "Armadura de couro",
		"escudo-leve":    "Escudo leve",
	}
	for catalogID, name := range cases {
		t.Run(catalogID, func(t *testing.T) {
			f, id := fighterFixture(t)
			for range 2 {
				if refusal := addToBag(t, f, id, catalogID); refusal != "" {
					t.Fatalf("adicionar foi recusado: %q", refusal)
				}
			}
			lines := bagLinesOf(t, f, id, name)
			if len(lines) != 2 {
				t.Fatalf("dois gestos criaram %d linhas de %q, e cada um é um item", len(lines), name)
			}
			for _, line := range lines {
				if line.Quantity != 1 {
					t.Errorf("%q entrou com quantidade %d, que só muda editando o item", name, line.Quantity)
				}
			}
		})
	}
}

// Dois itens caseiros são diferentes, mesmo com o mesmo nome.
func TestHomemadeItemsNeverSum(t *testing.T) {
	f, id := fighterFixture(t)

	for range 2 {
		if refusal := customItem(t, f, id, `{"item_name":"Pena","item_qty":1,"item_slots":0.5}`); refusal != "" {
			t.Fatalf("o item caseiro foi recusado: %q", refusal)
		}
	}

	if lines := bagLinesOf(t, f, id, "Pena"); len(lines) != 2 {
		t.Errorf("dois itens caseiros viraram %d linhas", len(lines))
	}
}

// Somar numa linha vestida, melhorada ou editada daria à peça nova o lugar, as
// melhorias ou o nome da outra, que ninguém pediu.
func TestALineThatIsNoLongerPlainDoesNotReceiveTheSum(t *testing.T) {
	changes := map[string]string{
		"vestida":            `UPDATE character_items SET equipped = 'vested' WHERE characterId = ?`,
		"melhorada":          `UPDATE character_items SET improvements = '["melhoria-discreta"]' WHERE characterId = ?`,
		"com material":       `UPDATE character_items SET material = 'material-mitral' WHERE characterId = ?`,
		"renomeada":          `UPDATE character_items SET name = 'Bandana da Ana' WHERE characterId = ?`,
		"com outros espaços": `UPDATE character_items SET slots = slots + 1 WHERE characterId = ?`,
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			f, id := fighterFixture(t)
			if refusal := addToBag(t, f, id, "bandana"); refusal != "" {
				t.Fatalf("adicionar foi recusado: %q", refusal)
			}
			if _, err := f.s.db.ExecContext(context.Background(), change, id); err != nil {
				t.Fatalf("mudar a linha: %v", err)
			}

			if refusal := addToBag(t, f, id, "bandana"); refusal != "" {
				t.Fatalf("adicionar de novo foi recusado: %q", refusal)
			}

			items, err := f.s.sceneCore().Queries().ListItemsByCharacter(context.Background(), id)
			if err != nil {
				t.Fatalf("listar os itens: %v", err)
			}
			if len(items) != 2 {
				t.Fatalf("a peça nova virou %d linhas, e a linha %s não recebe soma", len(items), name)
			}
			if items[0].Quantity != 1 || items[1].Quantity != 1 {
				t.Errorf("as quantidades ficaram %d e %d", items[0].Quantity, items[1].Quantity)
			}
		})
	}
}

// As linhas repetidas que já existem ficam como estão: a soma vai só para a primeira.
func TestTheSumGoesToTheFirstPlainLine(t *testing.T) {
	f, id := fighterFixture(t)
	if refusal := addToBag(t, f, id, "bandana"); refusal != "" {
		t.Fatalf("adicionar foi recusado: %q", refusal)
	}
	if _, err := f.s.db.ExecContext(context.Background(),
		`INSERT INTO character_items (characterId, catalogId, name, quantity, slots, improvements, createdAt)
		 SELECT characterId, catalogId, name, quantity, slots, improvements, createdAt FROM character_items WHERE characterId = ?`, id); err != nil {
		t.Fatalf("semear a linha repetida: %v", err)
	}

	if refusal := addToBag(t, f, id, "bandana"); refusal != "" {
		t.Fatalf("adicionar de novo foi recusado: %q", refusal)
	}

	lines := bagLinesOf(t, f, id, "Bandana")
	if len(lines) != 2 || lines[0].Quantity != 2 || lines[1].Quantity != 1 {
		t.Errorf("as linhas ficaram %+v, e esperava 2 e 1", lines)
	}
}

func TestTheSumDoesNotPassTheQuantityCeiling(t *testing.T) {
	f, id := fighterFixture(t)
	if refusal := addToBag(t, f, id, "traje-viajante"); refusal != "" {
		t.Fatalf("adicionar foi recusado: %q", refusal)
	}
	if _, err := f.s.db.ExecContext(context.Background(),
		`UPDATE character_items SET quantity = 9999 WHERE characterId = ?`, id); err != nil {
		t.Fatalf("semear a quantidade: %v", err)
	}

	refusal := addToBag(t, f, id, "traje-viajante")
	if !strings.Contains(refusal, "10000") || !strings.Contains(refusal, "9999") {
		t.Errorf("a recusa foi %q, e devia dizer a soma e o teto", refusal)
	}
}
