package api

import (
	"fmt"
	"strconv"
	"strings"

	"t20engine/engine"
	"t20engine/tabuleiro"
)

// As expressões da RÉGUA e do GABARITO (ALE-269, superfície 8).
//
// Arquivo próprio e não mais um pedaço do `piloto_mesa_tabuleiro_view.go`, que
// já passa de 800 linhas: o que mora aqui é uma responsabilidade fechada — as
// duas ferramentas que MEDEM, e nenhuma delas muda a cena.
//
// A divisão do trabalho entre os dois lados é a mesma do resto do tabuleiro, e
// vale dizer onde ela cai: o NAVEGADOR guarda as pontas e traduz pixel em
// quadrado (é ele quem sabe o zoom); o SERVIDOR conta os quadrados, escolhe a
// faixa do livro, desenha a forma e diz quem ela pega. Nada do que a régua
// responde é recalculado na tela.

// FerramentaDaRegua e FerramentaDoGabarito são os valores do sinal `$ferramenta`
// quando o clique MEDE em vez de mover ou pintar.
//
// Constantes pela mesma razão da `FerramentaDeMarcar`: cada uma aparece em meia
// dúzia de expressões e `data-show`, e escrita à mão a sexta ocorrência é a que
// erra a letra e vira uma ferramenta que a tela liga e o mapa nunca escuta.
const (
	FerramentaDaRegua    = "regua"
	FerramentaDoGabarito = "gabarito"
)

// oPincelEstaLigado é o teste que separa PINTAR de medir, escrito a partir da
// lista de espécies e nunca à mão.
//
// Ele existe porque a régua e o gabarito quebraram a pergunta antiga: a camada de
// pintura mostrava-se com `$ferramenta != ” && $ferramenta != 'marcador'`, que
// era verdade para toda ferramenta que ainda não existia. Uma lista escrita à mão
// no `.templ` teria o mesmo defeito adiado — a espécie nova nasceria fora dela.
func oPincelEstaLigado() string {
	nomes := make([]string, 0, len(tabuleiro.EspeciesDeTerreno))
	for _, e := range tabuleiro.EspeciesDeTerreno {
		nomes = append(nomes, fmt.Sprintf("%q", string(e.ID)))
	}
	return fmt.Sprintf("[%s].includes($ferramenta)", strings.Join(nomes, ", "))
}

// reguaNoPontoClicado é a máquina de dois cliques da régua.
//
// O primeiro clique fixa a origem, o segundo fecha a medida, e o terceiro
// RECOMEÇA de onde clicou. Sem o recomeço, medir a distância seguinte exigiria
// um botão de limpar — e a mesa mede muitas vezes seguidas, sempre a partir de
// outro lugar.
//
// A ida ao servidor acontece só no clique que FECHA: uma ponta solta não é
// distância nenhuma, e postar a cada clique dobraria a conversa para a metade das
// vezes em que ela não tem resposta.
//
// A aritmética do quadrado é a mesma do `paradaNoPontoClicado`, repetida pelo
// mesmo motivo que a `marcacaoNoPontoClicado` a repete — o destino é outro. Aqui
// ela ganha nome (`cx`, `cy`) porque o valor é usado duas vezes na mesma
// expressão.
func reguaNoPontoClicado(v tabuleiroView) string {
	return fmt.Sprintf(
		"const cx = Math.floor(evt.offsetX / $quadrado) + %d, cy = Math.floor(evt.offsetY / $quadrado) + %d; "+
			"if ($reguafase === 1) { $regua2x = cx; $regua2y = cy; $reguafase = 2; "+
			"@post('/piloto/mesa/%d/%d/tabuleiro/regua/' + $regua1x + '/' + $regua1y + '/' + cx + '/' + cy) } "+
			"else { $regua1x = cx; $regua1y = cy; $regua2x = cx; $regua2y = cy; $reguafase = 1; $reguatexto = '' }",
		v.X0, v.Y0, v.CampaignID, v.SessionID,
	)
}

