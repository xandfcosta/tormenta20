package master

import (
	"net/http"
	"strings"
	"testing"
)

// Os guardas do CONSTRUTOR DE ENCONTROS (ALE-259).
//
// A conta em si é do `engine` e tem os testes dela lá, contra o livro. O que se
// prende aqui é a ÁLGEBRA DO RASCUNHO e a tradução do gesto — que é onde um
// erro apaga o encontro do mestre sem avisar.

func encontroDe(t *testing.T, linhas []encounterRow) encountersView {
	t.Helper()
	return loadEncounters(nivelPadrao, grupoPadrao, linhas, "")
}

// TestTheSameCreatureRaisesTheCount, e não vira uma segunda linha.
//
// Duas linhas do mesmo verbete calculariam cada uma o próprio ND de grupo, e a
// regra da dobra (p282) só significa alguma coisa sobre UM grupo: dois grupos
// de dois ogros valeriam MENOS que um grupo de quatro, que é o oposto da regra.
// É por isso que este guarda mede o ND e não só a contagem de linhas.
func TestTheSameCreatureRaisesTheCount(t *testing.T) {
	linhas := []encounterRow{}
	for i := 0; i < 4; i++ {
		linhas = addRow(linhas, "ogro")
	}
	if len(linhas) != 1 {
		t.Fatalf("%d linhas, quero 1 com quantidade 4", len(linhas))
	}
	if linhas[0].Qtd != 4 {
		t.Fatalf("quantidade %d, quero 4", linhas[0].Qtd)
	}

	umGrupoDeQuatro := encontroDe(t, linhas).ND()
	doisGruposDeDois := encontroDe(t, []encounterRow{
		{ID: "ogro", Qtd: 2}, {ID: "ogro", Qtd: 2},
	}).ND()
	if umGrupoDeQuatro >= doisGruposDeDois {
		t.Errorf("um grupo de quatro deu ND %v e dois de dois deram %v — a regra da dobra "+
			"exige que juntar seja MAIS perigoso", umGrupoDeQuatro, doisGruposDeDois)
	}
}

// TestTheLastOneRemovedTakesTheRow: um grupo de zero criaturas não é um grupo, e
// deixar a linha com 0 mostraria "ND 0" numa linha que ainda parece parte do
// encontro.
func TestTheLastOneRemovedTakesTheRow(t *testing.T) {
	linhas := []encounterRow{{ID: "ogro", Qtd: 2}}
	linhas = lessRow(linhas, "ogro")
	if len(linhas) != 1 || linhas[0].Qtd != 1 {
		t.Fatalf("depois de um passo: %+v", linhas)
	}
	linhas = lessRow(linhas, "ogro")
	if len(linhas) != 0 {
		t.Errorf("a linha sobreviveu ao último passo: %+v", linhas)
	}
}

// TestAnEntryThatVanishedDoesNotBecomeAnEmptyRow. Um id velho colado numa URL
// renderizaria uma linha sem nome com quantidade viva.
func TestAnEntryThatVanishedDoesNotBecomeAnEmptyRow(t *testing.T) {
	v := encontroDe(t, []encounterRow{
		{ID: "ogro", Qtd: 1},
		{ID: "dragao-de-papel-machê", Qtd: 3},
	})
	if len(v.Linhas) != 1 {
		t.Fatalf("%d linhas, quero só a que existe", len(v.Linhas))
	}
	if v.Linhas[0].Verbete.Name == "" {
		t.Error("linha sem nome sobreviveu")
	}
}

// TestTheCopiedLinkReopensTheEncounter é o ciclo inteiro: montar, copiar, colar.
//
// Ele existe porque o formato do link é escrito num lugar e lido em outro, e é
// exatamente aí que um `:` vira `-` e o encontro chega vazio do outro lado sem
// nenhum erro.
func TestTheCopiedLinkReopensTheEncounter(t *testing.T) {
	original := loadEncounters(5, 3, []encounterRow{
		{ID: "ogro", Qtd: 2}, {ID: "goblin-salteador", Qtd: 4},
	}, "")

	link := encounterAddress(original)
	_, query, achou := strings.Cut(link, "?")
	if !achou {
		t.Fatalf("o link não tem query: %q", link)
	}

	// A volta: o que o `?c=` carrega tem de reconstruir a mesma composição.
	var c string
	for _, par := range strings.Split(query, "&") {
		if chave, valor, _ := strings.Cut(par, "="); chave == "c" {
			c = strings.ReplaceAll(valor, "%3A", ":")
			c = strings.ReplaceAll(c, "%2C", ",")
		}
	}
	devolta := loadEncounters(5, 3, rowsFromURL(c), "")

	if len(devolta.Linhas) != len(original.Linhas) {
		t.Fatalf("voltaram %d linhas de %d", len(devolta.Linhas), len(original.Linhas))
	}
	if devolta.ND() != original.ND() {
		t.Errorf("o ND mudou na volta: %v virou %v", original.ND(), devolta.ND())
	}
	for i := range original.Linhas {
		if devolta.Linhas[i].Verbete.ID != original.Linhas[i].Verbete.ID ||
			devolta.Linhas[i].Qtd != original.Linhas[i].Qtd {
			t.Errorf("linha %d voltou diferente: %+v", i, devolta.Linhas[i])
		}
	}
}

