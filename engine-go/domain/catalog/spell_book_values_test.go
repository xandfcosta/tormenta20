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
		PmCost         int    `json:"pmCost"`
		Exclusive      bool   `json:"exclusive"`
		ClassOnly      string `json:"classOnly"`
		RequiresCircle int    `json:"requiresCircle"`
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

// ONDE CADA APRIMORAMENTO MORA, para os que estavam na magia ERRADA (ALE-340).
//
// Cinco aprimoramentos existiam no catálogo pendurados na magia vizinha da que
// os tem no livro — não faltava transcrever, faltava MOVER. E eles contavam
// duas vezes no relatório do auditor: como falta numa magia e como sobra na
// outra.
//
// O que se prende aqui é a CONTAGEM de cada ponta, e ela basta para o defeito
// não voltar: ele era sempre um par, uma magia inchada e a vizinha esvaziada.
//
// A PALAVRA PRIMORDIAL com ZERO é o caso que mais precisa de guarda, e é o mais
// fácil de "consertar" de volta: o corpo dela no livro (p200) é uma lista de
// efeitos a escolher — Atordoar, Cegar, Matar —, e a magia termina ali. Uma
// lista de zero parece buraco para quem não foi ao livro.
func TestEveryMovedAugmentIsOnTheRightSpell(t *testing.T) {
	placed := []struct {
		spell string
		count int
		page  int
		// why explica a ponta: de onde veio ou para onde foi.
		why string
	}{
		{"ancora-dimensional", 5, 179, "quatro destes estavam na Amarras Etéreas, a vizinha de página"},
		{"amarras-etereas", 3, 179, "ela carregava quatro da Âncora Dimensional, e faltava o dela"},
		{"palavra-primordial", 0, 200, "o livro não lhe dá aprimoramento nenhum: o corpo é a lista de efeitos"},
		{"potencia-divina", 3, 201, "o terceiro estava na Palavra Primordial"},
		{"conjurar-mortos-vivos", 3, 186, "tinha uma CÓPIA do +1 PM da Consagrar, que já tem o dela"},
		{"consagrar", 3, 186, "o dono do +1 PM que a Conjurar Mortos-Vivos duplicava"},
	}
	spells := readSpells(t)
	for _, p := range placed {
		row, found := spells[p.spell]
		if !found {
			t.Errorf("a magia %q sumiu do catálogo", p.spell)
			continue
		}
		if got := len(row.Augments); got != p.count {
			t.Errorf("%s tem %d aprimoramentos e o livro (p%d) dá %d — %s",
				p.spell, got, p.page, p.count, p.why)
		}
	}
}

// QUANTOS APRIMORAMENTOS CADA MAGIA TEM, e por que AGORA isto pode ser preso.
//
// A issue pedia este guarda desde o começo, e ele não podia existir antes: o
// catálogo divergia do livro em catorze magias, e prender número não conferido
// é transcrever o palpite num segundo lugar. Com o `audit-spells.py` fechando em
// **198 de 198 casadas, zero divergências**, o total passou a ser um fato
// conferido — e é o total que se prende, não a tabela.
//
// A ARITMÉTICA É O CONTROLE, e é o que separa este guarda de um `expect` por
// campo: 516 é a soma do que o livro dá às 198 magias. Uma magia que perca um
// aprimoramento derruba a soma; uma que ganhe um inventado, também. Não é
// preciso listar as 198 para isso, e listá-las seria a tabela transcrita de novo.
//
// As três contagens menores existem porque cada uma protege uma regra diferente,
// e nenhuma delas se deduz do total.
func TestTheSpellCatalogKeepsWhatTheBookGives(t *testing.T) {
	spells := readSpells(t)
	if len(spells) != 198 {
		t.Fatalf("%d magias no catálogo, e o capítulo de Magia tem 198 — "+
			"os números abaixo perderam o sentido", len(spells))
	}

	total, exclusive, classOnly, requiresCircle := 0, 0, 0, 0
	for _, row := range spells {
		total += len(row.Augments)
		for _, a := range row.Augments {
			if a.Exclusive {
				exclusive++
			}
			if a.ClassOnly != "" {
				classOnly++
			}
			if a.RequiresCircle > 0 {
				requiresCircle++
			}
		}
	}

	if total != 516 {
		t.Errorf("o catálogo tem %d aprimoramentos, e o livro dá 516 às 198 magias.\n"+
			"Rode `python3 scripts/audit-spells.py` — ele diz QUAL magia e de que lado.", total)
	}
	// EXCLUSIVO: "não pode ser usado em conjunto com outros aprimoramentos"
	// (p171). São 16 — os 14 truques, que são exclusivos por definição, MAIS
	// dois que não são truque: o pó de rubi da Luz (p197) e a esfera da
	// Invisibilidade (p195). Os dois faltavam, e a ALE-340 previu isso com todas
	// as letras: eram os únicos dois exclusivos não-truque do livro, e por isso
	// não havia registro nenhum carregando a marca sem o truque junto.
	if exclusive != 16 {
		t.Errorf("%d aprimoramentos exclusivos, e o livro tem 16 (os 14 truques + o pó de "+
			"rubi da Luz e a esfera da Invisibilidade)", exclusive)
	}
	// APENAS ARCANOS / APENAS DIVINOS. Eram DOIS no catálogo inteiro quando a
	// ALE-340 abriu, e isso era o sintoma: só a Luz tem quatro deles.
	if classOnly != 5 {
		t.Errorf("%d aprimoramentos restritos a uma lista, e o livro tem 5", classOnly)
	}
	// "REQUER Nº CÍRCULO" é o que a tela desenha com cadeado em vez de esconder.
	//
	// O 143 foi CONTADO NO LIVRO pelo leitor por coordenada do
	// `audit-spells.py`, e não deduzido do catálogo — que é o que este guarda
	// existe para conferir. O `pdftotext` liso conta 120, porque ele quebra
	// "Requer 2º círculo" entre duas linhas e perde 23; os dois lados desta
	// comparação têm de ser medidos pelo MESMO instrumento.
	if requiresCircle != 143 {
		t.Errorf("%d aprimoramentos exigem círculo, e o livro marca 143", requiresCircle)
	}
}