// guardaARegua apaga as duas pontas E a leitura.
//
// As duas coisas juntas porque elas são a mesma: a frase de uma medida cujo
// desenho sumiu é a resposta a uma pergunta que ninguém consegue mais ver.
const guardaARegua = "$reguafase = 0; $reguatexto = ''"

// gabaritoNoPontoClicado põe o gabarito, e aponta quando a forma pede.
//
// O primeiro clique põe a origem; o segundo, quando a forma aponta, diz para
// onde. Depois disso o clique seguinte RECOMEÇA — a mesa posiciona a mesma bola
// de fogo três vezes antes de decidir, e um botão de limpar entre cada tentativa
// seria um clique a mais em cada uma.
//
// QUEM APONTA é dito pelo `$gabaritoaponta`, escrito pelo botão da forma com o
// valor que o `apontaOGabarito` do servidor deu. Perguntar aqui `$gabarito ===
// 'cone' || $gabarito === 'linha'` seria a segunda cópia da regra da p225, livre
// para divergir da primeira.
func gabaritoNoPontoClicado(v tabuleiroView) string {
	return fmt.Sprintf(
		"const cx = Math.floor(evt.offsetX / $quadrado) + %d, cy = Math.floor(evt.offsetY / $quadrado) + %d; "+
			"if ($gabaritofase === 1 && $gabaritoaponta) { $gabaritomirax = cx; $gabaritomiray = cy; $gabaritofase = 2 } "+
			"else { $gabaritox = cx; $gabaritoy = cy; $gabaritomirax = cx; $gabaritomiray = cy; $gabaritofase = 1 } "+
			"%s",
		v.X0, v.Y0, remedeOGabarito(v),
	)
}

// remedeOGabarito pede o desenho e a lista de quem ele pega.
//
// UMA expressão para os três gestos que mudam o gabarito — pôr, trocar de forma,
// mudar o tamanho —, porque os três fazem a mesma pergunta ao servidor. Três
// cópias dariam três lugares para esquecer de mandar um dos parâmetros, e o
// sintoma seria um gabarito que ignora o número que a pessoa acabou de digitar.
func remedeOGabarito(v tabuleiroView) string {
	return fmt.Sprintf(
		"@post('/piloto/mesa/%d/%d/tabuleiro/gabarito/' + $gabarito + '/' + $gabaritotamanho"+
			" + '/' + $gabaritox + '/' + $gabaritoy + '/' + $gabaritomirax + '/' + $gabaritomiray)",
		v.CampaignID, v.SessionID,
	)
}

// escolheAForma troca a forma e LARGA o que estava posto.
//
// Trocar sem largar foi medido na SPA e desenha errado: o primeiro clique depois
// da troca cai na regra do segundo e APONTA a forma nova a partir da origem da
// antiga — escolher "Cone" com uma esfera na tela desenhava um cone apontado para
// o lado de onde se clicou.
//
// O `$gabaritoaponta` sai daqui com o valor do SERVIDOR: é o botão que sabe qual
// forma ele liga, e é o `apontaOGabarito` que sabe quais formas apontam.
func escolheAForma(forma engine.AreaKind) string {
	return fmt.Sprintf("$gabarito = %q; $gabaritoaponta = %t; %s",
		string(forma), apontaOGabarito(forma), guardaOGabarito)
}

// aDicaDoGabaritoVazio é o que a barra diz enquanto não há gabarito posto, e ela
// vem do SERVIDOR — é a mesma frase que o `quemOGabaritoPega` devolve para uma
// área sem casa nenhuma. Um literal aqui seria a segunda cópia, livre para
// divergir da primeira no dia em que alguém melhorar uma das duas.
var aDicaDoGabaritoVazio = quemOGabaritoPega(nil, nil)

// guardaOGabarito apaga o desenho E a lista, pelo mesmo motivo da régua.
var guardaOGabarito = fmt.Sprintf(
	"$gabaritofase = 0; $gabaritopath = ''; $gabaritotexto = %q", aDicaDoGabaritoVazio)

