package api

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// O guarda da FAMÍLIA de uma armadilha que não deixa erro para trás.
//
// O sinal do Datastar é um PROXY REATIVO: ler um índice que não existe o CRIA.
// Com a reserva de doze rótulos no ar, `$ruler_points[i]` encheu o sinal de
// strings vazias —
//
//	[[9,3], "", "", "", "", "", "", "", "", "", "", "", "", [17,7], …]
//
// — e o resultado foi pingos na origem do plano e o servidor medindo zero, SEM
// erro em lugar nenhum. Guardar o sinal numa constante não resolve: a constante
// continua sendo o proxy. O que resolve é COPIAR (`[...$lista]`), e é isso que
// este caso prende.
//
// Ele varre o HTML SERVIDO e não o código, que é a única forma de alcançar a
// expressão que alguém escrever amanhã sem ler nada disto.
func TestNoExpressionIndexesTheListSignal(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	screen := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CONTROLE: as expressões da régua ESTÃO na página. Sem ele, não achar
	// `$ruler_points[` seria verdade também sobre uma cena que não desenhou régua
	// nenhuma.
	if !strings.Contains(screen, "ruler_points") {
		t.Fatal("a cena não tem as expressões da régua — o guarda mediria o vazio")
	}
	// A REGRA: depois de `$lista` só pode vir `=` (uma escrita) ou `]` (o fim de
	// um `[...$lista]`, que é a CÓPIA). Qualquer outra coisa — `[`, `.`, `;` — é
	// acesso ao proxy, e o proxy cria.
	//
	// Procurar só `$lista[` passa VERDE sobre a segunda forma do mesmo defeito:
	// `const lista = $ruler_points;` põe o PROXY na constante, e `lista[12]` cria
	// o índice do mesmo jeito. Por isso a regra é sobre o que PODE vir depois, e
	// não sobre uma forma errada conhecida.
	for _, list := range []string{"ruler_points", "ruler_labels"} {
		accesses := regexp.MustCompile(`\$`+list+`\s*(.)`).FindAllStringSubmatch(screen, -1)
		if len(accesses) == 0 {
			t.Errorf("`$%s` não aparece na cena — o controle acima não alcançou esta lista", list)
		}
		for _, a := range accesses {
			if a[1] != "=" && a[1] != "]" {
				t.Errorf("a cena acessa `$%s` seguido de %q: o proxy do Datastar CRIA o que se lê "+
					"e a lista se enche de strings vazias, sem erro nenhum. Copie com `[...$%s]`.",
					list, a[1], list)
			}
		}
	}
}

// Uma afirmação sobre a FORMA do que a página serve, e é o único jeito de
// alcançar os quatro de uma vez: clique acrescenta, duplo clique congela, botão
// direito apaga, e o ponteiro leva a perna viva atrás do dedo.
//
// O `evt.button !== 0` entra na lista porque ele é conserto de um defeito
// medido: sem ele, o clique sintético que acompanha o botão direito nasce com
// `offsetX` zero e a régua renasce na ORIGEM do plano no mesmo gesto que a apagou.
func TestTheScreenWiresTheFourRulerGestures(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	screen := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	for _, chunk := range []string{
		"data-on:dblclick",
		"data-on:contextmenu",
		"data-on:pointermove",
		"evt.button !== 0",
		"tabuleiro/regua",
	} {
		if !strings.Contains(screen, chunk) {
			t.Errorf("a cena não tem %q: um dos gestos da régua não acontece", chunk)
		}
	}
}

// O teto de paradas é o tamanho da RESERVA de nós no `.templ`, e uma polilinha
// maior teria pernas medidas que ninguém desenha.
func TestAForgedRulerIsRefused(t *testing.T) {
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")

	// O TETO vai escrito à mão (12 paradas, o tamanho da reserva de nós no
	// `.templ`): lê-lo do `stopsMax` da cena faria o esperado sair do código sob
	// teste, e um teto trocado passaria verde dos dois lados.
	const stopCeiling = 12
	points := make([]string, 0, stopCeiling+2)
	for i := range stopCeiling + 2 {
		points = append(points, "["+string(rune('0'+i%10))+",0]")
	}
	body := f.posts(t, f.gm, f.tableUrl()+"/tabuleiro/regua",
		`{"ruler_points":[`+strings.Join(points, ",")+`],"ruler_phase":2}`)
	if !strings.Contains(body, "teto") {
		t.Errorf("uma régua com %d paradas não foi recusada: %q", len(points), body)
	}
}

// REGRA DO LIVRO, não escolha de tela.
//
// p225, conferido no PDF p231:
//
//	"Esfera. Surge na INTERSEÇÃO DE QUATRO QUADRADOS, estendendo-se em todas as
//	 direções até o limite de seu raio."
//	"Quadrado. Surge NO QUADRADO ou quadrados escolhidos."
//
// Não há escolha de "centralizar": o `engine.sphereSquares` desenha a partir do
// CANTO, e uma tela que mandasse o quadrado do `floor` do clique põe até meio
// quadrado entre onde o dedo estava e onde a bola cai, sem dizer por quê.
//
// O guarda prende as DUAS pontas — a regra e o gesto — porque separadas elas já
// divergiram uma vez.
func TestTheSphereIsBornAtTheIntersection(t *testing.T) {
	// A metade da REGRA — só a esfera nasce na interseção — mora em `web/table`,
	// porque é função pura e não precisa de banco. Aqui fica a metade que só um
	// servidor montado prova.
	f := newSceneFixture(t)
	f.seedOpenBoard(t, "stone")
	screen := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(screen, "Math.round((evt.offsetX") {
		t.Error("a tela não arredonda o clique para o canto: com `floor` a esfera cai " +
			"meio quadrado longe do dedo, e o defeito é silencioso")
	}
	if !strings.Contains(screen, "template_at_intersection") {
		t.Error("a tela não pergunta ao servidor onde a forma nasce — a regra virou " +
			"uma segunda cópia na expressão")
	}
}
