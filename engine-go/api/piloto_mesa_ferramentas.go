package api

import (
	"fmt"
	"strings"

	"t20engine/tabuleiro"
)

// O TRILHO DE FERRAMENTAS do tabuleiro (ALE-203), em Datastar.
//
// A ALE-269 entregou o MODELO desta issue — uma ferramenta ativa por vez, com o
// que o clique faz legível ANTES do clique — e não entregou a GRAMÁTICA que ela
// descreve. O dono usou e apontou; o que muda aqui é a FORMA:
//
//  1. o trilho é VERTICAL e SOBREPÕE o tabuleiro, em vez de ser uma fileira
//     horizontal que empurra o mapa para baixo;
//  2. cada ferramenta tem um NÚMERO de atalho;
//  3. a BORRACHA deixa de ser um modo do pincel e vira ferramenta própria.
//
// # Por que o número sai do trilho INTEIRO, e não do trilho que aparece
//
// O trilho do jogador tem quatro entradas e o do mestre tem dez. Numerar o que
// cada papel VÊ faria a régua ser `3` para um e `2` para o outro, e a primeira
// ferramenta só do mestre desalinharia tudo. O número sai da posição na lista
// COMPLETA, antes de filtrar: quem aprendeu `4 = gabarito` mestrando continua
// com `4 = gabarito` jogando, e os números que o jogador não tem simplesmente
// não fazem nada.

// ferramentaDoMapa é uma entrada do trilho.
type ferramentaDoMapa struct {
	// ID é o valor que o sinal `$ferramenta` guarda. Vazio é MOVER, que é o
	// estado de repouso da cena.
	ID string
	// Atalho é a tecla, e ela é fixa por ferramenta (ver o comentário do topo).
	Atalho string
	Rotulo string
	Icone  string
	Dica   string
	// SoMestre: pintar chão e marcar lugar são gestos de quem MONTA a mesa.
	SoMestre bool
	// Amostra é a classe do quadradinho de cor dos pincéis de terreno — eles
	// mostram o que pintam em vez de um ícone genérico. Vazio nas outras.
	Amostra string
}

// FerramentaDaBorracha é o valor do sinal quando o clique LIMPA a casa.
//
// Ela era um modo (`$apagando`) que invertia o pincel selecionado, e isso
// produziu o defeito que o dono relatou como "a borracha não funciona": com
// `Cobertura` na mão, clicar num quadrado de `Difícil` apagava a cobertura que
// não estava lá — em silêncio. Medido na bancada, clique a clique.
//
// Agora ela é FERRAMENTA e limpa a casa inteira (decisão do dono): o pincel na
// mão não importa, e não existe mais o caso em que o gesto não faz nada.
const FerramentaDaBorracha = "borracha"

