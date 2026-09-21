package finder

import (
	"net/url"
	"strings"
	"t20engine/domain/search"
	"t20engine/serve/web/routes"
	"testing"
)

// OS GUARDAS DA REGRA DE BUSCA, no pacote onde a regra mora.
//
// UNITÁRIO porque aqui a regra é qual achado vem primeiro, e ela não precisa de
// HTTP para ser provada. O que precisa do servidor — a rota lendo o sinal, a
// porta não desenhando o buscador — mora no `api`.

// A escada de pontuação, do nome inteiro ao typo.
func TestTheRightEntryComesFirst(t *testing.T) {
	stairs := []struct {
		name, search string
		want         int
	}{
		{"Abalado", "abalado", 100},
		{"Abalado", "abal", 80},
		{"Bola de Fogo", "fogo", 60},
		{"Bola de Fogo", "ogo", 40},
		{"Necromante", "ncromante", 20},
		{"Naja", "abal", 0},
	}
	for _, c := range stairs {
		if point := search.Score(c.name, c.search); point != c.want {
			t.Errorf("pontuaBusca(%q, %q) = %d, esperado %d", c.name, c.search, point, c.want)
		}
	}
}

// PROVADO VERMELHO com a versão de um termo só: "bola fogo" devolvia ZERO — o
// nome não começa com a frase, não a contém, e pular o "de " estoura a folga do
// quase-igual. Digitar duas palavras do que se lembra é o gesto normal.
func TestTwoTermsFindTheWholeName(t *testing.T) {
	v := searchTheBook("bola fogo")
	if !hasHit(v, "Bola de Fogo") {
		t.Errorf("“bola fogo” não achou “Bola de Fogo” — %d achados", v.Findings)
	}
	if v.ByText {
		t.Error("o acerto foi anunciado como menção no texto, e é casamento de NOME")
	}
}

// PROVADO VERMELHO com o corpo valendo sempre: "abal" devolvia 282 entradas —
// 139 poderes cujo texto diz "Abalado" — e a condição "Abalado", que era o que
// se procurava, saía num grupo de seis.
//
// O CONTROLE é o segundo caso: quem NÃO sabe o nome ("chance de falha") continua
// achando, e a tela diz que aquilo é menção e não nome.
func TestTheRuleBodyOnlyEntersWhenNoNameMatches(t *testing.T) {
	byName := searchTheBook("abal")
	if byName.ByText {
		t.Fatal("houve casamento de nome e a busca caiu na segunda passada mesmo assim")
	}
	if byName.Groups[0].Findings[0].Name != "Abalado" {
		t.Errorf("o primeiro achado de “abal” é %q", byName.Groups[0].Findings[0].Name)
	}
	if byName.Findings > 20 {
		t.Errorf("“abal” trouxe %d achados: o corpo das regras vazou para a primeira passada", byName.Findings)
	}

	byText := searchTheBook("chance de falha")
	if byText.Findings == 0 {
		t.Fatal("quem não sabe o nome ficou sem nada — a segunda passada não roda")
	}
	if !byText.ByText {
		t.Error("a tela não vai avisar que estes achados são menção, e a lista vai parecer errada")
	}
}

// Corte silencioso ensina que não existe.
func TestTheCutoffSaysHowMuchIsLeftAndOffersAWayOut(t *testing.T) {
	v := searchTheBook("arma")
	powers := groupNamed(v, "Poderes")
	if powers == nil {
		t.Fatal("“arma” não achou poder nenhum — o guarda mediria outra coisa")
	}
	if len(powers.Findings) != hitsByGroup {
		t.Errorf("o grupo veio com %d linhas, e o corte é %d", len(powers.Findings), hitsByGroup)
	}
	if powers.Cortados() <= 0 {
		t.Fatalf("“arma” achou %d poderes: escolha outro termo para medir o corte", powers.Total)
	}
	if !strings.Contains(powers.More, url.QueryEscape("arma")) {
		t.Errorf("o “+%d” leva para %q, que não é a cena com a mesma busca", powers.Cortados(), powers.More)
	}
}

// Cada linha é um endereço, e eles diferem por ferramenta — criatura vai ao
// bestiário, o resto ao acervo.
func TestAHitKnowsWhereToLead(t *testing.T) {
	creature := firstOfGroup(t, searchTheBook("lobo"), "Criaturas")
	if !strings.HasPrefix(creature.Destination, routes.MasterBestiary+"?criatura=") {
		t.Errorf("a criatura leva para %q", creature.Destination)
	}
	if creature.Page == 0 {
		t.Error("a criatura veio sem página do livro, e o bestiário é o único catálogo que sabe a dele")
	}
	condition := firstOfGroup(t, searchTheBook("abalado"), "Condições")
	if !strings.Contains(condition.Destination, "/mestre/condicoes") {
		t.Errorf("a condição leva para %q", condition.Destination)
	}
}

func groupNamed(v finderView, label string) *finderGroup {
	for i := range v.Groups {
		if v.Groups[i].Label == label {
			return &v.Groups[i]
		}
	}
	return nil
}

func firstOfGroup(t *testing.T, v finderView, label string) finderHit {
	t.Helper()
	g := groupNamed(v, label)
	if g == nil || len(g.Findings) == 0 {
		t.Fatalf("nenhum achado em %q para %q", label, v.Search)
	}
	return g.Findings[0]
}

func hasHit(v finderView, name string) bool {
	for _, g := range v.Groups {
		for _, a := range g.Findings {
			if a.Name == name {
				return true
			}
		}
	}
	return false
}

// PROVADO VERMELHO contra a ordem fixa: com os grupos na ordem da FILEIRA DE
// ABAS, "medo" punha o verbete "Medo" (nome inteiro, nota máxima) no sexto
// grupo, abaixo de criaturas que só têm a palavra no nome. A ordem da fileira é
// a certa para NAVEGAR e a errada para BUSCAR.
func TestTheBestHitComesInTheFirstGroup(t *testing.T) {
	cases := []struct{ term, group, found string }{
		{"medo", "Efeitos", "Medo"},
		{"abal", "Condições", "Abalado"},
		{"lobo", "Criaturas", "Lobo"},
	}
	for _, tc := range cases {
		v := searchTheBook(tc.term)
		if len(v.Groups) == 0 {
			t.Errorf("%q não achou nada", tc.term)
			continue
		}
		first := v.Groups[0]
		if first.Label != tc.group || first.Findings[0].Name != tc.found {
			t.Errorf("%q → %s(%s); esperado %s(%s)",
				tc.term, first.Label, first.Findings[0].Name, tc.group, tc.found)
		}
	}
}

// A ordenação é ESTÁVEL: a ordem das abas — condição primeiro, porque é a
// consulta do combate — continua valendo quando dois grupos têm achados
// igualmente bons. Sem estabilidade, a mesma busca sairia em ordens diferentes.
func TestATieKeepsTheOrderOfTheRow(t *testing.T) {
	groups := []finderGroup{
		{Label: "Condições", Findings: []finderHit{{Name: "a", point: 40}}},
		{Label: "Magias", Findings: []finderHit{{Name: "b", point: 40}}},
		{Label: "Itens", Findings: []finderHit{{Name: "c", point: 100}}},
	}
	sortByRelevance(groups)
	if groups[0].Label != "Itens" {
		t.Errorf("o grupo com o melhor achado não veio na frente: %q", groups[0].Label)
	}
	if groups[1].Label != "Condições" || groups[2].Label != "Magias" {
		t.Errorf("o empate não manteve a ordem da fileira: %q, %q", groups[1].Label, groups[2].Label)
	}
}
