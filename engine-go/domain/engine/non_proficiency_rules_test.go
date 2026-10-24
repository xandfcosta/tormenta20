package engine

import (
	"strings"
	"testing"
)

// QUEM NÃO SABE USAR A ARMADURA PAGA A PENALIDADE DELA EM MAIS PERÍCIAS — e só
// isso (p152, ALE-444).
//
//	"Um personagem vestindo uma armadura ou empunhando escudo que não saiba
//	 usar aplica a penalidade da armadura em todas as perícias baseadas em
//	 Força e Destreza."
//
// É a frase inteira. Ela não fala em Defesa: o bloqueio da Destreza está na
// MESMA página, no verbete da armadura PESADA, e vale por ser pesada e não por
// faltar proficiência.
//
// # Os dois conjuntos são do livro, em COLUNAS SEPARADAS
//
// A Tabela 2-1 (p115) tem "Atributo-chave" e "Penalidade de Armadura?" lado a
// lado: 10 perícias têm For ou Des como chave, e 3 estão marcadas na coluna da
// penalidade. A p152 TROCA o conjunto de 3 por 10 — mesmo verbo e mesmo objeto
// da p153 ("aplica a penalidade da armadura em…"), só o conjunto muda —, então
// as três que estão nas duas listas pagam UMA vez (decisão do dono, ALE-444).
//
// E "baseada em Força" é o atributo-chave da tabela, com exemplo trabalhado na
// p114: "um personagem de 3º nível com Força 4 terá um valor de +5 nas perícias
// baseadas em Força, no caso, Atletismo e Luta".
func TestLackingProficiencyWidensTheArmorPenaltyInsteadOfBlockingDexterity(t *testing.T) {
	catalogs := primeBook(t)
	ptr := func(s string) *string { return &s }

	// Couro batido (−1, VESTIDO) e escudo leve (−1, EMPUNHADO). O escudo está
	// aqui porque ele é empunhado e não vestido: uma penalidade de
	// não-proficiência presa na condição `vested` deixaria todo escudo passar
	// de graça, que era o defeito.
	comProficiencias := func(prof string) ComputedSheet {
		return BookRuleset(catalogs).ComputeSheet(Character{
			Level: 2, Dexterity: 3, Strength: 1,
			Proficiencies: prof,
			Expertises:    asDuasColunasDaTabela21,
			Items: []CharacterItem{
				{CatalogID: ptr("couro-batido"), Name: "Couro batido",
					Equipped: ptr("vested"), Improvements: "[]"},
				{CatalogID: ptr("escudo-leve"), Name: "Escudo leve",
					Equipped: ptr("wielded"), Improvements: "[]"},
			},
		}, map[string]bool{})
	}
	penalidadeDe := func(sheet ComputedSheet, nome string) int {
		for _, ex := range sheet.Expertises {
			if ex.Name == nome {
				return ex.ItemBonus
			}
		}
		t.Fatalf("a ficha não tem a perícia %q", nome)
		return 0
	}

	// O CONTROLE, e ele é o denominador desta medição: PROFICIENTE nas duas
	// peças, a p153 vale sozinha. Sem ele, um motor que não penalizasse nada
	// passaria verde nos dois casos de baixo.
	sabe := comProficiencias(`["armaduras-leves","escudos"]`)
	if got := penalidadeDe(sabe, "Acrobacia"); got != -2 {
		t.Fatalf("o controle já estava errado: Acrobacia de quem SABE usar deu %d, "+
			"e a p153 cobra as duas peças (−1 do couro −1 do escudo)", got)
	}
	if got := penalidadeDe(sabe, "Luta"); got != 0 {
		t.Fatalf("o controle já estava errado: Luta de quem SABE usar deu %d, e a "+
			"coluna da Tabela 2-1 não a marca", got)
	}

	naoSabe := comProficiencias(`[]`)

	// A DESTREZA FICA. O que a cala é a armadura PESADA, e nenhuma destas é.
	if naoSabe.Defense.Attribute != "dexterity" {
		t.Errorf("a Defesa de quem não sabe usar aplicou %q: a p152 reserva o "+
			"bloqueio da Destreza à armadura PESADA, e couro batido é leve",
			naoSabe.Defense.Attribute)
	}
	if naoSabe.Defense.Total != 17 {
		t.Errorf("a Defesa deu %d, e é 10 + DES 3 + 3 do couro batido + 1 do "+
			"escudo leve", naoSabe.Defense.Total)
	}

	// AS 10 DA COLUNA "ATRIBUTO-CHAVE" passam a pagar, uma vez cada.
	if got := penalidadeDe(naoSabe, "Luta"); got != -2 {
		t.Errorf("Luta deu %d, e sem proficiência ela paga as duas peças (−2): a "+
			"p152 troca o conjunto de 3 por 10", got)
	}
	// E A LINHA DIZ POR QUE ESTÁ ALI, pela mesma razão que a da Defesa nomeia a
	// peça: a Tabela 2-1 não marca Luta, e uma penalidade sem causa na perícia
	// que o livro não penaliza se lê como defeito.
	if !temNota(naoSabe, "Luta", "sem proficiência") {
		t.Errorf("a penalidade em Luta não diz de onde veio: %+v",
			linhasDe(naoSabe, "Luta"))
	}
	if temNota(naoSabe, "Acrobacia", "sem proficiência") {
		t.Errorf("Acrobacia culpou a proficiência, e ela paga a penalidade de "+
			"qualquer jeito (p153): %+v", linhasDe(naoSabe, "Acrobacia"))
	}
	if got := penalidadeDe(naoSabe, "Atletismo"); got != -2 {
		t.Errorf("Atletismo deu %d, e sem proficiência ela paga −2", got)
	}

	// E AS TRÊS QUE ESTÃO NAS DUAS LISTAS pagam UMA vez, não duas.
	if got := penalidadeDe(naoSabe, "Acrobacia"); got != -2 {
		t.Errorf("Acrobacia deu %d: a p152 TROCA o conjunto penalizado, e não "+
			"acrescenta uma segunda aplicação sobre as três da p153 (seriam −4)", got)
	}

	// AS OUTRAS 19 NÃO PAGAM NADA. Era aqui que o motor cobrava de todo mundo.
	if got := penalidadeDe(naoSabe, "Diplomacia"); got != 0 {
		t.Errorf("Diplomacia deu %d, e o Carisma não é Força nem Destreza", got)
	}
	if got := penalidadeDe(naoSabe, "Vontade"); got != 0 {
		t.Errorf("Vontade deu %d, e a Sabedoria não é Força nem Destreza", got)
	}
}