// asFerramentasDoMapa é o trilho inteiro, na ordem em que ele desenha.
//
// A ordem é a do USO e não a do alfabeto: mover primeiro porque é o repouso e o
// retorno de toda outra; medir e mirar em seguida, que são de TODO MUNDO — "dá
// para acertar daqui?" é pergunta de quem ataca; e as do mestre por último,
// agrupadas, com a borracha fechando porque ela é o desfazer das quatro acima.
func asFerramentasDoMapa() []ferramentaDoMapa {
	trilho := []ferramentaDoMapa{
		{ID: "", Rotulo: "Mover a peça", Icone: "MousePointer2",
			Dica: "Mover a peça: o clique escolhe a casa para onde ela vai"},
		// A MÃO é a SEGUNDA e não a última, e ela é de TODO MUNDO: sem moldura
		// não há rolagem nativa, então arrastar a vista deixou de ser conforto e
		// virou o único jeito de chegar ao outro lado do plano (ALE-203).
		{ID: FerramentaDaVista, Rotulo: "Arrastar a vista", Icone: "Hand",
			Dica: "Arrastar a vista: o clique e o arrasto percorrem o plano, que não tem bordas"},
		{ID: FerramentaDaRegua, Rotulo: "Régua", Icone: "Ruler",
			Dica: "Régua: mede a distância e diz a faixa de alcance do livro (p224)"},
		{ID: FerramentaDoGabarito, Rotulo: "Gabarito", Icone: "Radar",
			Dica: "Gabarito de área: a esfera, o cone, a linha e o quadrado (p225), e quem eles pegam"},
		{ID: FerramentaDeMarcar, Rotulo: "Marcar", Icone: "MapPin", SoMestre: true,
			Dica: "Marcar um lugar: o clique põe um ponto ESCONDIDO no mapa, para revelar quando quiser"},
	}
	// Os PINCÉIS saem da lista de espécies e nunca de uma cópia escrita à mão: a
	// quinta espécie nasce no trilho, com atalho, sem ninguém lembrar disto.
	for _, pincel := range tabuleiro.EspeciesDeTerreno {
		trilho = append(trilho, ferramentaDoMapa{
			ID: string(pincel.ID), Rotulo: pincel.Rotulo, SoMestre: true,
			Dica:    pincel.Rotulo + ": " + pincel.Efeito + " (p238)",
			Amostra: "pincel-amostra tabuleiro-" + string(pincel.ID),
		})
	}
	trilho = append(trilho, ferramentaDoMapa{
		ID: FerramentaDaBorracha, Rotulo: "Borracha", Icone: "Eraser", SoMestre: true,
		Dica: "Borracha: o clique limpa a casa inteira, seja qual for o terreno nela",
	})
	return numeraOTrilho(trilho)
}

// asTeclasDoTrilho é a fileira de números do teclado, na ordem em que a mão a
// percorre: as nove digitais e o zero fechando, que é onde a borracha cai.
//
// DEZ é o teto desta gramática, e ele está escrito aqui de propósito: a décima
// primeira ferramenta não ganha uma letra sorteada — ela pede outra ideia
// (submenu, ferramenta que troca de modo), e o `numeraOTrilho` faz o problema
// aparecer em vez de nascer sem atalho em silêncio.
var asTeclasDoTrilho = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}

// numeraOTrilho escreve o atalho de cada ferramenta a partir da posição dela.
//
// Os números eram digitados à mão em cada linha, e a mão errou na primeira
// oportunidade: pôr a vista em segundo lugar teria exigido renumerar as seis
// abaixo, e uma esquecida daria duas ferramentas com a mesma tecla — a segunda
// simplesmente nunca ligaria, sem erro nenhum.
func numeraOTrilho(trilho []ferramentaDoMapa) []ferramentaDoMapa {
	if len(trilho) > len(asTeclasDoTrilho) {
		panic(fmt.Sprintf("o trilho tem %d ferramentas e só há %d teclas: %v",
			len(trilho), len(asTeclasDoTrilho), asTeclasDoTrilho))
	}
	for i := range trilho {
		trilho[i].Atalho = asTeclasDoTrilho[i]
	}
	return trilho
}

// oTrilhoDe devolve as ferramentas que aquele papel realmente tem.
//
// Filtrar AQUI e não no `.templ` é o que faz o atalho de teclado e o botão
// concordarem sobre quem existe: os dois leem esta função. Escritos em dois
// lugares, o jogador ganharia uma tecla que liga uma ferramenta sem botão.
func oTrilhoDe(mestre bool) []ferramentaDoMapa {
	return asVisiveisPara(mestre, asFerramentasDoMapa())
}

