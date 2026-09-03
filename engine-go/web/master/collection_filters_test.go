package master

import (
	"t20engine/book"
	"t20engine/web/bookui"
	"testing"
)

// O guarda dos FILTROS de cada catálogo (ALE-264).
//
// O que se protege é a regra de COMBINAÇÃO — OU dentro de um filtro, E entre
// eles — e o fato de que cada catálogo oferece os seus. Um filtro que some da
// cena não estoura nada: a lista continua desenhando, só que inteira.

// TestEachCatalogOffersItsOwnFilters: AMOSTRAGEM sobre as abas, para o
// catálogo que entrar amanhã nascer medido.
func TestEachCatalogOffersItsOwnFilters(t *testing.T) {
	esperado := map[string][]string{
		"condicoes": {"efeito"},
		"magias":    {"circulo", "escola", "classe"},
		"pericias":  {"atributo", "treino"},
		"poderes":   {"fonte"},
		"itens":     {"familia"},
		"deuses":    {"energia"},
		"racas":     {"linhagem"},
		// Classes (14), Efeitos (18) e Escolas (8) não têm filtro, e é decisão:
		// numa lista que cabe na tela, filtro é cromo que não poupa rolagem.
		"classes": nil,
		"efeitos": nil,
		"escolas": nil,
	}
	for _, aba := range collectionTabs {
		chaves := esperado[aba.ID]
		filtros := filtersForTab(aba.ID)
		if len(filtros) != len(chaves) {
			t.Errorf("a aba %q tem %d filtros, esperado %d", aba.ID, len(filtros), len(chaves))
			continue
		}
		for i, f := range filtros {
			if f.Chave != chaves[i] {
				t.Errorf("a aba %q: filtro %d é %q, esperado %q", aba.ID, i, f.Chave, chaves[i])
			}
			// Filtro sem opção é uma linha de rótulo e mais nada.
			if len(f.Opcoes) == 0 {
				t.Errorf("o filtro %q da aba %q nasceu sem opções", f.Chave, aba.ID)
			}
		}
	}
}

// TestTheFilterAddsWithinAndMultipliesAcross: a regra de combinação.
//
// Medido na tela: 198 magias, 39 no 3º círculo, 6 no 3º círculo E da escola de
// evocação. Os números ficam presos porque são o que separa "filtrou" de
// "filtrou do jeito certo" — um E virando OU daria 39 + as evocações todas, que
// também é uma lista plausível.
func TestTheFilterAddsWithinAndMultipliesAcross(t *testing.T) {
	todas := len(book.Catalogs().Magias)
	if todas != 198 {
		t.Fatalf("%d magias no catálogo — os números abaixo perderam o sentido", todas)
	}

	terceiro := quantasMagias(map[string][]string{"circulo": {"3"}})
	if terceiro != 39 {
		t.Errorf("%d magias de 3º círculo, esperado 39", terceiro)
	}
	// OU dentro do mesmo filtro: 3º ou 4º é mais que só 3º.
	doisCirculos := quantasMagias(map[string][]string{"circulo": {"3", "4"}})
	if doisCirculos <= terceiro {
		t.Errorf("3º OU 4º deu %d, e só 3º dá %d — o OU dentro do filtro sumiu", doisCirculos, terceiro)
	}
	// E entre filtros: 3º E evocação é menos que só 3º.
	comEscola := quantasMagias(map[string][]string{"circulo": {"3"}, "escola": {"evocacao"}})
	if comEscola != 6 {
		t.Errorf("3º círculo E evocação deu %d, esperado 6", comEscola)
	}
}

func quantasMagias(filtros map[string][]string) int {
	v := loadCollection(collectionCriteria{Aba: "magias", Filtros: filtros}, bookui.BookAddress{})
	return v.Achados
}

