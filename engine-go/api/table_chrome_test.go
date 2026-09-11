package api

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// O CROMO DO MAPA NÃO PODE ROUBAR O CLIQUE DO QUE NÃO É BOTÃO (ALE-294), e a
// forma de não roubar tem uma armadilha que este guarda existe para prender.
//
// Os painéis que flutuam sobre a cena são CAIXAS: eles recebiam o clique na
// faixa inteira que ocupam, inclusive nos vãos entre os ícones. Medido a 390px
// na `73658909`: dos 864 pontos do painel de verbos, 328 — 38% — caíam no fundo
// dele, e ali pintar terreno, largar marcador ou pegar uma peça era um gesto que
// sumia sem nada na tela para explicar.
//
// O remédio é `pointer-events: none` no contêiner com `auto` em cada controle, e
// ele só é seguro se TODO controle reativar: um que fique de fora vira botão
// morto, que é pior que o defeito e não deixa erro em lugar nenhum. Um botão sem
// clique é indistinguível de um botão que não faz nada.
//
// Ele cruza as duas pontas, como o `TestEveryClassPositionedByColAndRowHasABox`:
// a FOLHA COMPILADA diz quem não recebe ponteiro e quem o devolve, o HTML
// SERVIDO diz quem precisa dele. Quem acrescentar um botão a um desses painéis
// amanhã cai aqui sem ler nada disto.
func TestNoChromeOverTheMapStealsTheClickOffItsControls(t *testing.T) {
	f := novoPiloto(t)
	// A cena mais CHEIA que o mestre vê, senão o guarda mede o que sobrou: com
	// acervo (o baú só nasce com lugar guardado), com segunda aba (a barra de
	// abas), com peça (a camada de mover) e com movimento proposto.
	f.seedOpenBoard(t, "stone")
	f.savePlace(t, "Taverna do Javali")
	f.seedOpenBoard(t, "stone")
	f.openSecond(t, "Cripta do Rei Caolho")
	tokenID := f.onBoardAt(t, 4, 2)
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.tableUrl()+"/tabuleiro/"+tokenID+"/parada", `{"from":{"X":7,"Y":3}}`); rec.Code != http.StatusOK {
		t.Fatalf("propor a parada deu %d", rec.Code)
	}
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	semPonteiro, devolvem := pointerRulesOfTheMap(t)
	if len(semPonteiro) < 3 {
		t.Fatalf("a folha só tira o ponteiro de %d classes do tabuleiro: o canal não está aberto e o silêncio abaixo não é evidência", len(semPonteiro))
	}
	// O PAINEL DE VERBOS é o caso que escreveu a issue, e ele é nomeado porque a
	// varredura abaixo não o cobraria: um painel que recebe o clique inteiro não
	// tem controle morto nenhum — ele está errado do outro lado.
	if !semPonteiro["board-scene-verbs"] {
		t.Error("o painel de verbos recebe o clique na faixa INTEIRA que ocupa: medido a 390px, 328 dos 864 pontos dele (38%) caem no fundo entre os ícones, e ali pintar terreno, largar marcador ou pegar uma peça é um gesto que some")
	}

	raiz, err := html.Parse(strings.NewReader(tela))
	if err != nil {
		t.Fatalf("ler a cena servida: %v", err)
	}
	paineis, controles := 0, 0
	var visita func(no *html.Node, painel string, alcancado bool)
	visita = func(no *html.Node, painel string, alcancado bool) {
		if no.Type == html.ElementNode {
			if painel == "" {
				for _, classe := range strings.Fields(attr(no, "class")) {
					if !semPonteiro[classe] {
						continue
					}
					painel, alcancado = classe, reachedByRelief(no, true, devolvem[classe])
					paineis++
					break
				}
			} else if !alcancado {
				alcancado = reachedByRelief(no, false, devolvem[painel])
			}
			if painel != "" && needsThePointer(no) {
				controles++
				if !alcancado {
					t.Errorf("<%s %q> dentro de .%s precisa do ponteiro e a folha não o devolve: ele nasce morto",
						no.Data, attr(no, "aria-label"), painel)
				}
			}
		}
		for filho := no.FirstChild; filho != nil; filho = filho.NextSibling {
			visita(filho, painel, alcancado)
		}
	}
	visita(raiz, "", false)

	// O DENOMINADOR, e ele é o que separa "nada reprovou" de "não mediu": a cena
	// montada acima tem o painel dos verbos inteiro do mestre — enquadramento,
	// cortina, lente, pôr no mapa, abrir outro, acervo e encerrar.
	if paineis < 3 || controles < 7 {
		t.Fatalf("o guarda visitou %d painéis e %d controles: a montagem não produziu o que ele vem medir", paineis, controles)
	}
}

// pointerRelief é uma regra de `pointer-events: auto` escrita para dentro de um
// nó que não recebe o ponteiro. São as TRÊS formas que a folha usa hoje, e uma
// quarta faz o guarda FALHAR em vez de passar por cima: um seletor que ele não
// sabe ler é um controle que ele não sabe conferir, e a primeira versão deste
// guarda ignorou em silêncio o `.board-notices>*` — 14 botões vivos saíram
// como mortos, com cara de defeito achado.
type pointerRelief struct {
	filhoDireto bool            // `.painel>*`
	tags        map[string]bool // `.painel :is(a, button, …)` e `button.painel`
	oProprioNo  bool            // `button.painel`: a tag é do painel, não de dentro
}

