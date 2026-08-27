package api

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"t20engine/engine"
)

// Os guardas da RÉGUA DE VÁRIAS PARADAS (ALE-203, item 5 do dono).
//
// "A régua não permite calcular distâncias com mais de uma parada. Ao montar uma
// régua, não tem como apagá-la sem trocar de ferramenta."
//
// A aritmética de UMA perna não é medida aqui: `engine.Measure` tem guarda de
// regra próprio, escrito contra a p224. O que se prende deste lado é a
// composição — o total é a soma das pernas —, o formato do rótulo, e a família de
// armadilha que esta superfície descobriu no Datastar.

// TestOTotalEASomaDasPernas.
//
// A polilinha mede um CAMINHO, e o caminho é a soma dos trechos. Com DUAS
// paradas ela é uma reta e o total é a régua de sempre, que é o caso que não
// pode ter mudado.
func TestOTotalEASomaDasPernas(t *testing.T) {
	reta := aLeituraDaPolilinha([]engine.Square{{}, {X: 3}})
	if reta["reguatexto"] != leituraDaRegua(engine.Measure(engine.Square{}, engine.Square{X: 3})) {
		t.Errorf("a régua de duas paradas deixou de ser a régua de sempre: %q", reta["reguatexto"])
	}

	// Três paradas: 3 + 4 = 7 quadrados de caminho.
	caminho := aLeituraDaPolilinha([]engine.Square{{}, {X: 3}, {X: 3, Y: 4}})
	if !strings.HasPrefix(caminho["reguatexto"].(string), "7 quadrados") {
		t.Errorf("o total de 3+4 pernas saiu %q, esperado 7 quadrados", caminho["reguatexto"])
	}
	if rotulos := caminho["reguarotulos"].([]string); len(rotulos) != 2 {
		t.Errorf("três paradas deram %d rótulos, esperado 2 pernas: %v", len(rotulos), rotulos)
	}
}

// TestORotuloDaPernaVemEmMetros — pedido do dono, e ele é sobre a UNIDADE.
//
// Sobre a linha cabe uma unidade só, e a que a mesa fala em voz alta é o metro:
// "ele está a nove metros" é a frase do turno, e "seis quadrados" obriga a
// converter de cabeça. A frase do TOTAL continua trazendo as duas, porque lá cabe.
func TestORotuloDaPernaVemEmMetros(t *testing.T) {
	leitura := aLeituraDaPolilinha([]engine.Square{{}, {X: 6}})
	rotulos := leitura["reguarotulos"].([]string)
	if len(rotulos) != 1 || rotulos[0] != "9,0m" {
		t.Errorf("a perna de 6 quadrados saiu %v, esperado [\"9,0m\"] — 6 × 1,5m (p236)", rotulos)
	}
	// E o TOTAL continua trazendo as duas unidades e a faixa.
	total := leitura["reguatexto"].(string)
	for _, pedaco := range []string{"6 quadrados", "9,0m", "alcance"} {
		if !strings.Contains(total, pedaco) {
			t.Errorf("a frase do total perdeu %q: %q", pedaco, total)
		}
	}
}

// TestAPernaDeZeroNaoTemRotulo.
//
// O instante logo depois de um clique tem a MIRA em cima da parada que acabou de
// nascer. Um "0,0m" piscando sob o dedo é ruído sobre o gesto que a pessoa está
// fazendo, e o vazio é o que apaga o nó (ver `oRotuloExiste`).
func TestAPernaDeZeroNaoTemRotulo(t *testing.T) {
	leitura := aLeituraDaPolilinha([]engine.Square{{X: 4, Y: 4}, {X: 4, Y: 4}})
	if rotulos := leitura["reguarotulos"].([]string); len(rotulos) != 1 || rotulos[0] != "" {
		t.Errorf("a perna de zero saiu %v, esperado um rótulo VAZIO", rotulos)
	}
}

