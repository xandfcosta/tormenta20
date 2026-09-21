package table

import (
	"fmt"
	"strings"

	"t20engine/domain/board"
	"t20engine/serve/web/routes"
)

// O GESTO DO TABULEIRO, em EXPRESSÃO do Datastar.
//
// Toda função daqui devolve string: o `@post` que confirma uma parada, o
// `data-on:pointerdown` que começa o arraste, a expressão que faz a peça seguir
// o dedo. Elas mudam quando o PROTOCOLO do gesto muda, e não quando a peça
// ganha um campo — que é a razão de o arquivo existir separado (ALE-360).
//
// # O que NÃO mora aqui
//
// Decisão. Nenhuma destas funções decide se o movimento cabe, quanto custa ou
// quem pode mover: isso é do `domain/board`, e o que chega aqui já veio
// decidido. O que se escreve é o endereço e o payload.
//
// # Por que expressão e não uma ilha de JS
//
// Porque o que elas carregam é o ENDEREÇO do gesto, e o endereço é do servidor.
// Uma ilha precisaria receber o mesmo endereço por atributo, e aí haveria dois
// lugares dizendo a mesma URL.

// moveCommand escreve a chamada de confirmar ou cancelar.
func moveCommand(v BoardView, action string) string {
	return fmt.Sprintf("@post('%s/%s/%s')", v.Base, v.Movement.TokenID, action)
}

// clickedPointStop traduz o PONTO do clique em quadrado do plano.
//
// A conta é do cliente e não do servidor porque ela é sobre PIXELS: o servidor
// não sabe o zoom, que é do navegador. Mas ela não é REGRA — tudo o que decide
// (o caminho, o custo, se cabe) continua do outro lado.
//
// `offsetX/offsetY` são relativos à camada, que cobre o plano inteiro; a origem
// da moldura entra somada porque o quadrado 0 da tela é o `X0` do plano, e ele
// pode ser NEGATIVO.
func clickedPointStop(v BoardView) string {
	return fmt.Sprintf(
		"@post('%s/%s/parada', {payload: {from: {x: (%s), y: (%s)}}})",
		v.Base, v.MoveTarget, clicouEmX, clicouEmY,
	)
}

// ── o ARRASTO, puramente em CSS ──────────────────────────────────────────────
//
// O arrasto é VISUAL até soltar e não toca no DOM que o servidor governa:
// enquanto o dedo está em cima, o que muda é um `transform` alimentado por
// SINAIS, e a posição de verdade só muda quando a parada é aceita.
//
// **Os sinais vivem no `#table`**, que é a única raiz que o remendo nunca toca —
// as variáveis CSS descem por herança até a peça. No plano ou na peça, o
// primeiro remendo de outro jogador as apagaria no meio do gesto.
//
// O que se arrasta é sempre a PEÇA, contando do lugar onde ela está DESENHADA.
// Por isso a próxima parada conta do fim da trilha sem ninguém somar nada: é lá
// que a peça está.

// startsTheDrag escreve o `pointerdown`: marca quem está sendo arrastado e
// guarda o ponto de partida.
//
// NÃO chama `setPointerCapture`, e a ausência é deliberada. Quem faz o gesto
// sobreviver ao dedo sair de cima do elemento aqui é a JANELA: `followsFinger` e
// `dropFor` entram como `pointermove__window`/`pointerup__window`, e a
// janela recebe o evento com ou sem captura. Captura seria redundante e é a
// única chamada da expressão que LANÇA — `NotFoundError` quando o `pointerId`
// não é de um ponteiro ativo. E uma expressão Datastar que lança aborta a
// propagação DEPOIS de já ter escrito os sinais: o `pointerup` calcularia o
// deslocamento certo enquanto `data-class` e `data-attr:style` nunca reagiriam
// — o gesto funcionando e invisível.
//
// O QUE `$dragging` GUARDA É UMA IDENTIDADE, e não o nome do gesto: o ID da
// peça, ou `dragsTheParty` quando o gesto move o grupo. Cada peça pendura o
// próprio par de ouvintes na JANELA, então um valor igual para todas faria os
// `pointerup` de todas passarem na mesma guarda e o primeiro do DOM vencer —
// pegar uma peça moveria outra. Com o ID, cada expressão só reconhece a si
// mesma e a ordem dos ouvintes não importa.
func startsTheDrag(who string) string {
	return fmt.Sprintf(
		"$dragging = '%s'; $drag_start_x = evt.clientX; $drag_start_y = evt.clientY; "+
			"$drag_x = 0; $drag_y = 0", who)
}

