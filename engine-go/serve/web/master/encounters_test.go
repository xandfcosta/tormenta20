package master

import (
	"net/http"
	"strings"
	"testing"
)

// Os guardas do CONSTRUTOR DE ENCONTROS.
//
// A conta em si é do `engine` e tem os testes dela lá, contra o livro. O que se
// prende aqui é a ÁLGEBRA DO RASCUNHO e a tradução do gesto — que é onde um
// erro apaga o encontro do mestre sem avisar.

func encontroDe(t *testing.T, rows []encounterRow) encountersView {
	t.Helper()
	return loadEncounters(nivelPadrao, grupoPadrao, rows, "")
}

// A mesma criatura SOBE A CONTAGEM, e não vira uma segunda linha.
//
// Duas linhas do mesmo verbete calculariam cada uma o próprio ND de grupo, e a
// regra da dobra (p282) só significa alguma coisa sobre UM grupo: dois grupos
// de dois ogros valeriam MENOS que um grupo de quatro, que é o oposto da regra.
// É por isso que este guarda mede o ND e não só a contagem de linhas.
func TestTheSameCreatureRaisesTheCount(t *testing.T) {
	rows := []encounterRow{}
	for i := 0; i < 4; i++ {
		rows = addRow(rows, "ogro")
	}
	if len(rows) != 1 {
		t.Fatalf("%d linhas, quero 1 com quantidade 4", len(rows))
	}
	if rows[0].Qtd != 4 {
		t.Fatalf("quantidade %d, quero 4", rows[0].Qtd)
	}

	partyOfFour := encontroDe(t, rows).ND()
	twoPairs := encontroDe(t, []encounterRow{
		{ID: "ogro", Qtd: 2}, {ID: "ogro", Qtd: 2},
	}).ND()
	if partyOfFour >= twoPairs {
		t.Errorf("um grupo de quatro deu ND %v e dois de dois deram %v — a regra da dobra "+
			"exige que juntar seja MAIS perigoso", partyOfFour, twoPairs)
	}
}

// Um grupo de zero criaturas não é um grupo, e
// deixar a linha com 0 mostraria "ND 0" numa linha que ainda parece parte do
// encontro.
func TestTheLastOneRemovedTakesTheRow(t *testing.T) {
	rows := []encounterRow{{ID: "ogro", Qtd: 2}}
	rows = lessRow(rows, "ogro")
	if len(rows) != 1 || rows[0].Qtd != 1 {
		t.Fatalf("depois de um passo: %+v", rows)
	}
	rows = lessRow(rows, "ogro")
	if len(rows) != 0 {
		t.Errorf("a linha sobreviveu ao último passo: %+v", rows)
	}
}

// Um id velho colado numa URL renderizaria uma linha sem nome com quantidade
// viva.
func TestAnEntryThatVanishedDoesNotBecomeAnEmptyRow(t *testing.T) {
	v := encontroDe(t, []encounterRow{
		{ID: "ogro", Qtd: 1},
		{ID: "dragao-de-papel-machê", Qtd: 3},
	})
	if len(v.Rows) != 1 {
		t.Fatalf("%d linhas, quero só a que existe", len(v.Rows))
	}
	if v.Rows[0].Entry.Name == "" {
		t.Error("linha sem nome sobreviveu")
	}
}

