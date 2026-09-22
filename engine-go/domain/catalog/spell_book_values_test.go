package catalog

import (
	"encoding/json"
	"testing"
)

// OS VALORES QUE FORAM CONFERIDOS NO LIVRO, um por um (ALE-340).
//
// A regra da casa para catálogo é SCHEMA no dump, e não um `expect` por campo
// repetindo o mesmo número — prender a tabela inteira só transcreve o erro para
// um segundo lugar. O que se prende é a EXCEÇÃO, e aqui a exceção é: **estes
// seis estavam ERRADOS no catálogo e foram corrigidos contra a página do
// livro.** Um número consertado que não tem guarda volta.
//
// Eles são de duas naturezas, e a segunda é a mais cara:
//
//   - CUSTO DE PM errado, quatro casos. O jogador paga menos do que o livro
//     cobra. O `audit-spells.py` os achou comparando o TEXTO de cada
//     aprimoramento par a par — uma contagem não os veria, porque a contagem
//     bate: um aprimoramento no livro, um no catálogo.
//   - CÍRCULO errado, dois casos, e este é pior que um aprimoramento faltando:
//     o círculo decide em que NÍVEL cada classe destrava a magia, então uma
//     magia no círculo errado aparece cedo demais (ou tarde demais) na ficha de
//     quem conjura.
//
// O resto das divergências que o auditor acusa — 16 magias, com 44
// aprimoramentos faltando — não está aqui de propósito: elas ainda não foram
// conferidas contra o livro, e prender número não conferido é transcrever o
// palpite. Rode `python3 scripts/audit-spells.py` para o relatório.
type spellValueRow struct {
	Circle   int `json:"circle"`
	BookPage int `json:"bookPage"`
	Augments []struct {
		PmCost int `json:"pmCost"`
	} `json:"augments"`
}

func readSpells(t *testing.T) map[string]spellValueRow {
	t.Helper()
	raw, err := files.ReadFile("data/spells.json")
	if err != nil {
		t.Fatalf("ler spells.json: %v", err)
	}
	var rows map[string]spellValueRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatalf("spells.json não casa com o schema: %v", err)
	}
	return rows
}

// O CUSTO EM PM de um aprimoramento que o catálogo cobrava a menos.
func TestEveryCorrectedAugmentCostMatchesTheBook(t *testing.T) {
	// A ordem é a do livro, e o índice é a posição na lista do catálogo.
	corrected := []struct {
		spell  string
		index  int
		pmCost int
		page   int
		// O último campo é o valor ERRADO que estava lá. Ele fica escrito
		// porque é o que permite reconhecer a regressão de relance: voltar
		// para ele não é um número qualquer, é o defeito de novo.
		wasCharging int
	}{
		{"amedrontar", 0, 2, 179, 1},
		{"disfarce-ilusorio", 2, 2, 191, 1},
		{"enfeiticar", 1, 5, 191, 3},
		{"oracao", 0, 2, 200, 1},
	}
	spells := readSpells(t)
	for _, c := range corrected {
		row, found := spells[c.spell]
		if !found {
			t.Errorf("a magia %q sumiu do catálogo", c.spell)
			continue
		}
		if c.index >= len(row.Augments) {
			t.Errorf("%s: o aprimoramento [%d] sumiu — restaram %d",
				c.spell, c.index, len(row.Augments))
			continue
		}
		if got := row.Augments[c.index].PmCost; got != c.pmCost {
			complaint := "e o livro (p%d) diz +%d PM"
			if got == c.wasCharging {
				complaint = "que é exatamente o valor errado de antes da ALE-340; o livro (p%d) diz +%d PM"
			}
			t.Errorf("%s[%d] cobra +%d PM, "+complaint,
				c.spell, c.index, got, c.page, c.pmCost)
		}
	}
}

// O CÍRCULO de uma magia que o catálogo tinha na lista errada.
//
// Ele não é cosmético: o círculo é o que a progressão de conjuração lê para
// decidir em que nível a classe destrava a magia. Uma magia um círculo abaixo
// aparece na ficha de quem ainda não podia lançá-la.
func TestEveryCorrectedSpellCircleMatchesTheBook(t *testing.T) {
	corrected := []struct {
		spell     string
		circle    int
		page      int
		wasInList int
	}{
		// "Contato Extraplanar / Arcana 3 (Adivinhação)"
		{"contato-extraplanar", 3, 186, 2},
		// "Vitalidade Fantasma / Arcana 1 (Necromancia)"
		{"vitalidade-fantasma", 1, 211, 2},
	}
	spells := readSpells(t)
	for _, c := range corrected {
		row, found := spells[c.spell]
		if !found {
			t.Errorf("a magia %q sumiu do catálogo", c.spell)
			continue
		}
		if row.Circle != c.circle {
			complaint := "e o livro (p%d) a põe no %dº"
			if row.Circle == c.wasInList {
				complaint = "que é o círculo errado de antes da ALE-340; o livro (p%d) a põe no %dº"
			}
			t.Errorf("%s está no %dº círculo, "+complaint,
				c.spell, row.Circle, c.page, c.circle)
		}
		// A PÁGINA é a âncora da conferência: sem ela, quem duvidar do número
		// acima não sabe onde olhar. Se ela mudar, a conferência foi refeita e
		// este caso inteiro precisa ser refeito junto.
		if row.BookPage != c.page {
			t.Errorf("%s diz estar na p%d, e a conferência foi feita na p%d",
				c.spell, row.BookPage, c.page)
		}
	}
}
