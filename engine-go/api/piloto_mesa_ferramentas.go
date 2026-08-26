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
// # Por que o número é FIXO por ferramenta, e não a posição no trilho
//
// O trilho do jogador tem três entradas e o do mestre tem nove. Numerar por
// posição faria a régua ser `2` para um e `2` para o outro por acidente, e a
// primeira ferramenta só do mestre desalinharia tudo. Com número fixo, quem
// aprendeu `3 = gabarito` mestrando continua com `3 = gabarito` jogando — e os
// números que o jogador não tem simplesmente não fazem nada.

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
		{ID: "", Atalho: "1", Rotulo: "Mover a peça", Icone: "MousePointer2",
			Dica: "Mover a peça: o clique escolhe a casa para onde ela vai"},
		{ID: FerramentaDaRegua, Atalho: "2", Rotulo: "Régua", Icone: "Ruler",
			Dica: "Régua: mede a distância e diz a faixa de alcance do livro (p224)"},
		{ID: FerramentaDoGabarito, Atalho: "3", Rotulo: "Gabarito", Icone: "Radar",
			Dica: "Gabarito de área: a esfera, o cone, a linha e o quadrado (p225), e quem eles pegam"},
		{ID: FerramentaDeMarcar, Atalho: "4", Rotulo: "Marcar", Icone: "MapPin", SoMestre: true,
			Dica: "Marcar um lugar: o clique põe um ponto ESCONDIDO no mapa, para revelar quando quiser"},
	}
	// Os PINCÉIS saem da lista de espécies e nunca de uma cópia escrita à mão: a
	// quinta espécie nasce no trilho, com atalho, sem ninguém lembrar disto.
	for i, pincel := range tabuleiro.EspeciesDeTerreno {
		trilho = append(trilho, ferramentaDoMapa{
			ID: string(pincel.ID), Atalho: fmt.Sprint(5 + i), Rotulo: pincel.Rotulo, SoMestre: true,
			Dica:    pincel.Rotulo + ": " + pincel.Efeito + " (p238)",
			Amostra: "pincel-amostra tabuleiro-" + string(pincel.ID),
		})
	}
	return append(trilho, ferramentaDoMapa{
		ID: FerramentaDaBorracha, Atalho: fmt.Sprint(5 + len(tabuleiro.EspeciesDeTerreno)),
		Rotulo: "Borracha", Icone: "Eraser", SoMestre: true,
		Dica: "Borracha: o clique limpa a casa inteira, seja qual for o terreno nela",
	})
}

// oTrilhoDe devolve as ferramentas que aquele papel realmente tem.
//
// Filtrar AQUI e não no `.templ` é o que faz o atalho de teclado e o botão
// concordarem sobre quem existe: os dois leem esta função. Escritos em dois
// lugares, o jogador ganharia uma tecla que liga uma ferramenta sem botão.
func oTrilhoDe(mestre bool) []ferramentaDoMapa {
	fora := make([]ferramentaDoMapa, 0, len(asFerramentasDoMapa()))
	for _, f := range asFerramentasDoMapa() {
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

// limpaNoPontoClicado é o gesto da borracha.
//
// A mesma aritmética das outras camadas — o ponto do clique dividido pelo lado
// da casa, mais a quina da moldura —, e ela é repetida pelo mesmo motivo que a
// `marcacaoNoPontoClicado` a repete: o DESTINO é outro, e um helper compartilhado
// faria as rotas mudarem juntas no dia em que uma delas precisar de outra conta.
func limpaNoPontoClicado(v tabuleiroView) string {
	return fmt.Sprintf(
		"@post('/piloto/mesa/%d/%d/tabuleiro/terreno/limpar/' + (Math.floor(evt.offsetX / $quadrado) + %d) + '/' + (Math.floor(evt.offsetY / $quadrado) + %d))",
		v.CampaignID, v.SessionID, v.X0, v.Y0,
	)
}

// oNomeComAtalho é o que o leitor de tela e o `title` recebem.
//
// A tecla vai no NOME ACESSÍVEL e não só no `title`: um atalho que só existe no
// balão do mouse é um atalho que quem navega por teclado nunca descobre — e é
// justamente essa pessoa que mais o usaria.
func oNomeComAtalho(f ferramentaDoMapa) string {
	return fmt.Sprintf("%s (tecla %s)", f.Rotulo, f.Atalho)
}