// ARMADURA SEM PENALIDADE NÃO PENALIZA NINGUÉM, SAIBA USAR OU NÃO (p154,
// ALE-444).
//
// A p152 manda aplicar "a penalidade da armadura", e a Tabela 2-5 dá 0 para a
// armadura acolchoada e para a de couro. Zero é o número do livro: o motor
// escrevia −1 à mão quando o verbete não trazia modificador nenhum, e esse −1
// não existe em página alguma.
//
// É o caso do Batedor Nv2 da semente — ladino de armadura de couro, que a ficha
// mostrava com −1 em 29 perícias e sem Destreza na Defesa.
func TestArmorWithoutPenaltyCostsNothingToWhoCannotUseIt(t *testing.T) {
	catalogs := primeBook(t)
	ptr := func(s string) *string { return &s }

	sheet := BookRuleset(catalogs).ComputeSheet(Character{
		Level: 2, Dexterity: 3, Proficiencies: `[]`,
		Expertises: asDuasColunasDaTabela21,
		Items: []CharacterItem{{
			CatalogID: ptr("armadura-couro"), Name: "Armadura de couro",
			Equipped: ptr("vested"), Improvements: "[]",
		}},
	}, map[string]bool{})

	if sheet.Defense.Attribute != "dexterity" || sheet.Defense.Total != 15 {
		t.Errorf("a Defesa deu %d com o atributo %q, e é 10 + DES 3 + 2 da armadura "+
			"de couro — que é LEVE", sheet.Defense.Total, sheet.Defense.Attribute)
	}
	for _, ex := range sheet.Expertises {
		if ex.ItemBonus != 0 {
			t.Errorf("%s levou %d de uma armadura cuja penalidade é 0 no livro (p154)",
				ex.Name, ex.ItemBonus)
		}
	}
}

// linhasDe são as contribuições de item de uma perícia da ficha.
func linhasDe(sheet ComputedSheet, nome string) []BreakdownContribution {
	for _, ex := range sheet.Expertises {
		if ex.Name == nome {
			return ex.ItemContributions
		}
	}
	return nil
}

// temNota diz se alguma linha da perícia carrega o trecho de nota.
func temNota(sheet ComputedSheet, nome, trecho string) bool {
	for _, linha := range linhasDe(sheet, nome) {
		if strings.Contains(linha.Note, trecho) {
			return true
		}
	}
	return false
}

// asDuasColunasDaTabela21 é uma amostra das perícias da p115 escolhida para
// cobrir os dois conjuntos e a interseção deles: as três marcadas na coluna
// "Penalidade de Armadura?", as sete que só têm For ou Des como atributo-chave,
// e duas de fora para provar que o resto não paga nada.
var asDuasColunasDaTabela21 = []CharacterExpertise{
	// As TRÊS que estão nas duas listas.
	{Name: "Acrobacia", Attribute: "dexterity"},
	{Name: "Furtividade", Attribute: "dexterity"},
	{Name: "Ladinagem", Attribute: "dexterity"},
	// As SETE que só a coluna do atributo-chave alcança.
	{Name: "Atletismo", Attribute: "strength"},
	{Name: "Cavalgar", Attribute: "dexterity"},
	{Name: "Iniciativa", Attribute: "dexterity"},
	{Name: "Luta", Attribute: "strength"},
	{Name: "Pilotagem", Attribute: "dexterity"},
	{Name: "Pontaria", Attribute: "dexterity"},
	{Name: "Reflexos", Attribute: "dexterity"},
	// E duas das 19 de fora, que nenhuma das duas listas alcança.
	{Name: "Diplomacia", Attribute: "charisma"},
	{Name: "Vontade", Attribute: "wisdom"},
}