// asVisiveisPara é o filtro, e ele recebe o trilho em vez de buscá-lo.
//
// Separado por causa do GUARDA: a promessa do número fixo só é interessante
// quando uma ferramenta SÓ DO MESTRE vem ANTES de uma compartilhada — hoje todas
// as do mestre estão no fim, e um teste sobre o trilho real passaria mesmo com a
// numeração feita depois do filtro. Com o trilho como parâmetro, o guarda monta
// o caso que importa em vez de esperar que a ordem real o produza um dia.
func asVisiveisPara(mestre bool, trilho []ferramentaDoMapa) []ferramentaDoMapa {
	fora := make([]ferramentaDoMapa, 0, len(trilho))
	for _, f := range trilho {
		if mestre || !f.SoMestre {
			fora = append(fora, f)
		}
	}
	return fora
}

// oTecladoDoTrilho liga as ferramentas às teclas numéricas.
//
// A GUARDA DE ALVO DE DIGITAÇÃO é a mesma do zoom e do atalho da barra, e ela
// não é zelo: sem ela, digitar "5" no PV de um combatente trocaria a ferramenta
// do mapa atrás do formulário. Já aconteceu com o `-` do zoom.
//
// Montado a partir do MESMO trilho que desenha os botões: uma tabela escrita à
// mão aqui seria a segunda verdade sobre qual tecla liga o quê.
func oTecladoDoTrilho(mestre bool) string {
	var casos []string
	for _, f := range oTrilhoDe(mestre) {
		casos = append(casos, fmt.Sprintf("evt.key === %q ? ($ferramenta = %q)", f.Atalho, f.ID))
	}
	// ESC NÃO ENTRA AQUI, e isto é medido e não escolhido.
	//
	// Ele já tem dono: o `cena.js` mapeia Escape para "voltar" na gramática do
	// teclado e chama `preventDefault` + `stopPropagation` no `document` — o
	// evento **nunca chega à janela**, que é onde o `__window` escuta. Provado com
	// controle no navegador: um `keydown` de `F2` no mesmo nó liga a ferramenta, e
	// o de `Escape` não chega nem a um `addEventListener` cru na janela.
	//
	// A saída para quem ligou a régua sem querer é a TECLA 1, que é a ferramenta
	// de repouso — ou clicar de novo na que está acesa, que o
	// `escolheAFerramenta` já desliga. Escrever um ramo de Escape aqui seria uma
	// promessa que a tela não cumpre.
	casos = append(casos, "null")
	return semAlvoDeDigitacao + "(" + strings.Join(casos, " : ") + ")"
}

// semAlvoDeDigitacao é o prefixo que impede um atalho de roubar a tecla de quem
// está escrevendo. Extraído porque três atalhos do tabuleiro o repetiam, e o
// quarto é sempre o que esquece.
const semAlvoDeDigitacao = `!['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName) && ` +
	`!document.activeElement?.isContentEditable && `

// aFerramentaEstaLigada é o teste que marca o botão e mostra a camada dela.
func aFerramentaEstaLigada(id string) string {
	return fmt.Sprintf("$ferramenta === %q", id)
}

// oVestidoDaFerramenta liga UMA das duas aparências, e nunca deixa as duas.
//
// Os dois lados no `data-class` pela armadilha de CASCATA que o editor de bloco
// documenta: a marca de ligada mora em `@layer components` e as cores do
// Tailwind são utilidades, numa camada POSTERIOR — camada vence especificidade,
// e o dourado perderia para o cinza sem nada acusar.
func oVestidoDaFerramenta(id string) string {
	return fmt.Sprintf("{'pincel-ligado': %s, 'text-muted-foreground': !(%s)}",
		aFerramentaEstaLigada(id), aFerramentaEstaLigada(id))
}

// oNomeComAtalho é o que o leitor de tela e o `title` recebem.
//
// A tecla vai no NOME ACESSÍVEL e não só no `title`: um atalho que só existe no
// balão do mouse é um atalho que quem navega por teclado nunca descobre — e é
// justamente essa pessoa que mais o usaria.
func oNomeComAtalho(f ferramentaDoMapa) string {
	return fmt.Sprintf("%s (tecla %s)", f.Rotulo, f.Atalho)
}