// TestExoticRacesSayTheyAreExotic.
//
// PROVADO VERMELHO: o código procurava `exotica` e o dado guarda `extra`, então
// o `else` devolvia "Comum" para as DEZESSETE — inclusive as nove exóticas. Só
// apareceu ao medir o dado para montar o filtro; na tela era um rótulo plausível
// em todo cartão.
func TestExoticRacesSayTheyAreExotic(t *testing.T) {
	racas, _, _ := book.CharacterCatalogs()
	contagem := map[string]int{}
	for _, r := range racas {
		contagem[book.TierName(r.Tier)]++
	}
	if contagem["Exótica"] != 9 || contagem["Comum"] != 8 {
		t.Errorf("%d exóticas e %d comuns — o livro tem 9 e 8", contagem["Exótica"], contagem["Comum"])
	}
}

// TestTheSpellSchoolHasAnEntryAndBecomesALink (ALE-264).
//
// O filtro de escola nasceu antes do catálogo, e por duas horas o app teve DUAS
// listas das mesmas oito escolas: uma tabela de rótulos no código e o dado das
// magias. Pior: a escola decidia o filtro e não estava escrita em cartão nenhum
// — o mestre filtrava por evocação e as magias não diziam que eram de evocação.
func TestTheSpellSchoolHasAnEntryAndBecomesALink(t *testing.T) {
	escolas := book.SpellSchools()
	if len(escolas) != 8 {
		t.Fatalf("%d escolas de magia — o livro tem 8", len(escolas))
	}
	// TODA escola que alguma magia usa tem verbete: elo que aponta para o vazio
	// é pior que texto puro.
	conhecidas := map[string]bool{}
	for _, e := range escolas {
		conhecidas[e.ID] = true
		if e.Description == "" || e.BookPage == 0 {
			t.Errorf("a escola %q veio sem definição ou sem página", e.Name)
		}
	}
	for _, m := range book.Catalogs().Magias {
		if m.School != "" && !conhecidas[m.School] {
			t.Errorf("a magia %q é da escola %q, que não tem verbete", m.Name, m.School)
		}
	}
}

// TestTheExpertiseCarriesWhatTheBookPrintsBesideTheName (ALE-264).
//
// As perícias existiam como lista de NOME e ATRIBUTO dentro do `options.json` —
// sem página, sem as duas regras da Tabela 2-1 e sem lugar para um elo apontar.
//
// O CONTROLE que vale mais que os números: o atributo do catálogo novo é
// comparado com o do `options.json`, que é o que o motor usa para ROLAR. As duas
// fontes concordarem é o que diz que a leitura da tabela do livro está certa.
func TestTheExpertiseCarriesWhatTheBookPrintsBesideTheName(t *testing.T) {
	pericias := book.Expertises()
	if len(pericias) != 29 {
		t.Fatalf("%d perícias — o livro tem 29", len(pericias))
	}

	treinadas, comArmadura := 0, 0
	for _, p := range pericias {
		if p.SoTreinada {
			treinadas++
		}
		if p.PenalidadeDeArmadura {
			comArmadura++
		}
		if p.BookPage == 0 {
			t.Errorf("a perícia %q ficou sem página", p.Name)
		}
		if book.AttributeAbbrev(p.Attribute) == p.Attribute {
			t.Errorf("a perícia %q tem atributo desconhecido: %q", p.Name, p.Attribute)
		}
	}
	if treinadas != 11 || comArmadura != 3 {
		t.Errorf("%d só treinadas e %d com penalidade — a Tabela 2-1 dá 11 e 3", treinadas, comArmadura)
	}

	// A página é a da DESCRIÇÃO e não a da tabela: Pontaria e Reflexos não estão
	// no índice remissivo, e a busca por título as achava na p115, que é a lista
	// de nomes. É a mesma armadilha da página do índice, uma página adiante.
	for _, p := range pericias {
		if (p.Name == "Pontaria" || p.Name == "Reflexos") && p.BookPage == 115 {
			t.Errorf("a perícia %q aponta para a tabela e não para a descrição dela", p.Name)
		}
	}
}
