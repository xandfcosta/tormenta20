package table

import (
	"fmt"
	"strings"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// O TRILHO DE FERRAMENTAS do tabuleiro: uma ferramenta ativa por vez, com o que
// o clique faz legível ANTES do clique.
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
	// ID é o valor que o sinal `$tool` guarda. Vazio é MOVER, que é o
	// estado de repouso da cena.
	ID string
	// Shortcut é a tecla, e ela é fixa por ferramenta (ver o comentário do topo).
	Shortcut string
	Label    string
	Icon     string
	Hint     string
	// GMOnly: pintar chão e marcar lugar são gestos de quem MONTA a mesa.
	GMOnly bool
	// Hue é a classe que tinge o ícone com a cor da espécie, nos pincéis de
	// terreno. Vazio nas outras.
	//
	// O botão mostra o MESMO ícone que a casa recebe — o mestre reconhece o pincel
	// pelo que ele pinta, e não por uma amostra de cor ao lado.
	Hue string
}

// EraserTool é o valor do sinal quando o clique LIMPA a casa.
//
// FERRAMENTA e não um modo que inverte o pincel na mão: como modo, clicar com
// `Cobertura` num quadrado de `Difícil` apagava a cobertura que não estava lá —
// em silêncio. Limpando a casa inteira, o pincel na mão não importa, e não
// existe mais o caso em que o gesto não faz nada.
const EraserTool = "borracha"