// dragsTheParty é o valor de `$dragging` quando o gesto move o GRUPO marcado.
//
// Uma palavra e não um id, porque o grupo não tem um: quem começa é qualquer
// peça marcada, e todas as marcadas se movem juntas. É o único valor que não
// identifica um nó, e é por isso que ele tem nome.
const dragsTheParty = "grupo"

// dragsItself responde se o `pointerdown` DESTA peça move ELA, e não o grupo.
//
// Os quatro pedaços do gesto — pegar, seguir, soltar e deslocar na tela — têm de
// concordar sobre isso, e por isso a divisa mora aqui numa vez só. Cada um com a
// sua pergunta dá a classe de arrasto a UMA peça e o gesto a todas.
func dragsItself(v BoardView, id string) bool {
	return v.Draft || v.DragsToken == id
}

// followsFinger escreve o `pointermove`. Só mexe nos sinais se for ESTE que está
// sendo arrastado: os dois alvos escutam a mesma janela.
func followsFinger(who string) string {
	return fmt.Sprintf(
		"$dragging === '%s' && ($drag_x = evt.clientX - $drag_start_x, $drag_y = evt.clientY - $drag_start_y)", who)
}

// fingerFollowsWithPreview é o `followsFinger` da PEÇA, com a seta viva por cima.
//
// Ele pede a prévia ao servidor SÓ QUANDO O QUADRADO MUDA, e não a cada pixel:
// é a mesma trava do `rulerFollowsPointer`, e ela transforma "um pedido por
// evento de ponteiro" em "um pedido por casa atravessada". Sem ela, um arrasto
// de dois segundos abriria centenas de requisições para desenhar a mesma linha.
//
// A conta do quadrado é a MESMA do `dropFor` (`Math.round` do deslocamento
// pelo `--quadrado`), e tem de ser: se a prévia arredondasse diferente do
// soltar, a pessoa leria um custo e receberia outro — o defeito mais caro que
// esta tela pode ter, porque ele só aparece depois da decisão.
func fingerFollowsWithPreview(v BoardView, p boardToken) string {
	return fmt.Sprintf(
		"if ($dragging !== '%s') return; "+
			"$drag_x = evt.clientX - $drag_start_x; $drag_y = evt.clientY - $drag_start_y; "+
			"const cx = %d + Math.round($drag_x / $square), cy = %d + Math.round($drag_y / $square); "+
			"if (cx === $preview_x && cy === $preview_y) return; "+
			"$preview_x = cx; $preview_y = cy; "+
			"@post('%s/%s/previa', {payload: {from: {x: cx, y: cy}}})",
		p.ID, p.X, p.Y, v.Base, p.ID)
}

// erasePreview limpa a seta viva. Vai no `pointerup`, junto do que solta.
//
// QUEM LIMPA É QUEM TERMINA O GESTO, e não quem começa o próximo: um desenho de
// prévia que sobrevivesse ao soltar ficaria por cima da seta de verdade, com o
// mesmo formato e outra medida — dois caminhos na tela e nenhum jeito de saber
// qual é o que vale. É a mesma regra do nó compartilhado que o diálogo de senha
// ensinou (ver o CLAUDE.md deste pacote).
//
// O `$preview_x` volta para um valor IMPOSSÍVEL e não para zero: zero é uma casa
// legítima do plano, e o próximo arrasto que começasse nela não pediria prévia
// nenhuma — a trava do "só quando o quadrado muda" o engoliria em silêncio.
const erasePreview = "$preview_arrow_fits = ''; $preview_arrow_second = ''; $preview_arrow_beyond = ''; " +
	"$preview_labels = []; $preview_text = ''; $preview_x = null; $preview_y = null"