// TestNenhumaExpressaoIndexaOSinalDaLista — o guarda da FAMÍLIA, e ele existe
// porque a armadilha custou duas rodadas na bancada.
//
// O sinal do Datastar é um PROXY REATIVO: ler um índice que não existe o CRIA.
// Com a reserva de doze rótulos no ar, `$reguapontos[i]` encheu o sinal de
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
func TestNenhumaExpressaoIndexaOSinalDaLista(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	// O CONTROLE: as expressões da régua ESTÃO na página. Sem ele, não achar
	// `$reguapontos[` seria verdade também sobre uma cena que não desenhou régua
	// nenhuma.
	if !strings.Contains(tela, "reguapontos") {
		t.Fatal("a cena não tem as expressões da régua — o guarda mediria o vazio")
	}
	// A REGRA: depois de `$lista` só pode vir `=` (uma escrita) ou `]` (o fim de
	// um `[...$lista]`, que é a CÓPIA). Qualquer outra coisa — `[`, `.`, `;` — é
	// acesso ao proxy, e o proxy cria.
	//
	// A primeira versão deste guarda procurava só `$lista[`, e ela passou VERDE
	// sobre a segunda forma do mesmo defeito: `const lista = $reguapontos;` põe o
	// PROXY na constante, e `lista[12]` cria o índice do mesmo jeito. Provado na
	// bancada sabotando as duas formas — a primeira acusou, a segunda não. Por
	// isso a regra é sobre o que PODE vir depois, e não sobre uma forma errada
	// conhecida.
	for _, lista := range []string{"reguapontos", "reguarotulos"} {
		acessos := regexp.MustCompile(`\$`+lista+`\s*(.)`).FindAllStringSubmatch(tela, -1)
		if len(acessos) == 0 {
			t.Errorf("`$%s` não aparece na cena — o controle acima não alcançou esta lista", lista)
		}
		for _, a := range acessos {
			if a[1] != "=" && a[1] != "]" {
				t.Errorf("a cena acessa `$%s` seguido de %q: o proxy do Datastar CRIA o que se lê "+
					"e a lista se enche de strings vazias, sem erro nenhum. Copie com `[...$%s]`.",
					lista, a[1], lista)
			}
		}
	}
}

// TestATelaLigaOsQuatroGestosDaRegua.
//
// Uma afirmação sobre a FORMA do que a página serve, e é o único jeito de
// alcançar os quatro de uma vez: clique acrescenta, duplo clique congela, botão
// direito apaga, e o ponteiro leva a perna viva atrás do dedo.
//
// O `evt.button !== 0` entra na lista porque ele é conserto de um defeito
// medido: sem ele, o clique sintético que acompanha o botão direito nasce com
// `offsetX` zero e a régua renasce na ORIGEM do plano no mesmo gesto que a apagou.
func TestATelaLigaOsQuatroGestosDaRegua(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	for _, pedaco := range []string{
		"data-on:dblclick",
		"data-on:contextmenu",
		"data-on:pointermove",
		"evt.button !== 0",
		"tabuleiro/regua",
	} {
		if !strings.Contains(tela, pedaco) {
			t.Errorf("a cena não tem %q: um dos gestos da régua não acontece", pedaco)
		}
	}
}

// TestAReguaForjadaERecusada: o teto de paradas é o tamanho da RESERVA de nós no
// `.templ`, e uma polilinha maior teria pernas medidas que ninguém desenha.
func TestAReguaForjadaERecusada(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	pontos := make([]string, 0, oMaximoDeParadas+2)
	for i := range oMaximoDeParadas + 2 {
		pontos = append(pontos, "["+string(rune('0'+i%10))+",0]")
	}
	corpo := f.posta(t, f.mestre, f.urlDaMesa()+"/tabuleiro/regua",
		`{"reguapontos":[`+strings.Join(pontos, ",")+`],"reguafase":2}`)
	if !strings.Contains(corpo, "teto") {
		t.Errorf("uma régua com %d paradas não foi recusada: %q", len(pontos), corpo)
	}
}

// TestAEsferaNasceNaIntersecao — REGRA DO LIVRO, não escolha de tela.
//
// p225, conferido no PDF p231:
//
//	"Esfera. Surge na INTERSEÇÃO DE QUATRO QUADRADOS, estendendo-se em todas as
//	 direções até o limite de seu raio."
//	"Quadrado. Surge NO QUADRADO ou quadrados escolhidos."
//
// O dono pediu uma escolha de "montar esfera centralizada" e o livro respondeu
// que escolha não há. O `engine.sphereSquares` já desenhava a partir do CANTO —
// quem errava era a tela, que mandava o quadrado do `floor` do clique: até meio
// quadrado entre onde o dedo estava e onde a bola caía, e nada dizendo por quê.
//
// O guarda prende as DUAS pontas — a regra e o gesto — porque separadas elas já
// divergiram uma vez.
func TestAEsferaNasceNaIntersecao(t *testing.T) {
	if !aFormaNasceNaIntersecao(engine.AreaSphere) {
		t.Error("a esfera deixou de nascer na interseção (p225)")
	}
	for _, outra := range []engine.AreaKind{engine.AreaSquare, engine.AreaCone, engine.AreaLine} {
		if aFormaNasceNaIntersecao(outra) {
			t.Errorf("%q passou a nascer na interseção, e o livro só diz isso da esfera (p225)", outra)
		}
	}

	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()
	if !strings.Contains(tela, "Math.round((evt.offsetX") {
		t.Error("a tela não arredonda o clique para o canto: com `floor` a esfera cai " +
			"meio quadrado longe do dedo, e o defeito é silencioso")
	}
	if !strings.Contains(tela, "gabaritonaintersecao") {
		t.Error("a tela não pergunta ao servidor onde a forma nasce — a regra virou " +
			"uma segunda cópia na expressão")
	}
}