// MapTools é o trilho inteiro, na ordem em que ele desenha.
//
// A ordem é a do USO e não a do alfabeto: mover primeiro porque é o repouso e o
// retorno de toda outra; medir e mirar em seguida, que são de TODO MUNDO — "dá
// para acertar daqui?" é pergunta de quem ataca; e as do mestre por último,
// agrupadas, com a borracha fechando porque ela é o desfazer das quatro acima.
func MapTools() []mapTool {
	rail := []mapTool{
		{ID: "", Label: "Mover a peça", Icon: "MousePointer2",
			Hint: "Mover a peça: o clique escolhe a casa para onde ela vai"},
		// A MÃO é a SEGUNDA e não a última, e ela é de TODO MUNDO: sem moldura não há
		// rolagem nativa, então arrastar a vista deixou de ser conforto e virou o único
		// jeito de chegar ao outro lado do plano.
		{ID: ViewTool, Label: "Arrastar a vista", Icon: "Hand",
			Hint: "Arrastar a vista: o clique e o arrasto percorrem o plano, que não tem bordas"},
		{ID: FerramentaDaRegua, Label: "Régua", Icon: "Ruler",
			Hint: "Régua: mede a distância e diz a faixa de alcance do livro (p224)"},
		{ID: FerramentaDoGabarito, Label: "Gabarito", Icon: "Radar",
			Hint: "Gabarito de área: a esfera, o cone, a linha e o quadrado (p225), e quem eles pegam"},
		{ID: MarkTool, Label: "Marcar", Icon: "MapPin", GMOnly: true,
			Hint: "Marcar um lugar: o clique põe um ponto ESCONDIDO no mapa, para revelar quando quiser"},
	}
	// Os PINCÉIS saem da lista de espécies e nunca de uma cópia escrita à mão: a
	// quinta espécie nasce no trilho, com atalho, sem ninguém lembrar disto.
	for _, brush := range board.TerrainKinds {
		rail = append(rail, mapTool{
			ID: string(brush.ID), Label: brush.Label, GMOnly: true,
			Icon: drawing(brush.ID).Icon,
			Hint: brush.Label + ": " + brush.Effect + " (p238)",
			Hue:  "brush-hue board-hue-" + board.ClassOf(brush.ID),
		})
	}
	rail = append(rail, mapTool{
		ID: EraserTool, Label: "Borracha", Icon: "Eraser", GMOnly: true,
		Hint: "Borracha: o clique limpa a casa inteira, seja qual for o terreno nela",
	})
	return numberRail(rail)
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
// Digitados à mão em cada linha, inserir uma ferramenta no meio exigiria
// renumerar as de baixo, e uma esquecida daria duas ferramentas com a mesma
// tecla — a segunda simplesmente nunca ligaria, sem erro nenhum.
func numberRail(rail []mapTool) []mapTool {
	if len(rail) > len(railKeys) {
		panic(fmt.Sprintf("o trilho tem %d ferramentas e só há %d teclas: %v",
			len(rail), len(railKeys), railKeys))
	}
	for i := range rail {
		rail[i].Shortcut = railKeys[i]
	}
	return rail
}

// rail devolve as ferramentas que aquele papel realmente tem.
//
// Filtrar AQUI e não no `.templ` é o que faz o atalho de teclado e o botão
// concordarem sobre quem existe: os dois leem esta função. Escritos em dois
// lugares, o jogador ganharia uma tecla que liga uma ferramenta sem botão.
func rail(gm bool) []mapTool {
	return forVisible(gm, MapTools())
}

// forVisible é o filtro, e ele recebe o trilho em vez de buscá-lo.
//
// Separado por causa do GUARDA: a promessa do número fixo só é interessante
// quando uma ferramenta SÓ DO MESTRE vem ANTES de uma compartilhada — hoje todas
// as do mestre estão no fim, e um teste sobre o trilho real passaria mesmo com a
// numeração feita depois do filtro. Com o trilho como parâmetro, o guarda monta
// o caso que importa em vez de esperar que a ordem real o produza um dia.
func forVisible(gm bool, rail []mapTool) []mapTool {
	outside := make([]mapTool, 0, len(rail))
	for _, f := range rail {
		if gm || !f.GMOnly {
			outside = append(outside, f)
		}
	}
	return outside
}

// railKeyboard liga as ferramentas às teclas numéricas.
//
// A GUARDA DE ALVO DE DIGITAÇÃO é a mesma do zoom e do atalho da barra, e ela
// não é zelo: sem ela, digitar "5" no PV de um combatente trocaria a ferramenta
// do mapa atrás do formulário. Já aconteceu com o `-` do zoom.
//
// Montado a partir do MESMO trilho que desenha os botões: uma tabela escrita à
// mão aqui seria a segunda verdade sobre qual tecla liga o quê.
func railKeyboard(gm bool) string {
	var cases []string
	for _, f := range rail(gm) {
		cases = append(cases, fmt.Sprintf("evt.key === %q ? ($tool = %q)", f.Shortcut, f.ID))
	}
	// ESC NÃO ENTRA AQUI, e isto é medido e não escolhido.
	//
	// Ele já tem dono: o `scene.js` mapeia Escape para "voltar" na gramática do
	// teclado e chama `preventDefault` + `stopPropagation` no `document` — o evento
	// **nunca chega à janela**, que é onde o `__window` escuta. Controle: um
	// `keydown` de `F2` no mesmo nó liga a ferramenta, e o de `Escape` não chega nem
	// a um `addEventListener` cru na janela.
	//
	// A saída para quem ligou a régua sem querer é a TECLA 1, que é a ferramenta de
	// repouso — ou clicar de novo na que está acesa, que o `pickTool` já desliga.
	cases = append(cases, "null")
	return typingTargetWithout + "(" + strings.Join(cases, " : ") + ")"
}

// typingTargetWithout é o prefixo que impede um atalho de roubar a tecla de quem
// está escrevendo. Extraído porque três atalhos do tabuleiro o repetiam, e o
// quarto é sempre o que esquece.
const typingTargetWithout = `!['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName) && ` +
	`!document.activeElement?.isContentEditable && `

// onIsTool é o teste que marca o botão e mostra a camada dela.
func onIsTool(id string) string {
	return fmt.Sprintf("$tool === %q", id)
}

// toolStyling liga UMA das duas aparências, e nunca deixa as duas.
//
// Os dois lados no `data-class` pela armadilha de CASCATA que o editor de bloco
// documenta: a marca de ligada mora em `@layer components` e as cores do
// Tailwind são utilidades, numa camada POSTERIOR — camada vence especificidade,
// e o dourado perderia para o cinza sem nada acusar.
func toolStyling(id string) string {
	return fmt.Sprintf("{'brush-on': %s, 'text-muted-foreground': !(%s)}",
		onIsTool(id), onIsTool(id))
}

// shortcutName é o que o leitor de tela e o `title` recebem.
//
// A tecla vai no NOME ACESSÍVEL e não só no `title`: um atalho que só existe no
// balão do mouse é um atalho que quem navega por teclado nunca descobre — e é
// justamente essa pessoa que mais o usaria.
func shortcutName(f mapTool) string {
	return fmt.Sprintf("%s (tecla %s)", f.Label, f.Shortcut)
}

// piecesSizes são os SEIS tamanhos do livro, com o que cada um decide.
//
// Eram QUATRO, e eram o LADO em quadrados — 1, 2, 3 e 6 (p107, Tab. 1-21). O
// lado não consegue carregar a categoria: Minúsculo, Pequeno e Médio ocupam
// todos 1 quadrado, e a Tab. 5-4 (p239) lhes dá Defesa 15, 12 e 10. Uma porta
// minúscula e um barril médio eram a mesma escolha na tira, e a Defesa de um
// deles sairia errada.
//
// Então a tira passou a escolher a CATEGORIA, e o lado virou derivado — o
// `engine.FootprintForSize` o produz, e a peça o recalcula a cada mutação. O
// `Side` aqui é só rótulo: quem decide o desenho é o motor.
//
// A DEFESA viaja junto porque é ela que a tira mostra ao escolher, e o número
// vem do motor — escrevê-lo aqui seria a segunda grafia da escada da p239.
var piecesSizes = func() []struct {
	ID      string
	Side    string
	Defense int
} {
	out := []struct {
		ID      string
		Side    string
		Defense int
	}{}
	for _, nome := range engine.SizesOfTheBook() {
		defesa, err := engine.ObjectDefense(nome)
		if err != nil {
			// IMPOSSÍVEL por construção, e por isso explode no nascimento do
			// processo em vez de servir uma tira com um buraco: a lista acima e a
			// escada do motor são a mesma seis.
			panic(fmt.Sprintf("a tira da peça conhece o tamanho %q e o motor não: %v", nome, err))
		}
		out = append(out, struct {
			ID      string
			Side    string
			Defense int
		}{nome, fmt.Sprintf("%d×%d", engine.FootprintForSize(nome), engine.FootprintForSize(nome)), defesa})
	}
	return out
}()

// piecesMaterials são os materiais de OBJETO da Tab. 5-4, com a RD de cada um.
//
// Não são os materiais ESPECIAIS do item de ficha (mitral, aço-rubi, p166) — a
// palavra é a mesma e as coisas não são, e a colisão está registrada como C10 no
// GLOSSARY.md. A RD vem do motor pela mesma razão que a Defesa acima.
var piecesMaterials = func() []struct {
	ID string
	Rd int
} {
	out := []struct {
		ID string
		Rd int
	}{}
	for _, nome := range engine.ObjectMaterialsOfTheBook() {
		rd, err := engine.ObjectDamageReduction(nome)
		if err != nil {
			panic(fmt.Sprintf("a tira da peça conhece o material %q e o motor não: %v", nome, err))
		}
		out = append(out, struct {
			ID string
			Rd int
		}{nome, rd})
	}
	return out
}()

// piecesLooks são as duas aparências que a peça avulsa pode ter.
//
// `character` fica de fora de propósito, e o servidor recusa: a peça de ficha
// nasce ligada a um personagem pelo `Populate`, e uma "ficha" desenhada à mão
// seria uma peça que PARECE de jogador sem ninguém atrás dela.
var piecesLooks = []struct {
	ID    string
	Label string
	Hint  string
}{
	{"object", "Objeto", "Objeto: a porta, o baú, o barril — cenário que ocupa casa"},
	{"npc", "NPC", "NPC: a criatura que está no mapa e ainda não entrou na fila"},
}

// objectDefenseOf é a Defesa de um tamanho, para a tira escrever no `dataset` da
// opção. Ela ENGOLE o erro de propósito, e pode: a lista que a chama é a dos
// exemplos do próprio motor, e um tamanho fora da escada já teria explodido o
// `piecesSizes` no nascimento do processo.
func objectDefenseOf(size string) int {
	defesa, err := engine.ObjectDefense(size)
	if err != nil {
		panic(fmt.Sprintf("o exemplo da Tab. 5-4 traz o tamanho %q, que a escada não tem: %v", size, err))
	}
	return defesa
}

// objectRdOf é a irmã dela para o material.
func objectRdOf(material string) int {
	rd, err := engine.ObjectDamageReduction(material)
	if err != nil {
		panic(fmt.Sprintf("o exemplo da Tab. 5-4 traz o material %q, que a escada não tem: %v", material, err))
	}
	return rd
}