// dropFor escreve o `pointerup`: converte o deslocamento em QUADRADOS e
// propõe a parada.
//
// O arredondamento é para o quadrado mais próximo e não para baixo: quem solta a
// peça em cima de uma linha quis a casa que está debaixo do dedo, e `floor`
// faria o gesto cair sempre para cima e para a esquerda.
//
// Deslocamento de ZERO quadrado não propõe nada — é um clique que não andou, e
// propor ali gastaria uma parada no lugar onde a peça já está. Os sinais são
// limpos NOS DOIS caminhos, senão o `transform` fica pendurado e a peça não
// volta para o lugar.
//
// A PEÇA MARCADA move o GRUPO, e não propõe. A decisão fica AQUI, num lugar só,
// porque ela é sobre o que o gesto SIGNIFICA: arrastar a peça da vez propõe um
// movimento com custo, e arrastar uma peça marcada reposiciona o grupo. Sem
// esta linha, a peça que é as duas coisas — marcada E alvo do turno — proporia,
// e o mesmo arrasto significaria coisas diferentes conforme um estado que não
// está na ponta do dedo.
//
// MARCADA VENCE porque marcar é deliberado: ninguém marca sem querer.
func dropFor(v BoardView, who string, x, y int) string {
	stop := fmt.Sprintf("'%s/%s/parada', {payload: {from: {x: %d + dx, y: %d + dy}}}",
		v.Base, v.MoveTarget, x, y)
	destination := "@post(" + stop + ")"
	if who == "peca" && v.GM && v.MoveTarget != "" {
		group := fmt.Sprintf("@post('%s/grupo/mover', {payload: {delta: {x: dx, y: dy}, marked_tokens: $marked_tokens}})", v.Base)
		destination = fmt.Sprintf("%s ? %s : %s", markedIsToken(v.MoveTarget), group, destination)
	}
	return fmt.Sprintf(
		"if ($dragging === '%s') { "+
			"const dx = Math.round($drag_x / $square), dy = Math.round($drag_y / $square); "+
			"$dragging = ''; $drag_x = 0; $drag_y = 0; "+
			"if (dx || dy) %s }", who, destination)
}

// As variáveis do arrasto moram SÓ no `#table`, e descem por herança até quem
// está sendo arrastado.
//
// A razão é que o `data-attr:style` SUBSTITUI o atributo inteiro: pô-lo num
// elemento posicionado apaga o `--col`/`--lin` que o posicionava, e a coisa vai
// parar na quina do plano. O atributo continua lá e com cara de certo.
//
// A expressão ficou inline no `table.templ`, num lugar só, para não haver
// um segundo elemento tentado a usá-la.

// ── QUEM RECEBE O GESTO DA PEÇA, decidido em GO ──────────────────────────────
//
// As três funções abaixo existem porque **o templ não aceita `else if` numa
// lista de atributos** e não reclama: os dois ramos saem, o navegador guarda o
// primeiro e o outro morre em silêncio. A armadilha está no `engine-go/CLAUDE.md`,
// seção "templ".
//
// A escolha entre os dois gestos volta para o Go, e o elemento passa a ter UMA
// lista de atributos: exclusão por CONSTRUÇÃO.

// tokenReceivesGesture diz se ela escuta o ponteiro.
//
// O mestre entra sempre porque marcar é gesto dele: uma peça que não é alvo do
// movimento ainda pode estar num grupo marcado, e o `partyTakes` é quem checa a
// marca. Para o jogador só a peça dele responde.
func tokenReceivesGesture(v BoardView, id string) bool {
	return v.DragsToken == id || v.GM
}

// takeToken escolhe entre começar o arrasto DA PEÇA e o DO GRUPO.
func takeToken(v BoardView, id string) string {
	// No RASCUNHO toda peça se arrasta sozinha: não há grupo marcado nem alvo do
	// turno, e a única coisa que o mestre quer fazer com uma peça guardada é
	// mudá-la de lugar.
	if dragsItself(v, id) {
		return startsTheDrag(id)
	}
	return partyTakes(id)
}