// pointerRulesOfTheMap lê a folha COMPILADA e devolve, do tabuleiro, quem não
// recebe ponteiro e o que o devolve dentro de cada um.
//
// Só entra no primeiro conjunto o seletor de UMA CLASSE SOZINHA: um
// `.board-with-tool .board-token` tira o ponteiro sob uma condição
// que não está no nó, e ler isso como "a peça está morta" reprovaria a cena
// inteira toda vez que uma ferramenta estivesse desligada.
func pointerRulesOfTheMap(t *testing.T) (map[string]bool, map[string][]pointerRelief) {
	t.Helper()
	semPonteiro := map[string]bool{}
	devolvem := map[string][]pointerRelief{}
	for _, regra := range strings.Split(compiledStylesheet(t), "}") {
		abre := strings.Index(regra, "{")
		if abre < 0 {
			continue
		}
		corpo, seletores := regra[abre:], splitSelectors(regra[:abre])
		for _, seletor := range seletores {
			seletor = strings.TrimSpace(seletor)
			switch {
			case strings.Contains(corpo, "pointer-events:none"):
				nome := strings.TrimPrefix(seletor, ".")
				if strings.HasPrefix(nome, "board-") && !strings.ContainsAny(nome, " >+~.:[") {
					semPonteiro[nome] = true
				}
			case strings.Contains(corpo, "pointer-events:auto"):
				if painel, alivio, ok := parseRelief(t, seletor); ok {
					devolvem[painel] = append(devolvem[painel], alivio)
				}
			}
		}
	}
	return semPonteiro, devolvem
}

// splitSelectors reparte a lista de seletores de uma regra pelas vírgulas que
// separam SELETORES, e não pelas que estão dentro de um `:is(a, button, …)`.
//
// Um `strings.Split` cru quebrava `.painel :is(a,button)` em `.painel :is(a` e
// `button`, e as duas metades caíam no ramo que ignora em silêncio — os dez
// botões do painel saíram como mortos, com o alívio escrito na folha.
func splitSelectors(lista string) []string {
	seletores, inicio, profundidade := []string{}, 0, 0
	for i, r := range lista {
		switch r {
		case '(':
			profundidade++
		case ')':
			profundidade--
		case ',':
			if profundidade == 0 {
				seletores = append(seletores, lista[inicio:i])
				inicio = i + 1
			}
		}
	}
	return append(seletores, lista[inicio:])
}

func parseRelief(t *testing.T, seletor string) (string, pointerRelief, bool) {
	t.Helper()
	// `button.board-marker`: a folha diz que ESTA tag do painel recebe o
	// ponteiro. O marcador é `none` para a mesa e `auto` para o mestre, e a
	// diferença entre os dois é a tag que o servidor escreve.
	if tag, nome, achou := strings.Cut(seletor, "."); achou && tag != "" && !strings.ContainsAny(nome, " >+~.:[") {
		if strings.HasPrefix(nome, "board-") {
			return nome, pointerRelief{tags: map[string]bool{tag: true}, oProprioNo: true}, true
		}
		return "", pointerRelief{}, false
	}
	if nome, achou := strings.CutSuffix(strings.TrimPrefix(seletor, "."), ">*"); achou {
		if !strings.HasPrefix(nome, "board-") {
			return "", pointerRelief{}, false
		}
		return nome, pointerRelief{filhoDireto: true}, true
	}
	nome, resto, achou := strings.Cut(strings.TrimPrefix(seletor, "."), " ")
	if !achou || !strings.HasPrefix(nome, "board-") {
		// Seletor que não fala de dentro de um painel do tabuleiro — o
		// `.board-marker-actions` é um nó solto e não um alívio.
		return "", pointerRelief{}, false
	}
	lista, ok := strings.CutPrefix(resto, ":is(")
	if !ok {
		t.Fatalf("a folha devolve o ponteiro por %q, uma forma que este guarda não sabe ler: ensine-o antes de confiar no verde", seletor)
	}
	tags := map[string]bool{}
	for _, tag := range strings.Split(strings.TrimSuffix(lista, ")"), ",") {
		tags[strings.TrimSpace(tag)] = true
	}
	return nome, pointerRelief{tags: tags}, true
}

// reachedByRelief diz se ESTE nó recebe o ponteiro de volta. A subárvore dele vem
// junto por herança, e é por isso que quem chama carrega o `alcancado` adiante.
func reachedByRelief(no *html.Node, ehOPainel bool, alivios []pointerRelief) bool {
	for _, alivio := range alivios {
		if alivio.oProprioNo != ehOPainel {
			continue
		}
		if alivio.filhoDireto || alivio.tags[no.Data] {
			return true
		}
	}
	return false
}

// needsThePointer: o nó é alvo de dedo, seja pela tag ou por um gesto de
// ponteiro escrito nele.
//
// O `__window` e o `__document` ficam de FORA, e a distinção não é sutileza: o
// `clipboardStrip` carrega um `data-on:keydown__window` e o ouvinte dele mora na
// janela — aquele `<div>` não precisa receber clique nenhum para o CTRL+V
// funcionar, e cobrá-lo mataria justamente os vãos que a issue vem devolver.
func needsThePointer(no *html.Node) bool {
	switch no.Data {
	case "a", "button", "input", "label", "select", "summary", "textarea":
		return true
	}
	for _, a := range no.Attr {
		gesto, ok := strings.CutPrefix(a.Key, "data-on:")
		if !ok || strings.Contains(gesto, "__window") || strings.Contains(gesto, "__document") {
			continue
		}
		switch strings.Split(gesto, "__")[0] {
		case "click", "contextmenu", "dblclick", "pointerdown", "pointerup", "pointermove", "wheel":
			return true
		}
	}
	return false
}

func attr(no *html.Node, nome string) string {
	for _, a := range no.Attr {
		if a.Key == nome {
			return a.Val
		}
	}
	return ""
}