// aMedidaDaForma é a palavra que nomeia o número que a pessoa digita: a esfera
// tem raio, o cone tem alcance, a linha tem comprimento e o quadrado tem lado
// (p225).
//
// O rótulo muda com a forma porque "tamanho" não responde a pergunta de nenhuma
// das quatro — o conjurador está lendo "raio 6m" na ficha, e é esse número que
// ele quer digitar.
func aMedidaDaForma(k engine.AreaKind) string {
	switch k {
	case engine.AreaSphere:
		return "raio"
	case engine.AreaCone:
		return "alcance"
	case engine.AreaLine:
		return "comprimento"
	default:
		return "lado"
	}
}

// asFormasDoLivro é a ordem em que a mesa as usa (p225), e ela é a mesma da SPA.
var asFormasDoLivro = []engine.AreaKind{
	engine.AreaSphere, engine.AreaCone, engine.AreaLine, engine.AreaSquare,
}

// rotuloDaForma é o nome na tela, com maiúscula.
//
// Escrito por extenso e não derivado do id com um `ToUpper` na primeira letra: o
// id é minúsculo e sem acento porque é identificador, e o dia em que uma forma
// tiver acento no nome a derivação erraria em silêncio.
func rotuloDaForma(k engine.AreaKind) string {
	switch k {
	case engine.AreaSphere:
		return "Esfera"
	case engine.AreaCone:
		return "Cone"
	case engine.AreaLine:
		return "Linha"
	default:
		return "Quadrado"
	}
}

// aPontaDaRegua é a coordenada do CENTRO da casa, em unidades de QUADRADO.
//
// O SVG desenha em quadrados e não em pixels, e é isso que faz a régua acompanhar
// o zoom sem uma linha de JavaScript: o `viewBox` é a moldura em quadrados, e o
// elemento mede `colunas × --quadrado`. Mudar o zoom muda a escala do SVG inteiro.
func aPontaDaRegua(ponta, eixo string) string {
	return fmt.Sprintf("$regua%s%s + 0.5", ponta, eixo)
}

// aQuinaDaMoldura tira do desenho a origem do plano.
//
// O sinal guarda a coordenada ABSOLUTA — o plano não tem bordas e a peça pode
// estar em -3 —, e é o servidor que sabe onde a moldura começa. Pôr a subtração
// aqui, num `transform` que o remendo redesenha, é o que mantém a medida certa
// quando a moldura cresce debaixo dela.
func aQuinaDaMoldura(v tabuleiroView) string {
	return fmt.Sprintf("translate(%d %d)", -v.X0, -v.Y0)
}

// aMolduraEmQuadrados é o `viewBox`: o SVG passa a falar a língua do tabuleiro.
func aMolduraEmQuadrados(v tabuleiroView) string {
	return fmt.Sprintf("0 0 %d %d", v.Colunas, v.Linhas)
}

// oTamanhoEmMetros converte o número digitado para a unidade da FICHA.
//
// É o único cálculo desta superfície que fica do lado do navegador, e a razão é
// a digitação: ele acompanha a tecla, e uma ida ao servidor por tecla trocaria um
// número que muda a cada instante por uma conversa. O FATOR vem do
// `engine.SquareMetres` — escrever `1.5` no `.templ` seria mais uma cópia da
// p236 solta no repositório.
func oTamanhoEmMetros() string {
	return fmt.Sprintf("($gabaritotamanho * %g).toFixed(1).replace('.', ',') + 'm'", engine.SquareMetres)
}

// oMaiorGabarito é o teto da caixa de digitação: o alcance longo do livro
// (p224), que é o maior gabarito que cabe numa mesa. É o MESMO número que o
// `tamanhoDoGabarito` trava do lado do servidor, e da mesma constante — a caixa
// é cortesia, e a trava é de lá.
func oMaiorGabarito() string { return strconv.Itoa(engine.LongRangeSquares) }