// dropToken é o par do `takeToken`, e os dois têm de concordar: um
// `pointerdown` de grupo com um `pointerup` de parada proporia o movimento de
// uma peça que a pessoa nem estava movendo.
//
// As coordenadas são as DESENHADAS (`p.X`/`p.Y`), que com movimento proposto são
// o fim do caminho — é o que faz a próxima parada contar do lugar onde a peça
// está.
func dropToken(v BoardView, p boardToken) string {
	if !dragsItself(v, p.ID) {
		return dropParty(v)
	}
	if v.Draft {
		return draftMoveDrop(v, p)
	}
	return erasePreview + "; " + dropFor(v, p.ID, p.X, p.Y)
}

// draftMoveDrop põe a peça ONDE ELA FOI SOLTA, e acabou.
//
// O arrasto da mesa manda uma PARADA e o servidor devolve uma proposta com
// custo, para alguém confirmar. Aqui não há vez para gastar nem mesa para
// avisar: a peça vai para a casa e a cena guardada muda.
//
// A aritmética é a mesma do `dropFor` — o deslocamento em pixels dividido pelo
// tamanho da casa, arredondado para o quadrado mais PRÓXIMO — e ela é repetida
// em vez de extraída porque o que muda entre as duas é justamente o resto: o
// destino, o desvio para o grupo e a prévia. Um helper comum guardaria três
// linhas e faria as duas mudarem juntas no dia em que uma delas precisar de
// outro arredondamento.
func draftMoveDrop(v BoardView, p boardToken) string {
	return fmt.Sprintf(
		"if ($dragging === '%s') { "+
			"const dx = Math.round($drag_x / $square), dy = Math.round($drag_y / $square); "+
			"$dragging = ''; $drag_x = 0; $drag_y = 0; "+
			"if (dx || dy) @post('%s/pecas/%s/mover', {payload: {from: {x: %d + dx, y: %d + dy}}}) }",
		p.ID, v.Base, p.ID, p.X, p.Y)
}

// followToken é o par do `takeToken` no `pointermove`: a peça que se
// arrasta ganha a PRÉVIA, e o grupo continua só empurrando pixels.
//
// A divisa é a mesma dos outros dois, e ela tem de ser: a prévia mede o custo de
// UMA peça, e o gesto do grupo move várias sem regra de deslocamento nenhuma —
// pedir prévia ali desenharia a seta de uma peça sobre o arrasto de todas.
func followToken(v BoardView, p boardToken) string {
	// A PRÉVIA fica de fora do rascunho, e não por economia: ela pergunta ao
	// servidor quanto o caminho CUSTA, e custo de deslocamento é conta de turno.
	// Fora da sessão não há turno, então a seta desenharia um orçamento que não
	// existe — a peça só está sendo posta no lugar.
	if !dragsItself(v, p.ID) {
		return followsFinger(dragsTheParty)
	}
	if v.Draft {
		return followsFinger(p.ID)
	}
	return fingerFollowsWithPreview(v, p)
}

// sceneBoardCommand escreve a chamada de abrir ou encerrar.
//
// Irmão do `moveCommand` e separado dele de propósito: aquele leva o id
// da PEÇA no caminho, e este não tem peça nenhuma — abrir acontece justamente
// quando não há tabuleiro.
func sceneBoardCommand(v BoardView, action string) string {
	return fmt.Sprintf("@post('%s/%s')", v.Base, action)
}

// campaignCollection traduz os lugares guardados para a tela.
//
// A DATA é encurtada para o dia: o acervo responde "quando joguei isto?", e a
// hora não ajuda a escolher entre a taverna de ontem e a cripta de março. O
// formato vem do banco em ISO, e cortar no `T` é mais honesto que reformatar —
// não inventa fuso que o servidor não guardou.
func campaignCollection(places []board.Place, open []*board.BoardState) []lugarDoAcervo {
	// O índice é montado UMA vez: comparar cada linha com cada aba é a lista
	// inteira multiplicada pelo número de cenas abertas, a cada carga da página
	// e a cada quadro do stream.
	onTable := make(map[string]string, len(open))
	for _, isOpen := range open {
		onTable[isOpen.Place] = isOpen.ID
	}
	collection := make([]lugarDoAcervo, 0, len(places))
	for _, l := range places {
		collection = append(collection, lugarDoAcervo{
			ID: l.ID, Name: l.Name, Tokens: l.Tokens, When: diaDe(l.UpdatedAt),
			// Pelo NOME, que é a identidade que o `Archive` já dá ao lugar — ver
			// `placeTab`, onde o argumento inteiro está escrito.
			OpenedAt: onTable[l.Name],
		})
	}
	return collection
}

