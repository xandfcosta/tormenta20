package master

import (
	"t20engine/domain/book"
	"t20engine/serve/web/bookui"
	"testing"
)

// O guarda dos FILTROS de cada catálogo.
//
// O que se protege é a regra de COMBINAÇÃO — OU dentro de um filtro, E entre
// eles — e o fato de que cada catálogo oferece os seus. Um filtro que some da
// cena não estoura nada: a lista continua desenhando, só que inteira.

// AMOSTRAGEM sobre as abas, para o catálogo que entrar amanhã nascer medido.
func TestEachCatalogOffersItsOwnFilters(t *testing.T) {
	want := map[string][]string{
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
		keys := want[aba.ID]
		filters := filtersForTab(aba.ID)
		if len(filters) != len(keys) {
			t.Errorf("a aba %q tem %d filtros, esperado %d", aba.ID, len(filters), len(keys))
			continue
		}
		for i, f := range filters {
			if f.Key != keys[i] {
				t.Errorf("a aba %q: filtro %d é %q, esperado %q", aba.ID, i, f.Key, keys[i])
			}
			// Filtro sem opção é uma linha de rótulo e mais nada.
			if len(f.Options) == 0 {
				t.Errorf("o filtro %q da aba %q nasceu sem opções", f.Key, aba.ID)
			}
		}
	}
}

// A regra de combinação: soma dentro do filtro, multiplica entre filtros.
//
// Medido na tela: 198 magias, 39 no 3º círculo, 6 no 3º círculo E da escola de
// evocação. Os números ficam presos porque são o que separa "filtrou" de
// "filtrou do jeito certo" — um E virando OU daria 39 + as evocações todas, que
// também é uma lista plausível.
func TestTheFilterAddsWithinAndMultipliesAcross(t *testing.T) {
	all := len(book.Catalogs().Spells)
	if all != 198 {
		t.Fatalf("%d magias no catálogo — os números abaixo perderam o sentido", all)
	}

	third := quantasMagias(map[string][]string{"circulo": {"3"}})
	if third != 39 {
		t.Errorf("%d magias de 3º círculo, esperado 39", third)
	}
	// OU dentro do mesmo filtro: 3º ou 4º é mais que só 3º.
	twoCircles := quantasMagias(map[string][]string{"circulo": {"3", "4"}})
	if twoCircles <= third {
		t.Errorf("3º OU 4º deu %d, e só 3º dá %d — o OU dentro do filtro sumiu", twoCircles, third)
	}
	// E entre filtros: 3º E evocação é menos que só 3º.
	withSchool := quantasMagias(map[string][]string{"circulo": {"3"}, "escola": {"evocacao"}})
	if withSchool != 6 {
		t.Errorf("3º círculo E evocação deu %d, esperado 6", withSchool)
	}
}

func quantasMagias(filters map[string][]string) int {
	v := loadCollection(collectionCriteria{Aba: "magias", Filters: filters}, bookui.BookAddress{})
	return v.Findings
}

// PROVADO VERMELHO: procurar `exotica` onde o dado guarda `extra` faz o `else`
// devolver "Comum" para as DEZESSETE raças — inclusive as nove exóticas. Na tela
// é um rótulo plausível em todo cartão.
func TestExoticRacesSayTheyAreExotic(t *testing.T) {
	races, _, _ := book.CharacterCatalogs()
	count := map[string]int{}
	for _, r := range races {
		count[book.TierName(r.Tier)]++
	}
	if count["Exótica"] != 9 || count["Comum"] != 8 {
		t.Errorf("%d exóticas e %d comuns — o livro tem 9 e 8", count["Exótica"], count["Comum"])
	}
}

// A escola de magia tem verbete e vira ELO.
//
// Sem isso, a escola decide o filtro e não está escrita em cartão nenhum — o
// mestre filtra por evocação e as magias não dizem que são de evocação. E duas
// listas das mesmas oito escolas (uma tabela de rótulos e o dado) divergem.
func TestTheSpellSchoolHasAnEntryAndBecomesALink(t *testing.T) {
	schools := book.SpellSchools()
	if len(schools) != 8 {
		t.Fatalf("%d escolas de magia — o livro tem 8", len(schools))
	}
	// TODA escola que alguma magia usa tem verbete: elo que aponta para o vazio
	// é pior que texto puro.
	known := map[string]bool{}
	for _, e := range schools {
		known[e.ID] = true
		if e.Description == "" || e.BookPage == 0 {
			t.Errorf("a escola %q veio sem definição ou sem página", e.Name)
		}
	}
	for _, m := range book.Catalogs().Spells {
		if m.School != "" && !known[m.School] {
			t.Errorf("a magia %q é da escola %q, que não tem verbete", m.Name, m.School)
		}
	}
}

// A perícia carrega o que o livro imprime ao lado do nome.
//
// O CONTROLE que vale mais que os números: o atributo do catálogo novo é
// comparado com o do `options.json`, que é o que o motor usa para ROLAR. As duas
// fontes concordarem é o que diz que a leitura da tabela do livro está certa.
func TestTheExpertiseCarriesWhatTheBookPrintsBesideTheName(t *testing.T) {
	expertises := book.Expertises()
	if len(expertises) != 29 {
		t.Fatalf("%d perícias — o livro tem 29", len(expertises))
	}

	trained, withArmor := 0, 0
	for _, p := range expertises {
		if p.TrainedOnly {
			trained++
		}
		if p.ArmorPenalty {
			withArmor++
		}
		if p.BookPage == 0 {
			t.Errorf("a perícia %q ficou sem página", p.Name)
		}
		if book.AttributeAbbrev(p.Attribute) == p.Attribute {
			t.Errorf("a perícia %q tem atributo desconhecido: %q", p.Name, p.Attribute)
		}
	}
	if trained != 11 || withArmor != 3 {
		t.Errorf("%d só treinadas e %d com penalidade — a Tabela 2-1 dá 11 e 3", trained, withArmor)
	}

	// A página é a da DESCRIÇÃO e não a da tabela: Pontaria e Reflexos não estão
	// no índice remissivo, e a busca por título as achava na p115, que é a lista
	// de nomes. É a mesma armadilha da página do índice, uma página adiante.
	for _, p := range expertises {
		if (p.Name == "Pontaria" || p.Name == "Reflexos") && p.BookPage == 115 {
			t.Errorf("a perícia %q aponta para a tabela e não para a descrição dela", p.Name)
		}
	}
}