// O ciclo inteiro: montar, copiar, colar.
//
// Ele existe porque o formato do link é escrito num lugar e lido em outro, e é
// exatamente aí que um `:` vira `-` e o encontro chega vazio do outro lado sem
// nenhum erro.
func TestTheCopiedLinkReopensTheEncounter(t *testing.T) {
	original := loadEncounters(5, 3, []encounterRow{
		{ID: "ogro", Qtd: 2}, {ID: "goblin-salteador", Qtd: 4},
	}, "")

	link := encounterAddress(original)
	_, query, found := strings.Cut(link, "?")
	if !found {
		t.Fatalf("o link não tem query: %q", link)
	}

	// A volta: o que o `?c=` carrega tem de reconstruir a mesma composição.
	var c string
	for _, par := range strings.Split(query, "&") {
		if key, value, _ := strings.Cut(par, "="); key == "c" {
			c = strings.ReplaceAll(value, "%3A", ":")
			c = strings.ReplaceAll(c, "%2C", ",")
		}
	}
	back := loadEncounters(5, 3, rowsFromURL(c), "")

	if len(back.Rows) != len(original.Rows) {
		t.Fatalf("voltaram %d linhas de %d", len(back.Rows), len(original.Rows))
	}
	if back.ND() != original.ND() {
		t.Errorf("o ND mudou na volta: %v virou %v", original.ND(), back.ND())
	}
	for i := range original.Rows {
		if back.Rows[i].Entry.ID != original.Rows[i].Entry.ID ||
			back.Rows[i].Qtd != original.Rows[i].Qtd {
			t.Errorf("linha %d voltou diferente: %+v", i, back.Rows[i])
		}
	}
}

// O link chega por chat, e um caractere a mais não pode zerar o que veio junto.
func TestACrookedLinkDoesNotCostTheWholeEncounter(t *testing.T) {
	rows := rowsFromURL("ogro:2,lixo,goblin-salteador:x,,cascavel:3")
	if len(rows) != 2 {
		t.Fatalf("%d linhas de duas boas: %+v", len(rows), rows)
	}
	if rows[0].ID != "ogro" || rows[1].ID != "cascavel" {
		t.Errorf("as linhas boas não sobreviveram: %+v", rows)
	}
}

// Nível e tamanho vêm da URL, que qualquer um edita à mão, e um nível 999
// mudaria a dificuldade sem mudar o encontro.
func TestAnAbsurdLevelAndSizeFallBackToTheDefault(t *testing.T) {
	v := loadEncounters(999, -3, nil, "")
	if v.Level != nivelPadrao || v.Group != grupoPadrao {
		t.Errorf("nível %d e grupo %d, quero os padrões %d e %d",
			v.Level, v.Group, nivelPadrao, grupoPadrao)
	}
}

// Mostrar as 80 criaturas abaixo da composição empurraria o VEREDITO para fora
// da tela, e o veredito é o assunto da ferramenta.
func TestTheSearchPanelOnlyShowsWithATerm(t *testing.T) {
	if got := loadEncounters(1, 4, nil, "").Findings; len(got) != 0 {
		t.Errorf("sem termo vieram %d criaturas", len(got))
	}
	if got := loadEncounters(1, 4, nil, "ogro").Findings; len(got) == 0 {
		t.Error("com termo não veio criatura nenhuma")
	}
}

// ── pelo fio ─────────────────────────────────────────────────────────────────

// O link colado no chat abre montado, numa carga fria.
func TestTheEncounterInTheUrlHoldsOnAColdLoad(t *testing.T) {
	rec := pedeNaCena(t, "/mestre/encontros?nivel=3&grupo=4&c=ogro:2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Ogro") {
		t.Error("o encontro do link não foi desenhado")
	}
	// A dificuldade vai CALCULADA À MÃO pelo livro, e não colhida de
	// `loadEncounters`:
	//
	//	ogro ND 4, dois deles → 4 + 2·log2(2) = ND 6 (p282)
	//	6 − nível 3 = diferença 3 → acima da faixa "Difícil" → Mortal (p281)
	//
	// Chamar a mesma função que a página chama e afirmar que a página contém o
	// resultado dela fica verde mesmo com a conta errada, porque os dois lados
	// erram junto.
	const dificuldadeEsperada = "Mortal"
	if !strings.Contains(body, dificuldadeEsperada) {
		t.Errorf("dois ogros contra um grupo de nível 3 são %q pelo livro, e a página não diz",
			dificuldadeEsperada)
	}
}