// TestACrookedLinkDoesNotCostTheWholeEncounter: ele chega por chat, e um caractere a
// mais não pode zerar o que veio junto.
func TestACrookedLinkDoesNotCostTheWholeEncounter(t *testing.T) {
	linhas := rowsFromURL("ogro:2,lixo,goblin-salteador:x,,cascavel:3")
	if len(linhas) != 2 {
		t.Fatalf("%d linhas de duas boas: %+v", len(linhas), linhas)
	}
	if linhas[0].ID != "ogro" || linhas[1].ID != "cascavel" {
		t.Errorf("as linhas boas não sobreviveram: %+v", linhas)
	}
}

// TestAnAbsurdLevelAndSizeFallBackToTheDefault. Os dois vêm da URL, que qualquer um
// edita à mão, e um nível 999 mudaria a dificuldade sem mudar o encontro.
func TestAnAbsurdLevelAndSizeFallBackToTheDefault(t *testing.T) {
	v := loadEncounters(999, -3, nil, "")
	if v.Nivel != nivelPadrao || v.Grupo != grupoPadrao {
		t.Errorf("nível %d e grupo %d, quero os padrões %d e %d",
			v.Nivel, v.Grupo, nivelPadrao, grupoPadrao)
	}
}

// TestTheSearchPanelOnlyShowsWithATerm: mostrar as 80 criaturas abaixo da
// composição empurraria o VEREDITO para fora da tela, e o veredito é o assunto
// da ferramenta (ALE-170).
func TestTheSearchPanelOnlyShowsWithATerm(t *testing.T) {
	if got := loadEncounters(1, 4, nil, "").Achados; len(got) != 0 {
		t.Errorf("sem termo vieram %d criaturas", len(got))
	}
	if got := loadEncounters(1, 4, nil, "ogro").Achados; len(got) == 0 {
		t.Error("com termo não veio criatura nenhuma")
	}
}

// ── pelo fio ─────────────────────────────────────────────────────────────────

// TestTheEncounterInTheUrlHoldsOnAColdLoad — é o link colado no chat abrindo montado.
func TestTheEncounterInTheUrlHoldsOnAColdLoad(t *testing.T) {
	rec := pedeNaCena(t, "/mestre/encontros?nivel=3&grupo=4&c=ogro:2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	corpo := rec.Body.String()
	if !strings.Contains(corpo, "Ogro") {
		t.Error("o encontro do link não foi desenhado")
	}
	// A dificuldade vai ESCRITA À MÃO e não colhida de `loadEncounters`.
	//
	// A versão anterior chamava a mesma função que a página chama e afirmava
	// que a página continha o resultado dela — o que passa verde mesmo se a
	// conta estiver errada, porque os dois lados erram junto. É o que o
	// CLAUDE.md chama de derivar o esperado do código sob teste, e a fronteira
	// desta fatia o expôs ao tirar a função do alcance do `api` (ALE-278).
	// A dificuldade vai CALCULADA À MÃO pelo livro, e não colhida de
	// `loadEncounters`:
	//
	//	ogro ND 4, dois deles → 4 + 2·log2(2) = ND 6 (p282)
	//	6 − nível 3 = diferença 3 → acima da faixa "Difícil" → Mortal (p281)
	//
	// A versão anterior chamava a mesma função que a página chama e afirmava
	// que a página continha o resultado dela — o que fica verde mesmo com a
	// conta errada, porque os dois lados erram junto. É o que o CLAUDE.md chama
	// de derivar o esperado do código sob teste, e quem o expôs foi a fronteira
	// desta fatia, ao tirar a função do alcance do `api` (ALE-278).
	const dificuldadeEsperada = "Mortal"
	if !strings.Contains(corpo, dificuldadeEsperada) {
		t.Errorf("dois ogros contra um grupo de nível 3 são %q pelo livro, e a página não diz",
			dificuldadeEsperada)
	}
}