func diaDe(iso string) string {
	if day, _, found := strings.Cut(iso, "T"); found {
		return day
	}
	return iso
}

// placeCommand escreve a chamada de reabrir ou apagar um lugar do acervo.
func placeCommand(v BoardView, placeID int64, action string) string {
	return fmt.Sprintf("@post('%s/lugares/%d/%s')", v.Base, placeID, action)
}

// tabCommand escreve a troca de aba a partir do acervo.
//
// A MESMA rota que a barra de abas usa, e não uma "reabrir que só troca": o que
// se quer aqui é literalmente ir até a aba que já existe, e uma segunda porta
// para isso seria uma segunda regra sobre o que significa escolher uma cena.
func tabCommand(v BoardView, boardID string) string {
	return fmt.Sprintf("@post('%s/aba/%s')", v.Base, boardID)
}

// ── ONDE O TABULEIRO POSTA ───────────────────────────────────────────────────
//
// O mesmo gesto (pintar, pôr peça, marcar) posta em endereços diferentes
// conforme o que está sendo montado — a mesa de sábado ou o rascunho de um
// lugar do acervo. Por isso o prefixo é DADO da `BoardView` (`Base`) e não um
// literal em cada chamada: quem monta a view decide para onde os gestos dela
// vão, e nenhum desenho precisa saber que existe mais de um destino.
//
// As duas funções abaixo são os únicos lugares do pacote onde o caminho do
// tabuleiro é escrito, e é isso que o `TestNoBoardRouteIsHandwritten` varre. Elas
// são IRMÃS agora: as duas somam o sufixo a um endereço do `routes`, e nenhuma
// das duas escreve `/campanhas/` (ALE-346).

// tableBoardBase é o tabuleiro DA MESA: a cena que a sessão está jogando.
func tableBoardBase(campaignID, sessionID int64) string {
	return routes.Session(campaignID, sessionID) + "/tabuleiro"
}

// placeDraftBase é o tabuleiro do RASCUNHO: a cena que o mestre monta no acervo
// da campanha, fora da sessão.
//
// Sem sessão no caminho de propósito — o rascunho é do ACERVO e sobrevive a
// qualquer sessão.
func placeDraftBase(campaignID, placeID int64) string {
	return routes.PlaceDraft(campaignID, placeID) + "/tabuleiro"
}

// ── o PINCEL de terreno ──────────────────────────────────────────────────────
//
// Vazio é o pincel guardado, e aí o clique volta a mover a peça. É a mesma
// superfície disputada por dois gestos, e quem arbitra é o sinal.

// pickTool liga uma ferramenta, ou a DESliga se ela já estava.
//
// Clicar de novo na ferramenta ativa guarda o pincel, que é o gesto que devolve
// o clique ao movimento sem precisar de mais um botão "nenhum".
//
// UM SINAL SÓ, e o valor É a ferramenta: as quatro espécies de terreno, o
// `marcador`, e vazio para mover. Alternadores independentes deixam ligar dois
// ao mesmo tempo, e o estado impossível não estoura — ele aparece como o clique
// indo para a ferramenta errada. Aqui a exclusão fica POR CONSTRUÇÃO, e ninguém
// precisa lembrar de desligar a vizinha ao acrescentar a sexta.
func pickTool(which string) string {
	return fmt.Sprintf("$tool = ($tool === %q ? '' : %q)", which, which)
}

// MarkTool é o valor do sinal quando o clique MARCA.
//
// Constante e não string solta porque ela aparece em quatro expressões e num
// `data-show`: escrita à mão, a quinta ocorrência é a que erra a letra e vira
// uma ferramenta que a tela liga e o mapa nunca escuta.
const MarkTool = "marcador"

// NewPieceTool é o valor do sinal quando o clique CRIA uma peça avulsa.
//
// Ela é ferramenta pela divisa que o trilho já desenha — "ferramenta muda o que
// o CLIQUE faz, ação acontece uma vez e acaba" — e mesmo assim NÃO entra na
// fileira numerada: o `railKeys` tem dez dígitos e a décima primeira ferramenta
// não ganha uma letra sorteada. Ela é um modo ao lado do trilho, valor do MESMO
// sinal `$tool`, e por isso continua excluindo as outras por construção.
//
// Sem atalho de tecla, então, e de propósito. O botão é focável e é o caminho
// de teclado.
const NewPieceTool = "peca-nova"

