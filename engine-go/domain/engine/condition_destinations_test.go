package engine

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// TODA CONDIÇÃO DO LIVRO TEM DESTINO NO APP (ver GLOSSARY, **condição**): números no
// `conditionModifierTable`, ou uma linha no `conditionRules` — a ação, o dado da vez, o mecanismo de
// fora, ou o mestre com o porquê. A conta de quantas faltavam já envelheceu uma vez sem ninguém ver;
// este guarda é o que a mantém, e falha com o NOME da condição.
//
// Ele varre os DOIS sentidos: a condição do livro sem destino, e a chave das duas tabelas que não é
// condição do livro — um "fascinada" escrito errado seria uma regra que nunca dispara.
func TestEveryBookConditionHasADestination(t *testing.T) {
	var catalog map[string]struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(bookResource(t, "conditions"), &catalog); err != nil {
		t.Fatalf("catálogo de condições ilegível: %v", err)
	}
	// O DENOMINADOR: o livro tem dezenas de condições, e um catálogo que não carregou mediria o vazio.
	if len(catalog) < 30 {
		t.Fatalf("o catálogo servido tem %d condições, e o livro tem mais de 30 (p394-395): o guarda mediria o vazio", len(catalog))
	}
	ids := make([]string, 0, len(catalog))
	for id := range catalog {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	visited := 0
	for _, id := range ids {
		visited++
		if len(conditionModifierTable[id]) > 0 || conditionRules[id].hasDestination() {
			continue
		}
		t.Errorf("%s (%q) não tem destino no app: dê a ela números no `conditionModifierTable`, ou uma linha no "+
			"`conditionRules` — a ação, o dado da vez, o mecanismo que a aplica, ou o ByMaster com o porquê", catalog[id].Name, id)
	}
	if visited != len(catalog) {
		t.Fatalf("o guarda visitou %d de %d condições", visited, len(catalog))
	}
	for id := range conditionModifierTable {
		if _, ok := catalog[id]; !ok {
			t.Errorf("o `conditionModifierTable` tem %q, que não é condição do livro: os números dela nunca valem", id)
		}
	}
	for id, rule := range conditionRules {
		if _, ok := catalog[id]; !ok {
			t.Errorf("o `conditionRules` tem %q, que não é condição do livro: a regra dela nunca dispara", id)
		}
		if !rule.hasDestination() {
			t.Errorf("a linha de %q no `conditionRules` está vazia, e diria um destino que não existe", id)
		}
	}
}

// O DADO DA VEZ das condições (p394): o 1d6 de fogo do Em Chamas, e o 1d6 do Confuso com a tabela.
func TestTheConditionsRollTheirDieAtTheStartOfTheTurn(t *testing.T) {
	rolls := ConditionTurnRolls([]string{"abalado", "em-chamas", "confuso"})
	if len(rolls) != 2 {
		t.Fatalf("abalado, em chamas e confuso rolam %+v; want os dois dados da vez", rolls)
	}
	burning, confused := rolls[0], rolls[1]
	if burning.ID != "em-chamas" || burning.Name != "Em Chamas" || burning.Dice != "1d6" || burning.Kind != "fogo" {
		t.Errorf("as chamas rolam %+v; want 1d6 de fogo", burning)
	}
	if confused.ID != "confuso" || confused.Name != "Confuso" || confused.Dice != "1d6" || confused.Kind != "" {
		t.Errorf("o confuso rola %+v; want 1d6 sem tipo", confused)
	}
}

// A TABELA DO CONFUSO, resultado a resultado (p394).
func TestTheConfusedTableSaysWhatEachResultDoes(t *testing.T) {
	for d6, want := range map[int]string{
		1: "1d8", 2: "não age", 3: "não age", 4: "ataca a criatura mais próxima",
		5: "ataca a criatura mais próxima", 6: "a condição termina",
	} {
		if got := ConditionRollSays("confuso", d6); !strings.Contains(got, want) {
			t.Errorf("confuso com %d no 1d6 diz %q; want algo com %q", d6, got, want)
		}
	}
	if got := ConditionRollSays("em-chamas", 3); got != "" {
		t.Errorf("as chamas não têm tabela, e a frase veio %q", got)
	}
}
