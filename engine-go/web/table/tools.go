package table

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

// mapTool é uma entrada do trilho.
type mapTool struct {
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
	// Matiz é a classe que tinge o ícone com a cor da espécie, nos pincéis de
	// terreno. Vazio nas outras.
	//
	// Ela substituiu um QUADRADINHO de cor (`Amostra`), e a troca veio junto com o
	// desenho novo das casas (ALE-203): quando cada espécie ganhou ícone próprio,
	// quatro quadradinhos que só diferiam de matiz viraram a legenda que o desenho
	// da casa acabara de tornar desnecessária. Agora o botão mostra o MESMO ícone
	// que a casa recebe — o mestre reconhece o pincel pelo que ele pinta.
	Matiz string
}

// EraserTool é o valor do sinal quando o clique LIMPA a casa.
//
// Ela era um modo (`$apagando`) que invertia o pincel selecionado, e isso
// produziu o defeito que o dono relatou como "a borracha não funciona": com
// `Cobertura` na mão, clicar num quadrado de `Difícil` apagava a cobertura que
// não estava lá — em silêncio. Medido na bancada, clique a clique.
//
// Agora ela é FERRAMENTA e limpa a casa inteira (decisão do dono): o pincel na
// mão não importa, e não existe mais o caso em que o gesto não faz nada.
const EraserTool = "borracha"

// MapTools é o trilho inteiro, na ordem em que ele desenha.
//
// A ordem é a do USO e não a do alfabeto: mover primeiro porque é o repouso e o
// retorno de toda outra; medir e mirar em seguida, que são de TODO MUNDO — "dá
// para acertar daqui?" é pergunta de quem ataca; e as do mestre por último,
// agrupadas, com a borracha fechando porque ela é o desfazer das quatro acima.
func MapTools() []mapTool {
	trilho := []mapTool{
		{ID: "", Rotulo: "Mover a peça", Icone: "MousePointer2",
			Dica: "Mover a peça: o clique escolhe a casa para onde ela vai"},
		// A MÃO é a SEGUNDA e não a última, e ela é de TODO MUNDO: sem moldura
		// não há rolagem nativa, então arrastar a vista deixou de ser conforto e
		// virou o único jeito de chegar ao outro lado do plano (ALE-203).
		{ID: ViewTool, Rotulo: "Arrastar a vista", Icone: "Hand",
			Dica: "Arrastar a vista: o clique e o arrasto percorrem o plano, que não tem bordas"},
		{ID: FerramentaDaRegua, Rotulo: "Régua", Icone: "Ruler",
			Dica: "Régua: mede a distância e diz a faixa de alcance do livro (p224)"},
		{ID: FerramentaDoGabarito, Rotulo: "Gabarito", Icone: "Radar",
			Dica: "Gabarito de área: a esfera, o cone, a linha e o quadrado (p225), e quem eles pegam"},
		{ID: MarkTool, Rotulo: "Marcar", Icone: "MapPin", SoMestre: true,
			Dica: "Marcar um lugar: o clique põe um ponto ESCONDIDO no mapa, para revelar quando quiser"},
	}
	// Os PINCÉIS saem da lista de espécies e nunca de uma cópia escrita à mão: a
	// quinta espécie nasce no trilho, com atalho, sem ninguém lembrar disto.
	for _, pincel := range tabuleiro.TerrainKinds {
		trilho = append(trilho, mapTool{
			ID: string(pincel.ID), Rotulo: pincel.Rotulo, SoMestre: true,
			Icone: drawing(pincel.ID).Icone,
			Dica:  pincel.Rotulo + ": " + pincel.Efeito + " (p238)",
			Matiz: "pincel-matiz tabuleiro-matiz-" + string(pincel.ID),
		})
	}
	trilho = append(trilho, mapTool{
		ID: EraserTool, Rotulo: "Borracha", Icone: "Eraser", SoMestre: true,
		Dica: "Borracha: o clique limpa a casa inteira, seja qual for o terreno nela",
	})
	return numberRail(trilho)
}

// railKeys é a fileira de números do teclado, na ordem em que a mão a
// percorre: as nove digitais e o zero fechando, que é onde a borracha cai.
//
// DEZ é o teto desta gramática, e ele está escrito aqui de propósito: a décima
// primeira ferramenta não ganha uma letra sorteada — ela pede outra ideia
// (submenu, ferramenta que troca de modo), e o `numberRail` faz o problema
// aparecer em vez de nascer sem atalho em silêncio.
var railKeys = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}

// numberRail escreve o atalho de cada ferramenta a partir da posição dela.
//
// Os números eram digitados à mão em cada linha, e a mão errou na primeira
// oportunidade: pôr a vista em segundo lugar teria exigido renumerar as seis
// abaixo, e uma esquecida daria duas ferramentas com a mesma tecla — a segunda
// simplesmente nunca ligaria, sem erro nenhum.
func numberRail(trilho []mapTool) []mapTool {
	if len(trilho) > len(railKeys) {
		panic(fmt.Sprintf("o trilho tem %d ferramentas e só há %d teclas: %v",
			len(trilho), len(railKeys), railKeys))
	}
	for i := range trilho {
		trilho[i].Atalho = railKeys[i]
	}
	return trilho
}

