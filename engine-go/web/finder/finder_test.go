package finder

import (
	"net/url"
	"strings"
	"t20engine/search"
	"t20engine/web/routes"
	"testing"
)

// OS GUARDAS DA REGRA DE BUSCA, no pacote onde a regra mora (ALE-278).
//
// Eles vieram do antigo `api/piloto_buscador`, que misturava duas camadas: sete
// casos que exercitam o casamento e o RANQUEAMENTO — funções puras — e dois que
// precisam do servidor de verdade para provar que a rota lê o sinal e que a
// porta não desenha o buscador. Aqueles dois ficaram no `api`.
//
// A divisão é a regra da casa: unitário para o que carrega REGRA, integração
// para composição. Aqui a regra é qual achado vem primeiro, e ela não precisa de
// HTTP para ser provada.

// TestTheRightEntryComesFirst: a escada de pontuação, do nome inteiro ao typo.
func TestTheRightEntryComesFirst(t *testing.T) {
	escada := []struct {
		nome, busca string
		esperado    int
	}{
		{"Abalado", "abalado", 100},
		{"Abalado", "abal", 80},
		{"Bola de Fogo", "fogo", 60},
		{"Bola de Fogo", "ogo", 40},
		{"Necromante", "ncromante", 20},
		{"Naja", "abal", 0},
	}
	for _, c := range escada {
		if ponto := search.Score(c.nome, c.busca); ponto != c.esperado {
			t.Errorf("pontuaBusca(%q, %q) = %d, esperado %d", c.nome, c.busca, ponto, c.esperado)
		}
	}
}

// TestTwoTermsFindTheWholeName.
//
// PROVADO VERMELHO com a versão de um termo só: "bola fogo" devolvia ZERO —
// o nome não começa com a frase, não a contém, e pular o "de " estoura a folga
// do quase-igual. Digitar duas palavras do que se lembra é o gesto normal.

// TestTwoTermsFindTheWholeName.
//
// PROVADO VERMELHO com a versão de um termo só: "bola fogo" devolvia ZERO —
// o nome não começa com a frase, não a contém, e pular o "de " estoura a folga
// do quase-igual. Digitar duas palavras do que se lembra é o gesto normal.
func TestTwoTermsFindTheWholeName(t *testing.T) {
	v := searchTheBook("bola fogo")
	if !hasHit(v, "Bola de Fogo") {
		t.Errorf("“bola fogo” não achou “Bola de Fogo” — %d achados", v.Achados)
	}
	if v.PeloTexto {
		t.Error("o acerto foi anunciado como menção no texto, e é casamento de NOME")
	}
}

// TestTheRuleBodyOnlyEntersWhenNoNameMatches.
//
// PROVADO VERMELHO com o corpo valendo sempre: "abal" devolvia 282 entradas —
// 139 poderes cujo texto diz "Abalado" — e a condição "Abalado", que era o que
// se procurava, saía num grupo de seis ao lado de "Naja" e "Jiboia".
//
// O controle é o segundo caso: quem NÃO sabe o nome ("chance de falha") continua
// achando, e a tela diz que aquilo é menção e não nome.

// TestTheRuleBodyOnlyEntersWhenNoNameMatches.
//
// PROVADO VERMELHO com o corpo valendo sempre: "abal" devolvia 282 entradas —
// 139 poderes cujo texto diz "Abalado" — e a condição "Abalado", que era o que
// se procurava, saía num grupo de seis ao lado de "Naja" e "Jiboia".
//
// O controle é o segundo caso: quem NÃO sabe o nome ("chance de falha") continua
// achando, e a tela diz que aquilo é menção e não nome.
func TestTheRuleBodyOnlyEntersWhenNoNameMatches(t *testing.T) {
	porNome := searchTheBook("abal")
	if porNome.PeloTexto {
		t.Fatal("houve casamento de nome e a busca caiu na segunda passada mesmo assim")
	}
	if porNome.Grupos[0].Achados[0].Nome != "Abalado" {
		t.Errorf("o primeiro achado de “abal” é %q", porNome.Grupos[0].Achados[0].Nome)
	}
	if porNome.Achados > 20 {
		t.Errorf("“abal” trouxe %d achados: o corpo das regras vazou para a primeira passada", porNome.Achados)
	}

	porTexto := searchTheBook("chance de falha")
	if porTexto.Achados == 0 {
		t.Fatal("quem não sabe o nome ficou sem nada — a segunda passada não roda")
	}
	if !porTexto.PeloTexto {
		t.Error("a tela não vai avisar que estes achados são menção, e a lista vai parecer errada")
	}
}

// TestTheCutoffSaysHowMuchIsLeftAndOffersAWayOut: corte silencioso ensina que não existe.

// TestTheCutoffSaysHowMuchIsLeftAndOffersAWayOut: corte silencioso ensina que não existe.
func TestTheCutoffSaysHowMuchIsLeftAndOffersAWayOut(t *testing.T) {
	v := searchTheBook("arma")
	poderes := groupNamed(v, "Poderes")
	if poderes == nil {
		t.Fatal("“arma” não achou poder nenhum — o guarda mediria outra coisa")
	}
	if len(poderes.Achados) != hitsByGroup {
		t.Errorf("o grupo veio com %d linhas, e o corte é %d", len(poderes.Achados), hitsByGroup)
	}
	if poderes.Cortados() <= 0 {
		t.Fatalf("“arma” achou %d poderes: escolha outro termo para medir o corte", poderes.Total)
	}
	if !strings.Contains(poderes.Mais, url.QueryEscape("arma")) {
		t.Errorf("o “+%d” leva para %q, que não é a cena com a mesma busca", poderes.Cortados(), poderes.Mais)
	}
}