// clickedSquareNewPiece cria a peça avulsa NA CASA CLICADA.
//
// ELE NOMEIA OS TRÊS SINAIS porque o `payload` do Datastar SUBSTITUI os sinais
// em vez de acrescentá-los: um gesto que precisa da casa E do formulário tem de
// listar o formulário à mão.
//
// O preço é uma grafia a mais de cada nome de sinal, num lugar que um `grep` de
// `$nome` não acha — e esquecer um faz a peça nascer sem aquele campo, em
// silêncio. Quem cobra é o `TestEveryPayloadKeyMatchesTheSignalItReads`: a
// chave tem de ter o nome do sinal que ela lê, então
// `new_token_name: $new_token_look` reprova.
func clickedSquareNewPiece(v BoardView) string {
	return fmt.Sprintf(
		"@post('%s/pecas/nova', {payload: {from: {x: %s, y: %s}, "+
			"new_token_name: $new_token_name, new_token_size: $new_token_size, "+
			"new_token_look: $new_token_look}})",
		v.Base, clicouEmX, clicouEmY,
	)
}

// clickedPointMarking põe um marcador na casa que o dedo acertou.
//
// Mesma aritmética da pintura — o ponto do clique dividido pelo tamanho da casa,
// mais a origem da moldura —, e ela é repetida porque o DESTINO é outro. Extrair
// a conta para um helper compartilhado economizaria uma linha e faria as duas
// rotas mudarem juntas no dia em que uma delas precisar do canto e não do centro.
func clickedPointMarking(v BoardView) string {
	return fmt.Sprintf(
		"@post('%s/marcadores/novo', {payload: {from: {x: (%s), y: (%s)}}})",
		v.Base, clicouEmX, clicouEmY,
	)
}

// markerCommand escreve o gesto sobre um marcador que já existe.
func markerCommand(v BoardView, id, action string) string {
	return fmt.Sprintf("@post('%s/marcadores/%s/%s')", v.Base, id, action)
}

// markerName é o que o leitor de tela anuncia, e ele DIZ o estado.
//
// "Marcador A em 3, 2" não conta a única coisa que o mestre precisa saber antes
// de clicar: se a mesa já está vendo aquilo. O estado entra no nome porque é
// aqui que ele muda o que a pessoa vai fazer.
func markerName(m boardMarker) string {
	state := "visível para a mesa"
	if m.Hidden {
		state = "escondido da mesa"
	}
	return fmt.Sprintf("Marcador %s em %s, %s", m.Text, m.Where, state)
}

// chosenMarker é a pergunta que mostra as ações de UM marcador.
func chosenMarker(id string) string {
	return fmt.Sprintf("$marker_chosen === %q", id)
}

// pickMarker abre as ações, ou as fecha se já estavam abertas.
//
// Clicar de novo no mesmo marcador FECHA, que é o gesto que sai de lá sem
// precisar de um botão "fechar" — o mesmo padrão do trilho de ferramentas.
// Passar "" fecha sem abrir outro, e é o que o apagar usa: as ações de um
// marcador que deixou de existir ficariam penduradas na tela até o próximo
// clique.
func pickMarker(id string) string {
	if id == "" {
		return "$marker_chosen = ''"
	}
	return fmt.Sprintf("$marker_chosen = ($marker_chosen === %q ? '' : %q)", id, id)
}

// curtainCommand escreve o gesto que fecha ou abre.
func curtainCommand(v BoardView, state string) string {
	return fmt.Sprintf("@post('%s/cortina/%s')", v.Base, state)
}

// curtainTarget é para onde o botão do cabeçalho leva.
//
// O botão ALTERNA e a tira só ABRE, e são dois destinos e não um alternar cego —
// a razão está no `runsCurtain`. Aqui é só a tradução do estado atual para o
// verbo que falta.
func curtainTarget(closed bool) string {
	if closed {
		return "abrir"
	}
	return "fechar"
}