// rail devolve as ferramentas que aquele papel realmente tem.
//
// Filtrar AQUI e não no `.templ` é o que faz o atalho de teclado e o botão
// concordarem sobre quem existe: os dois leem esta função. Escritos em dois
// lugares, o jogador ganharia uma tecla que liga uma ferramenta sem botão.
func rail(mestre bool) []mapTool {
	return forVisible(mestre, MapTools())
}

// forVisible é o filtro, e ele recebe o trilho em vez de buscá-lo.
//
// Separado por causa do GUARDA: a promessa do número fixo só é interessante
// quando uma ferramenta SÓ DO MESTRE vem ANTES de uma compartilhada — hoje todas
// as do mestre estão no fim, e um teste sobre o trilho real passaria mesmo com a
// numeração feita depois do filtro. Com o trilho como parâmetro, o guarda monta
// o caso que importa em vez de esperar que a ordem real o produza um dia.
func forVisible(mestre bool, trilho []mapTool) []mapTool {
	fora := make([]mapTool, 0, len(trilho))
	for _, f := range trilho {
		if mestre || !f.SoMestre {
			fora = append(fora, f)
		}
	}
	return fora
}

// railKeyboard liga as ferramentas às teclas numéricas.
//
// A GUARDA DE ALVO DE DIGITAÇÃO é a mesma do zoom e do atalho da barra, e ela
// não é zelo: sem ela, digitar "5" no PV de um combatente trocaria a ferramenta
// do mapa atrás do formulário. Já aconteceu com o `-` do zoom.
//
// Montado a partir do MESMO trilho que desenha os botões: uma tabela escrita à
// mão aqui seria a segunda verdade sobre qual tecla liga o quê.
func railKeyboard(mestre bool) string {
	var casos []string
	for _, f := range rail(mestre) {
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
	// `pickTool` já desliga. Escrever um ramo de Escape aqui seria uma
	// promessa que a tela não cumpre.
	casos = append(casos, "null")
	return typingTargetWithout + "(" + strings.Join(casos, " : ") + ")"
}

// typingTargetWithout é o prefixo que impede um atalho de roubar a tecla de quem
// está escrevendo. Extraído porque três atalhos do tabuleiro o repetiam, e o
// quarto é sempre o que esquece.
const typingTargetWithout = `!['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName) && ` +
	`!document.activeElement?.isContentEditable && `

// onIsTool é o teste que marca o botão e mostra a camada dela.
func onIsTool(id string) string {
	return fmt.Sprintf("$ferramenta === %q", id)
}

// toolStyling liga UMA das duas aparências, e nunca deixa as duas.
//
// Os dois lados no `data-class` pela armadilha de CASCATA que o editor de bloco
// documenta: a marca de ligada mora em `@layer components` e as cores do
// Tailwind são utilidades, numa camada POSTERIOR — camada vence especificidade,
// e o dourado perderia para o cinza sem nada acusar.
func toolStyling(id string) string {
	return fmt.Sprintf("{'pincel-ligado': %s, 'text-muted-foreground': !(%s)}",
		onIsTool(id), onIsTool(id))
}

// shortcutName é o que o leitor de tela e o `title` recebem.
//
// A tecla vai no NOME ACESSÍVEL e não só no `title`: um atalho que só existe no
// balão do mouse é um atalho que quem navega por teclado nunca descobre — e é
// justamente essa pessoa que mais o usaria.
func shortcutName(f mapTool) string {
	return fmt.Sprintf("%s (tecla %s)", f.Rotulo, f.Atalho)
}

// piecesFootprints são os tamanhos que a Tabela 1-21 produz (T20 p107).
//
// Quatro e não uma faixa: 4 e 5 não são tamanho de criatura nenhuma. A tira
// desenha botões por isso — um campo numérico convidaria a digitar o que o
// servidor vai recusar, e recusa que se descobre clicando é pior que a escolha
// não existir.
var piecesFootprints = []struct {
	Lados  int
	Lado   string
	Rotulo string
}{
	{1, "1×1", "Médio"},
	{2, "2×2", "Grande"},
	{3, "3×3", "Enorme"},
	{6, "6×6", "Colossal"},
}

// piecesLooks são as duas aparências que a peça avulsa pode ter.
//
// `character` fica de fora de propósito, e o servidor recusa: a peça de ficha
// nasce ligada a um personagem pelo `Populate`, e uma "ficha" desenhada à mão
// seria uma peça que PARECE de jogador sem ninguém atrás dela.
var piecesLooks = []struct {
	ID     string
	Rotulo string
	Dica   string
}{
	{"object", "Objeto", "Objeto: a porta, o baú, o barril — cenário que ocupa casa"},
	{"npc", "NPC", "NPC: a criatura que está no mapa e ainda não entrou na fila"},
}