// TestAHitKnowsWhereToLead: cada linha é um endereço, e eles diferem por
// ferramenta — criatura vai ao bestiário, o resto ao acervo.

// TestAHitKnowsWhereToLead: cada linha é um endereço, e eles diferem por
// ferramenta — criatura vai ao bestiário, o resto ao acervo.
func TestAHitKnowsWhereToLead(t *testing.T) {
	criatura := firstOfGroup(t, searchTheBook("lobo"), "Criaturas")
	if !strings.HasPrefix(criatura.Destino, routes.MasterBestiary+"?criatura=") {
		t.Errorf("a criatura leva para %q", criatura.Destino)
	}
	if criatura.Pagina == 0 {
		t.Error("a criatura veio sem página do livro, e o bestiário é o único catálogo que sabe a dele")
	}
	condicao := firstOfGroup(t, searchTheBook("abalado"), "Condições")
	if !strings.Contains(condicao.Destino, "/mestre/condicoes") {
		t.Errorf("a condição leva para %q", condicao.Destino)
	}
}

// TestTheFinderRouteReadsTheSignal: o caminho que o navegador usa de verdade.
//
// Pelo SINAL e não por `?busca=`: é assim que o `@get` do Datastar manda o que
// foi digitado, e a URL é só o caminho de quem abre o endereço à mão. Um guarda
// que só medisse a URL passaria verde com o sinal quebrado.

func groupNamed(v finderView, rotulo string) *finderGroup {
	for i := range v.Grupos {
		if v.Grupos[i].Rotulo == rotulo {
			return &v.Grupos[i]
		}
	}
	return nil
}

func firstOfGroup(t *testing.T, v finderView, rotulo string) finderHit {
	t.Helper()
	g := groupNamed(v, rotulo)
	if g == nil || len(g.Achados) == 0 {
		t.Fatalf("nenhum achado em %q para %q", rotulo, v.Busca)
	}
	return g.Achados[0]
}

func hasHit(v finderView, nome string) bool {
	for _, g := range v.Grupos {
		for _, a := range g.Achados {
			if a.Nome == nome {
				return true
			}
		}
	}
	return false
}

// TestTheBestHitComesInTheFirstGroup (ALE-264).
//
// PROVADO VERMELHO contra a ordem fixa: os grupos saíam na ordem da FILEIRA DE
// ABAS, e o dono viu o efeito — digitando "medo", o verbete "Medo" (nome
// inteiro, nota máxima) aparecia no sexto grupo, abaixo de criaturas que só têm
// a palavra no nome. A ordem da fileira é a certa para NAVEGAR e a errada para
// BUSCAR.

// TestTheBestHitComesInTheFirstGroup (ALE-264).
//
// PROVADO VERMELHO contra a ordem fixa: os grupos saíam na ordem da FILEIRA DE
// ABAS, e o dono viu o efeito — digitando "medo", o verbete "Medo" (nome
// inteiro, nota máxima) aparecia no sexto grupo, abaixo de criaturas que só têm
// a palavra no nome. A ordem da fileira é a certa para NAVEGAR e a errada para
// BUSCAR.
func TestTheBestHitComesInTheFirstGroup(t *testing.T) {
	casos := []struct{ termo, grupo, achado string }{
		{"medo", "Efeitos", "Medo"},
		{"abal", "Condições", "Abalado"},
		{"lobo", "Criaturas", "Lobo"},
	}
	for _, caso := range casos {
		v := searchTheBook(caso.termo)
		if len(v.Grupos) == 0 {
			t.Errorf("%q não achou nada", caso.termo)
			continue
		}
		primeiro := v.Grupos[0]
		if primeiro.Rotulo != caso.grupo || primeiro.Achados[0].Nome != caso.achado {
			t.Errorf("%q → %s(%s); esperado %s(%s)",
				caso.termo, primeiro.Rotulo, primeiro.Achados[0].Nome, caso.grupo, caso.achado)
		}
	}
}

// TestATieKeepsTheOrderOfTheRow: a ordenação é ESTÁVEL.
//
// A ordem das abas tem razão registrada — condição primeiro porque é a consulta
// do combate — e ela continua valendo quando dois grupos têm achados igualmente
// bons. Sem estabilidade, a mesma busca poderia sair em ordens diferentes.

// TestATieKeepsTheOrderOfTheRow: a ordenação é ESTÁVEL.
//
// A ordem das abas tem razão registrada — condição primeiro porque é a consulta
// do combate — e ela continua valendo quando dois grupos têm achados igualmente
// bons. Sem estabilidade, a mesma busca poderia sair em ordens diferentes.
func TestATieKeepsTheOrderOfTheRow(t *testing.T) {
	grupos := []finderGroup{
		{Rotulo: "Condições", Achados: []finderHit{{Nome: "a", ponto: 40}}},
		{Rotulo: "Magias", Achados: []finderHit{{Nome: "b", ponto: 40}}},
		{Rotulo: "Itens", Achados: []finderHit{{Nome: "c", ponto: 100}}},
	}
	sortByRelevance(grupos)
	if grupos[0].Rotulo != "Itens" {
		t.Errorf("o grupo com o melhor achado não veio na frente: %q", grupos[0].Rotulo)
	}
	if grupos[1].Rotulo != "Condições" || grupos[2].Rotulo != "Magias" {
		t.Errorf("o empate não manteve a ordem da fileira: %q, %q", grupos[1].Rotulo, grupos[2].Rotulo)
	}
}
