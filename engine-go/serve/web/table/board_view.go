package table

import (
	"fmt"
	"strings"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/serve/web/routes"
)

// O TABULEIRO como dado. Puro de propósito: o handler busca, este arquivo
// decide, o template só desenha.
//
// **Nenhuma regra nova nasce aqui.** A aparência da peça mora no `board`, e o
// estado já chega REDIGIDO pelo `BoardForRole` — reescrever a redação aqui
// mediria a reescrita.
//
// As coordenadas são ABSOLUTAS do plano, sempre. Não há moldura, e não há um
// segundo par relativo a ela: duas coordenadas para a mesma peça seriam duas
// chances de usar a errada.

// BoardView é o tabuleiro de uma mesa, pronto para desenhar.
type BoardView struct {
	// Aberto separa "não há tabuleiro" de "há um vazio": o primeiro é a cena
	// antes de o mestre abrir, e ele NÃO desenha grade nenhuma.
	Aberto bool
	// Cortina: o tabuleiro EXISTE para o mestre e a mesa vê uma cortina no lugar
	// dele. É diferente de "não há tabuleiro", e as duas telas precisam se
	// parecer o MENOS possível: são estados que o jogador resolve de formas
	// diferentes — um é esperar, o outro é cutucar o mestre.
	Cortina bool
	// AvisoDaCortina é a tira que o MESTRE vê quando ela está fechada: o mapa
	// dele fica IGUALZINHO com a cortina aberta ou fechada, e sem a tira ele
	// narra a taverna para uma mesa que está olhando um aviso. É a única coisa
	// na tela dele que denuncia o modo.
	AvisoDaCortina bool
	Lugar          string
	// Chao é a APARÊNCIA do lugar (pedra, taverna, cripta…), e não o terreno
	// difícil, que é regra de movimento e vive no `Dificil`. Ver GLOSSARY.md.
	Chao string
	// O servidor manda o que EXISTE; quem decide o que APARECE é a janela, que
	// mora no navegador ao lado do zoom. O tabuleiro é infinito para quem usa.
	Pecas      []boardToken
	Marcadores []boardMarker
	// Candidatos é a fila oferecida ao diálogo "Pôr no mapa", e ela é DO MESTRE:
	// a lista traz todo combatente, inclusive o assassino que ainda não entrou em
	// cena. Preenchê-la para o jogador escreveria no HTML dele os nomes que a
	// cortina e o `hidden` existem para não contar — o vazamento não apareceria na
	// tela, só no "ver código-fonte".
	Candidatos []candidatoAoMapa
	// Terreno são os quadrados pintados de TODAS as espécies (T20 p238), numa
	// lista só com a espécie dentro. No ESTADO elas são quatro listas separadas,
	// porque só o difícil alimenta o motor; aqui todas viram um `<div>` com uma
	// classe, e quatro laços idênticos seriam repetição sem razão.
	Terreno []terrainSquare
	// Movimento é o proposto e ainda não confirmado, ou nil.
	Movimento *moveView
	// AlvoDoMovimento é a peça que o clique numa casa vai mover: a que já tem
	// movimento proposto, ou a que quem olha pode COMEÇAR a mover agora. Vazio
	// quando não há o que mover — e aí a camada de casas nem existe, porque um
	// alvo que não faz nada é pior que a ausência dele.
	AlvoDoMovimento string
	RotuloDoAlvo    string
	// Alcance são as casas que cabem na AÇÃO DE MOVIMENTO. Vazio fora de
	// combate: sem vez não há ação, não há teto, e desenhar um seria inventá-lo.
	Alcance []boardSquare
	// AlcanceSegundo são as que só se alcançam gastando também a AÇÃO PADRÃO
	// (T20 p233), na segunda cor. Faixa própria porque a pergunta da mesa não é
	// "dá para chegar?" e sim "chegar aí custa o turno inteiro?". O que passa
	// das duas não é desenhado — não há terceira ação de movimento.
	AlcanceSegundo []boardSquare
	// Fantasma é a peça DESENHADA na origem do movimento proposto, ou nil. Ela é
	// a peça INTEIRA e não um disco genérico: com três zumbis em campo, só o
	// monograma responde qual deles está a caminho.
	Fantasma *boardToken
	// ArrastaAPeca liga o gesto na PEÇA. Com proposta aberta ela é a mesma peça,
	// desenhada no fim do caminho — ver `dropWasWhereLandsToken`.
	ArrastaAPeca string
	// CampaignID e SessionID moram aqui porque o tabuleiro escreve as próprias
	// rotas, como a `View` faz com as dela.
	CampaignID, SessionID int64
	// Base é ONDE os gestos deste tabuleiro postam, sem barra no fim: o mesmo
	// desenho serve a DUAS superfícies — a mesa jogando e o RASCUNHO de um lugar
	// do acervo. Sem ela, cada uma das vinte e cinco chamadas escolheria o
	// caminho, e a que alguém esquecesse postaria na mesa o gesto do rascunho.
	//
	// Escrita SÓ pelo `tableBoardBase` e pelo `placeDraftBase`.
	Base string
	// TabuleiroID é qual tabuleiro ESTA view desenha, e ele viaja para a tela
	// porque o COPIAR precisa registrar de onde a peça saiu.
	TabuleiroID string
	// Rascunho é a cena sendo montada NO ACERVO, fora da sessão. É um MODO
	// declarado, e não `SessionID == 0`: zero também acontece por engano, e
	// decidir por ele atenderia um estado inválido como se fosse recurso.
	//
	// O que ele governa é o que não existe fora da sessão — cortina, lente,
	// abas, encerrar, pôr na fila — mais o ARRASTO: na mesa ele PROPÕE um
	// movimento que alguém confirma; no rascunho põe a peça onde foi solta e
	// acabou, porque não há vez para gastar nem mesa para avisar.
	Rascunho bool
	// Mestre sai do mesmo `quem.Role` que a REDAÇÃO usa: duas fontes para o
	// papel é como nasce a tela que esconde o botão de quem pode e o mostra para
	// quem não.
	Mestre bool
	// Lente é o mestre vendo a cena COMO A MESA. Ela não muda o que ele PODE, só
	// o que ele VÊ — o tabuleiro chega pelo mesmo `BoardForRole` da mesa.
	Lente bool
	// PecasEscondidas responde "a emboscada está mesmo invisível?". Contar o que
	// sobrou na tela não responde: ele não sabe o que não está vendo.
	PecasEscondidas int
	// Abas são os tabuleiros ABERTOS da sessão, e a barra só existe a partir de
	// dois: com um só não há o que trocar, e a tira de fichas seria enfeite
	// ocupando mapa. Ver `tableTabs`.
	Abas []boardTab
	// Puxado é a tira "o mestre trouxe você para cá", ou nil.
	//
	// É o único aviso desta cena que fala de uma mudança que quem lê NÃO fez: a
	// cortina e a lente são modos que o dono da tela ligou. Ver `removePull`.
	Puxado *pullScreen
	// Acervo são os LUGARES guardados da campanha, e só o mestre tem — a mesa não
	// escolhe onde joga. Vem no retrato e não sob demanda: a página inteira já é
	// servida pelo servidor, e uma segunda viagem para buscar o que ele tem na
	// mão inventaria latência.
	Acervo []lugarDoAcervo
}

// lugarDoAcervo é uma cena guardada, pronta para listar. A CONTAGEM de peças e
// não a lista: mandar a cena inteira de cada lugar seria mandar a crônica toda a
// cada abertura de menu. A cena chega ao REABRIR.
type lugarDoAcervo struct {
	ID     int64
	Nome   string
	Pecas  int
	Quando string
	// AbertaEm é a aba em que este lugar JÁ ESTÁ na mesa, ou vazio: é o que faz a
	// lista distinguir o que se REABRE do que se VÊ. Sem ele, "Reabrir" a que já
	// está aberta daria duas abas da mesma cena, com duas verdades sobre onde as
	// peças estão.
	AbertaEm string
}

// boardToken é uma peça posicionada e já com a aparência resolvida.
type boardToken struct {
	ID     string
	Rotulo string
	// X e Y são o lugar no PLANO, com sinal: o CSS os multiplica pelo
	// `--quadrado` depois de descontar a janela. `Onde` é o mesmo escrito para
	// gente ler, e é o que o nome acessível diz.
	X, Y int
	Onde string
	// SaiuDe é a casa gravada enquanto há movimento proposto, e vazia quando não
	// há: o `X`/`Y` acima é onde a peça é DESENHADA — o fim do caminho —, então
	// sem ela o leitor de tela receberia a peça já na parada, como se ela
	// tivesse andado.
	//
	// TEXTO e não um segundo par de coordenadas, e é isso que a torna segura:
	// ninguém calcula com ela.
	SaiuDe string
	Pegada int
	// Monograma, Instancia e Matiz: a cor é da ESPÉCIE e o número é da INSTÂNCIA.
	Monograma string
	Instancia string
	Matiz     int
	// NaVez acende o anel dourado, o MESMO sinal que a fila usa: duas cores para
	// "a vez" fariam a mesa procurar duas coisas.
	NaVez bool
	// TemBloco: há bloco de criatura para COPIAR, então o menu pode oferecer o
	// "com bloco próprio". Falso para o herói e para o NPC digitado à mão.
	TemBloco bool
	// PV é a porcentagem restante, ou nil quando não há número para mostrar —
	// inclusive para o jogador quando o mestre ocultou os PV. É assim que a
	// redação por papel chega até a peça.
	PV *int
	// DeOndeVeio decide se o menu oferece "voltar para onde estava". Nil quando a
	// peça não foi movida nesta cena, e aí o verbo não é desenhado: botão que não
	// faz nada é pior que nenhum.
	DeOndeVeio *engine.Square
	// Oculta é a peça que o mestre escondeu da mesa. Ela só existe na view dele:
	// o `BoardForRole` já a tirou da do jogador.
	Oculta bool
	// IsObject desenha a peça QUADRADA em vez de redonda: redondo é criatura em
	// toda mesa de VTT, e uma porta redonda pede tradução.
	//
	// A FORMA e não a cor, e a segunda razão é de medição: tinta nova entra na
	// conta do medidor de contraste, e uma variante que só aparece com peça de
	// cenário no mapa nasceria sem medição.
	IsObject bool
}

type boardMarker struct {
	ID    string
	Texto string
	Cor   string
	X, Y  int
	Onde  string
	// Escondido só chega ao MESTRE — para a mesa o marcador nem existe. Sem ele o
	// mestre revelava e a tela dele não mudava, e revelar existe justamente para
	// responder "o que a mesa está vendo?".
	Escondido bool
}

// terrainSquare é uma casa pintada que sabe de que espécie é. A espécie vai
// como STRING porque o que a tela faz com ela é virar nome de classe.
type terrainSquare struct {
	boardSquare
	Especie string
}

// boardSquare é uma casa, no mesmo par de números que o servidor guarda: nada
// se traduz para desenhar, e nada se desloca quando a cena cresce.
type boardSquare struct {
	X, Y int
}

// boardViewOf monta o tabuleiro a partir do estado JÁ REDIGIDO.
//
// A saúde chega de fora, num mapa por `entryId`, porque ela não é do tabuleiro:
// é da FILA, e o tabuleiro só a mostra. Derivá-la aqui seria a segunda conta de
// PV do app.
func boardViewOf(b *board.BoardState, st *live.SessionRuntimeState, saude map[string]int, naVez string, quem board.Mover, meus map[int64]bool, campaignID, sessionID int64) BoardView {
	// A cena VAZIA ainda precisa saber quem olha e onde ela está: é dela que sai
	// o "Abrir tabuleiro", e um botão sem rota não é botão. Devolver o zero aqui
	// daria ao mestre a MESMA moldura tracejada sem gesto que o jogador vê, que
	// é justamente o que as duas telas não podem ter em comum.
	if b == nil {
		return BoardView{
			Mestre: quem.Role == "gm", CampaignID: campaignID, SessionID: sessionID,
			Base: tableBoardBase(campaignID, sessionID),
		}
	}
	// A CORTINA sai antes de tudo: o que chega aqui já veio vazio do
	// `BoardForRole`, sem peça, sem terreno e sem o nome do lugar — "Covil do
	// Dragão" já contaria a cena que ela existe para esconder. Montar moldura e
	// peças sobre isso desenharia uma grade vazia, que é justamente a tela do
	// "ainda não abri um tabuleiro".
	// A cortina é o que a MESA vê, e não o que o mestre vê: para ele o
	// `BoardForRole` devolveu a cena inteira, e esconder aqui tiraria o mapa de
	// quem está montando a cena.
	if b.Curtained && quem.Role != "gm" {
		return BoardView{
			Aberto: true, Cortina: true, CampaignID: campaignID, SessionID: sessionID,
			Base: tableBoardBase(campaignID, sessionID),
		}
	}
	v := BoardView{
		Aberto: true, AvisoDaCortina: b.Curtained,
		Lugar: b.Place, Chao: chaoConhecido(b.Terrain),
		// O ID DESTE tabuleiro vem do ESTADO e não da aba ativa: a aba é a
		// escolha de quem olha, e a fonte de qual tabuleiro está desenhado é o
		// próprio `b`. Ele é o que o COPIAR guarda na área, para o colar achar a
		// original mesmo depois de a pessoa trocar de aba.
		TabuleiroID: b.ID,
	}
	comBloco := blocosDaFila(st)
	for i := range b.Tokens {
		v.Pecas = append(v.Pecas, boardTokenOf(&b.Tokens[i], saude, comBloco, naVez))
	}
	for i := range b.Markers {
		m := &b.Markers[i]
		v.Marcadores = append(v.Marcadores, boardMarker{
			ID: m.ID, Texto: m.Text, Cor: m.Color, Escondido: m.Hidden,
			X: m.X, Y: m.Y, Onde: Coordinate(m.X, m.Y),
		})
	}
	// A ORDEM do laço é a de `TerrainKinds`, então o desenho de uma casa
	// com duas espécies é sempre o mesmo — folhagens são difícil E camuflagem
	// (p267), e uma ordem que variasse faria a mesma casa mudar de cara entre
	// dois remendos.
	for _, pincel := range board.TerrainKinds {
		for _, q := range board.SquaresOf(b, pincel.ID) {
			v.Terreno = append(v.Terreno, terrainSquare{
				boardSquare{X: q.X, Y: q.Y}, string(pincel.ID),
			})
		}
	}
	v.CampaignID, v.SessionID = campaignID, sessionID
	v.Base = tableBoardBase(campaignID, sessionID)
	v.Mestre = quem.Role == "gm"
	if v.Mestre {
		v.Candidatos = MapCandidates(b, st)
	}
	v.Movimento = moveBoard(b, quem)
	alcance := reachAndTarget(b, st, quem, meus)
	v.AlvoDoMovimento, v.RotuloDoAlvo = alcance.Alvo, alcance.Rotulo
	v.Alcance, v.AlcanceSegundo = alcance.Dentro, alcance.Segundo
	if v.Movimento != nil && v.Movimento.Meu {
		v.Movimento.Restante = alcance.Restante
	}
	// A PEÇA POUSA ONDE FOI SOLTA, e por isso ela é a única coisa que se
	// arrasta: um losango de destino, com a peça já no fim do caminho, seria um
	// segundo alvo em cima do primeiro.
	v.Fantasma = dropWasWhereLandsToken(v.Pecas, v.Movimento, v.Mestre)
	v.ArrastaAPeca = v.AlvoDoMovimento
	return v
}

// dropWasWhereLandsToken leva a peça proposta para o FIM do caminho e devolve o
// FANTASMA que fica na origem, ou nil quando não há movimento.
//
// A peça segue com UM par de coordenadas, e ele passa a ser onde ela é
// DESENHADA — guardar os dois lugares nela daria duas chances de usar o errado.
// Quem precisa da casa gravada é o fantasma, que é uma peça à parte. É também o
// que faz o deslocamento continuar contando do lugar certo sem ninguém somar
// nada: o `dropFor` recebe o `X`/`Y` desenhado.
//
// **PARA O MESTRE É O CONTRÁRIO**: a peça sólida fica onde ela realmente está e
// o fantasma vai para o fim do caminho. A inversão diz de quem é a decisão —
// quem confirma vê o mundo como ele é; quem pede vê o mundo como ele quer.
func dropWasWhereLandsToken(pecas []boardToken, mov *moveView, mestre bool) *boardToken {
	if mov == nil {
		return nil
	}
	for i := range pecas {
		if pecas[i].ID != mov.TokenID {
			continue
		}
		copia := pecas[i]
		if mestre {
			// A sólida NÃO se move; o fantasma é que vai para o destino.
			copia.X, copia.Y = mov.Fim.X, mov.Fim.Y
			copia.Onde = Coordinate(mov.Fim.X, mov.Fim.Y)
			copia.SaiuDe = pecas[i].Onde
			return &copia
		}
		pecas[i].X, pecas[i].Y = mov.Fim.X, mov.Fim.Y
		pecas[i].Onde = Coordinate(mov.Fim.X, mov.Fim.Y)
		pecas[i].SaiuDe = copia.Onde
		return &copia
	}
	return nil
}

func boardTokenOf(t *board.BoardToken, saude map[string]int, comBloco map[string]bool, naVez string) boardToken {
	a := board.AppearanceOf(t.Label)
	pegada := t.Footprint
	if pegada < 1 {
		pegada = 1
	}
	p := boardToken{
		ID: t.ID, Rotulo: t.Label,
		X: t.X, Y: t.Y, Onde: Coordinate(t.X, t.Y),
		Pegada:    pegada,
		Monograma: a.Monograma, Instancia: a.Instancia, Matiz: a.Matiz,
		Oculta:     t.Hidden,
		DeOndeVeio: t.DeOndeVeio,
		IsObject:   t.Kind == "object",
	}
	if t.EntryID != nil {
		p.NaVez = naVez != "" && *t.EntryID == naVez
		if pct, ok := saude[*t.EntryID]; ok {
			p.PV = &pct
		}
		p.TemBloco = comBloco[*t.EntryID]
	}
	return p
}

// Coordinate escreve o lugar COM SINAL, que é o número que o servidor guarda.
//
// Num plano sem bordas o "+1" de planilha mente sobre onde a peça está, e é este
// texto que o leitor de tela recebe — sem ele a peça é um disco anônimo.
func Coordinate(x, y int) string { return fmt.Sprintf("%d, %d", x, y) }

// blocosDaFila diz quais combatentes têm bloco de criatura do mestre.
//
// Mapa por `entryId` como a saúde, e pela mesma razão: não é do tabuleiro, é da
// FILA, e o tabuleiro só mostra. Ele decide se o menu da peça OFERECE o
// "com bloco próprio" — oferecer o que o servidor vai recusar é desenhar um erro,
// que é o que o `sessionConfig` já escreve com todas as letras.
func blocosDaFila(st *live.SessionRuntimeState) map[string]bool {
	comBloco := map[string]bool{}
	if st == nil {
		return comBloco
	}
	for i := range st.Initiative {
		if st.Initiative[i].CreatureID != nil {
			comBloco[st.Initiative[i].ID] = true
		}
	}
	return comBloco
}

// saudeDaFila é quanto de PV resta a cada combatente, em porcentagem.
//
// Lê o estado JÁ REDIGIDO: o combatente cujo PV o mestre ocultou chega sem
// `HpMax`, não entra no mapa, e a peça dele sai sem barra. É assim que a redação
// por papel alcança o tabuleiro sem uma segunda decisão sobre quem vê o quê.
func saudeDaFila(st *live.SessionRuntimeState) map[string]int {
	saude := map[string]int{}
	if st == nil {
		return saude
	}
	for i := range st.Initiative {
		e := &st.Initiative[i]
		if e.HpMax == nil || *e.HpMax <= 0 {
			continue
		}
		saude[e.ID] = tableBarOf(live.DerefOr(e.HpCurrent, 0), *e.HpMax, false).Pct
	}
	return saude
}

// turnCombatant é o `entryId` de quem está na vez, ou vazio fora de combate.
// A peça acende com o MESMO dourado da linha, porque é o mesmo fato.
func turnCombatant(st *live.SessionRuntimeState) string {
	if st == nil || st.TurnIndex < 0 || st.TurnIndex >= len(st.Initiative) {
		return ""
	}
	return st.Initiative[st.TurnIndex].ID
}

// posicaoNoPlano escreve o lugar da coisa em variáveis que o CSS multiplica pelo
// `--quadrado`.
//
// Posição ABSOLUTA e não grade: o `grid-column` exigiria que o plano fosse uma
// grade de N×M trilhas, e uma grade de 280 trilhas custa leiaute a cada remendo
// para colocar meia dúzia de coisas. Com absoluto, o plano é uma caixa e cada
// coisa sabe onde fica.
func posicaoNoPlano(col, lin, pegada int) string {
	return fmt.Sprintf("--col:%d; --lin:%d; --pegada:%d;", col, lin, pegada)
}

// tokenName é o que o leitor de tela recebe: QUEM e ONDE.
func tokenName(p boardToken) string {
	nome := p.Rotulo + " em " + p.Onde
	// A PARADA PROPOSTA na frase, porque a peça está desenhada nela: sem esta
	// linha o leitor de tela ouviria a peça já no destino e concluiria que o
	// movimento aconteceu.
	if p.SaiuDe != "" {
		nome += " — parada proposta, saiu de " + p.SaiuDe
	}
	if p.NaVez {
		nome += " — na vez"
	}
	if p.Oculta {
		nome += " — escondida da mesa"
	}
	return nome
}

// markerColor traduz a cor guardada para a variável que pinta.
//
// A lista de cores conhecidas vem do `board` e NÃO é escrita aqui: um conjunto
// próprio que não casasse com o da autoridade jogaria TODO marcador no padrão,
// sem estourar nada.
//
// A cor vem do banco, então é dado de cliente: fora da lista ela cai no padrão,
// porque string livre daqui iria direto para o `style`.
func markerColor(c string) string {
	if board.KnownMarkerColor(c) {
		return "var(--marcador-" + c + ")"
	}
	return "var(--marcador-" + board.DefaultMarkerColor() + ")"
}

// ── o MOVIMENTO em curso ─────────────────────────────────────────────────────
//
// O movimento é uma sequência de PARADAS, e não um destino: a pessoa move a peça
// para uma casa e pode mover de novo, contornando o que quiser. Mas nada disso
// precisa de uma lista guardada em lugar nenhum — o CAMINHO PROPOSTO já é o
// acumulado, e a última parada é o último quadrado dele. Acrescentar uma parada
// é estender o caminho; e o alcance sai do fim dele com o que sobrou.
//
// Guardar as paradas num sinal do CLIENTE era a outra opção, e elas sumiriam
// num F5.

// moveView é o movimento proposto, do ponto de vista de quem olha.
type moveView struct {
	TokenID string
	Rotulo  string
	// Trilha são as casas por onde a peça passa, já em Coordinate da tela.
	Trilha []boardSquare
	Custo  int
	// Orcamento -1 é "sem orçamento": o mestre move qualquer peça a qualquer
	// hora, e fora de combate cada um anda com a sua. Nesses casos não há
	// alcance para desenhar, porque não há teto que ele desenharia.
	Orcamento int
	Restante  int
	// Meu diz se quem olha decide sobre este movimento. O mestre decide por
	// qualquer um — é ele quem toca a mesa.
	Meu bool
	// Paradas são as casas onde a pessoa CLICOU, sem a primeira nem a última:
	// elas viram um pingo na trilha, e é ele que faz o "Desfazer parada" ter o
	// que desfazer aos olhos de quem clica. As duas pontas ficam de fora porque
	// já têm desenho próprio — a origem é o FANTASMA e o fim é a PEÇA.
	Paradas []boardSquare
	// PodeDesfazer é ter mais de UMA perna. Com uma só, desfazer é cancelar — e o
	// Cancelar está ali do lado, dizendo isso com a palavra certa.
	PodeDesfazer bool
	// Origem é a casa de onde a peça SAIU, e ela existe porque a peça deixou de
	// ficar lá: a peça é desenhada onde foi SOLTA, e quem marca o começo do
	// movimento é o fantasma nesta casa.
	Origem boardSquare
	// Fim é a casa onde a peça pousa, que é o fim do caminho. É dela que o
	// arrasto da próxima parada conta o deslocamento — a peça está lá.
	Fim boardSquare
	// Fio é o `d` da seta que liga o fantasma à peça, dobrando nas paradas. Vem
	// pronto do servidor porque o caminho é dele; ver
	// `move_drawing.go` para por que ela dobra na PARADA e não
	// em cada casa.
	Fio string
	// FioSegundo é o trecho que passa da ação de movimento e ainda cabe na ação
	// PADRÃO trocada por movimento (T20 p233): sai AZUL.
	//
	// FioAlem é o que passa das duas: sai VERMELHO, e não há terceira ação de
	// movimento no turno para pagá-lo.
	//
	// Os dois são vazios quando o caminho cabe, e também fora de combate — sem vez
	// não há ação padrão para trocar, e desenhar as faixas inventaria um teto.
	FioSegundo string
	FioAlem    string
	// Pernas são os rótulos em metros, um por trecho entre duas paradas. Eles
	// contam o CUSTO da perna e não a distância geométrica dela, para que o metro
	// do rótulo seja o mesmo metro que decide onde o `FioAlem` começa.
	Pernas []moveLeg
}

// moveBoard monta o movimento em curso, ou nil quando não há.
//
// O ALCANCE só é desenhado para quem PODE decidir: oferecer casas clicáveis a
// quem não vai poder confirmar é convidar para um beco.
func moveBoard(b *board.BoardState, m board.Mover) *moveView {
	if b == nil || b.Pending == nil {
		return nil
	}
	p := b.Pending
	peca := board.FindToken(b, p.TokenID)
	if peca == nil {
		return nil
	}
	v := &moveView{
		TokenID: p.TokenID, Rotulo: peca.Label, Custo: p.Cost, Orcamento: p.Budget,
		Meu: m.Role == "gm" || p.ByUserID == m.UserID,
	}
	for _, q := range p.Path {
		v.Trilha = append(v.Trilha, boardSquare{X: q.X, Y: q.Y})
	}
	// As duas PONTAS do caminho têm desenho de peça: o fantasma sai da origem e
	// a peça pousa no fim.
	if len(p.Path) > 0 {
		inicio, fim := p.Path[0], p.Path[len(p.Path)-1]
		v.Origem = boardSquare{X: inicio.X, Y: inicio.Y}
		v.Fim = boardSquare{X: fim.X, Y: fim.Y}
	}
	// A SETA, os RÓTULOS em metros e a divisão dourado/vermelho saem das MESMAS
	// dobras e do MESMO terreno: é o que faz o número escrito sobre a linha
	// explicar a cor dela em vez de contradizê-la.
	dobras := moveFolds(p)
	custos := legsCosts(dobras, moveTerrain(b))
	v.Fio, v.FioSegundo, v.FioAlem = moveWires(dobras, custos, p.Budget)
	v.Pernas = moveLegs(dobras, custos)
	// As paradas INTERMEDIÁRIAS: a última é onde a peça pousou e a primeira é de
	// onde ela saiu — as duas já são um disco na tela, e marcá-las de novo
	// contaria a mesma coisa duas vezes.
	if len(p.Stops) > 2 {
		v.PodeDesfazer = true
		for _, q := range p.Stops[1 : len(p.Stops)-1] {
			v.Paradas = append(v.Paradas, boardSquare{X: q.X, Y: q.Y})
		}
	}
	// O `Restante` é preenchido pelo chamador: ele sai da MESMA chamada que
	// desenha o alcance, e recalculá-lo aqui seria a segunda conta da regra.
	return v
}

// moveTerrain traduz o terreno difícil do tabuleiro para o motor.
//
// Existe porque o `moveTerrainOf` do pacote `board` é privado, e duplicar a
// TRADUÇÃO é barato — duplicar a REGRA não seria. Se um dia ela virar três
// linhas, ela sobe para lá.
func moveTerrain(b *board.BoardState) engine.MoveTerrain {
	if len(b.Difficult) == 0 {
		return engine.MoveTerrain{}
	}
	dificil := make(map[engine.Square]bool, len(b.Difficult))
	for _, q := range b.Difficult {
		dificil[q] = true
	}
	return engine.MoveTerrain{Difficult: dificil}
}

// reachAndTarget é a peça que quem olha pode COMEÇAR a mover agora, ou "".
//
// Uma só e não uma lista: a Mesa move uma peça por vez, e com um movimento em
// curso a resposta é vazia.
//
// **Quem responde é o `board.CanMove`, não esta função** — é o mesmo
// `assertMovable` que a ESCRITA usa. Perguntar de outro jeito na tela é como
// nasce um botão que existe e o servidor recusa. E o estado da sessão tem de ir
// junto: sem ele o `assertMovable` lê "fora de combate" e libera, e a tela
// ofereceria mover a peça do jogador fora da vez dele.
//
// O `Restante` sai daqui junto com as casas porque é a MESMA conta. Duas
// chamadas com os mesmos argumentos, cada uma jogando fora metade, é como este
// repositório já mostrou dois números diferentes para o mesmo combatente.
func reachAndTarget(b *board.BoardState, st *live.SessionRuntimeState, quem board.Mover, meus map[int64]bool) boardReach {
	if b == nil {
		return boardReach{}
	}
	var alvo, rotulo string
	// COM movimento em curso o alvo é a peça dele, e o alcance sai do fim do
	// caminho com o que sobrou. Sem, é a primeira peça que quem olha pode mover,
	// e o alcance sai de onde ela está com o orçamento inteiro.
	de := []engine.Square(nil)
	orcamento := 0
	if p := b.Pending; p != nil && (quem.Role == "gm" || p.ByUserID == quem.UserID) {
		if peca := board.FindToken(b, p.TokenID); peca != nil {
			alvo, rotulo, de, orcamento = p.TokenID, peca.Label, p.Path, p.Budget
		}
	}
	if alvo == "" && b.Pending == nil {
		for i := range b.Tokens {
			// A POSSE é por PEÇA e não por pessoa: o `Mover` carrega um booleano
			// só, e deixá-lo em falso aqui tira o alcance do jogador NA VEZ dele
			// — a tela diria que ele não pode mover a própria peça.
			//
			// Quem responde é o `meus`, montado contra o banco pelo `tableRoster`:
			// a ponte até a pessoa é o DONO do personagem.
			dela := quem
			if id := b.Tokens[i].CharacterID; id != nil {
				dela.OwnsCharacter = meus[*id]
			}
			podeMover, orcamentoDela := board.CanMoveWith(b, st, b.Tokens[i].ID, dela)
			if !podeMover {
				continue
			}
			alvo, rotulo = b.Tokens[i].ID, b.Tokens[i].Label
			de = []engine.Square{{X: b.Tokens[i].X, Y: b.Tokens[i].Y}}
			orcamento = orcamentoDela
			break
		}
	}
	if alvo == "" {
		return boardReach{}
	}
	dentro, segundo, restante := engine.ReachFromStops(de, orcamento, moveTerrain(b))
	return boardReach{
		Alvo: alvo, Rotulo: rotulo, Restante: restante,
		Dentro: screenSquares(dentro), Segundo: screenSquares(segundo),
	}
}

// boardReach junta o que sai de UMA pergunta: qual peça o clique move, e
// até onde ela vai com cada uma das duas ações de movimento (T20 p233).
//
// Struct e não seis valores de retorno porque as partes só fazem sentido juntas
// — o `Restante` é medido a partir do mesmo caminho que produz as faixas —, e
// porque a lista de retornos já tinha passado de quatro.
type boardReach struct {
	Alvo     string
	Rotulo   string
	Dentro   []boardSquare
	Segundo  []boardSquare
	Restante int
}

func screenSquares(casas []engine.Square) []boardSquare {
	out := make([]boardSquare, 0, len(casas))
	for _, q := range casas {
		out = append(out, boardSquare{X: q.X, Y: q.Y})
	}
	return out
}

// moveBalance diz o que SOBRA do deslocamento, ou "" quando o caminho passou.
//
// O `Restante` não serve para dizer o excesso: o `reachAndTarget` o trava em
// zero de propósito, porque ele alimenta o desenho das casas alcançáveis e
// alcance negativo não é lugar nenhum.
//
// @example moveBalance(&moveView{Custo: 4, Orcamento: 6, Restante: 2}) // "sobram 2"
func moveBalance(m *moveView) string {
	if m.Custo <= m.Orcamento {
		return fmt.Sprintf("sobram %d", m.Restante)
	}
	// PASSANDO DO DESLOCAMENTO não há saldo a dizer, e quem conta a história é a
	// LEGENDA logo abaixo. Um "além do deslocamento" mediria a mesma coisa que a
	// faixa acesa, e fica ambíguo com dois limiares — além de QUAL dos dois? O
	// metro por perna continua escrito sobre a seta, que é onde ele explica a cor.
	return ""
}

// spentActions nomeia o que o caminho CUSTA em ações do turno (T20 p233).
//
// Em AÇÕES e não em quadrados: "13 de 6" diz que estourou, não diz que estourar
// aqui é legítimo e custa o turno inteiro.
//
// A frase é a MESMA leitura das cores da seta, em palavras: quem não distingue o
// azul do vermelho no mapa lê aqui, e quem lê o mapa confirma aqui.
//
// @example spentActions(&moveView{Custo: 8, Orcamento: 6}) // "ação de movimento + ação principal"
func spentActions(m *moveView) string {
	return rangesThree[costRange(m)].Texto
}

// moveRange é uma linha da LEGENDA das cores, no rodapé do movimento.
//
// Um mapa que ensina a regra pela cor só ensina se disser o que a cor quer dizer
// — senão ele pede que a mesa adivinhe, e adivinhar cor é pior que não ter cor.
type moveRange struct {
	Classe string
	Texto  string
	// Ativa é a faixa em que o caminho INTEIRO cai, e é a que fica acesa. As
	// outras continuam na tela, apagadas: quem nunca viu o azul não descobriria
	// que ele existe se só a faixa da vez aparecesse.
	Ativa bool
}

// rangesThree são as faixas na ordem em que se gastam, e a ÚNICA lista delas.
//
// O texto daqui é o mesmo que o `spentActions` devolve, de propósito: a legenda
// e a frase do rodapé são a mesma leitura, e duas listas divergiriam no dia em
// que alguém reescrevesse uma — com a tela dizendo "gasta a ação principal" ao
// lado de uma bolinha que diz outra coisa.
var rangesThree = []moveRange{
	{Classe: "board-band-fits", Texto: "ação de movimento"},
	{Classe: "board-band-second", Texto: "ação de movimento + ação principal"},
	{Classe: "board-band-beyond", Texto: "não cabe no turno"},
}

// costRange diz em qual das três faixas o caminho INTEIRO cai (T20 p233).
//
// É o índice em `rangesThree`, e é a mesma conta que parte a seta — o que muda é
// a granularidade: a seta corta perna a perna, e isto olha o total. Por isso a
// legenda acesa concorda com a cor da PONTA da seta, que é onde o caminho acaba.
func costRange(m *moveView) int {
	switch {
	case m.Custo <= m.Orcamento:
		return 0
	case m.Custo <= 2*m.Orcamento:
		return 1
	default:
		return 2
	}
}

// moveLegend monta as três linhas, com a da vez acesa.
func moveLegend(m *moveView) []moveRange {
	ativa := costRange(m)
	legenda := make([]moveRange, 0, len(rangesThree))
	for i, f := range rangesThree {
		f.Ativa = i == ativa
		legenda = append(legenda, f)
	}
	return legenda
}

// endWireFits é a ponta da seta, ou `none` quando ela não é dele.
//
// A seta tem UMA ponta e ela vai no FIM do caminho. Havendo faixa depois desta,
// o dourado termina no MEIO do plano — no ponto em que a ação de movimento
// acabou —, e uma ponta ali apontaria para o nada e pareceria um segundo
// destino.
func endWireFits(m *moveView) string {
	return endForEnd(m, m.FioSegundo == "" && m.FioAlem == "", "move")
}

// endWireSecond e endWireBeyond completam a regra: a ponta vai em quem
// TERMINA o caminho, e cada faixa a carrega na cor dela.
func endWireSecond(m *moveView) string {
	return endForEnd(m, m.FioAlem == "", "second")
}

func endWireBeyond(m *moveView) string {
	return endForEnd(m, true, "beyond")
}

// endForEnd devolve a ponta da cor pedida, ou `none`.
//
// `none` e não atributo ausente: `marker-end` é escrito pela mesma linha do
// `.templ` nos dois casos, e o templ não aceita `else if` numa lista de
// atributos — os DOIS ramos sairiam e o navegador guardaria o primeiro.
func endForEnd(m *moveView, eOFim bool, cor string) string {
	if !eOFim {
		return "none"
	}
	return "url(#board-tip-" + cor + ")"
}

// moveCommand escreve a chamada de confirmar ou cancelar.
func moveCommand(v BoardView, acao string) string {
	return fmt.Sprintf("@post('%s/%s/%s')", v.Base, v.Movimento.TokenID, acao)
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
		v.Base, v.AlvoDoMovimento, clicouEmX, clicouEmY,
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
func startsTheDrag(quem string) string {
	return fmt.Sprintf(
		"$dragging = '%s'; $drag_start_x = evt.clientX; $drag_start_y = evt.clientY; "+
			"$drag_x = 0; $drag_y = 0", quem)
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
	return v.Rascunho || v.ArrastaAPeca == id
}

// followsFinger escreve o `pointermove`. Só mexe nos sinais se for ESTE que está
// sendo arrastado: os dois alvos escutam a mesma janela.
func followsFinger(quem string) string {
	return fmt.Sprintf(
		"$dragging === '%s' && ($drag_x = evt.clientX - $drag_start_x, $drag_y = evt.clientY - $drag_start_y)", quem)
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
func dropFor(v BoardView, quem string, x, y int) string {
	parada := fmt.Sprintf("'%s/%s/parada', {payload: {from: {x: %d + dx, y: %d + dy}}}",
		v.Base, v.AlvoDoMovimento, x, y)
	destino := "@post(" + parada + ")"
	if quem == "peca" && v.Mestre && v.AlvoDoMovimento != "" {
		grupo := fmt.Sprintf("@post('%s/grupo/mover', {payload: {delta: {x: dx, y: dy}, marked_tokens: $marked_tokens}})", v.Base)
		destino = fmt.Sprintf("%s ? %s : %s", markedIsToken(v.AlvoDoMovimento), grupo, destino)
	}
	return fmt.Sprintf(
		"if ($dragging === '%s') { "+
			"const dx = Math.round($drag_x / $square), dy = Math.round($drag_y / $square); "+
			"$dragging = ''; $drag_x = 0; $drag_y = 0; "+
			"if (dx || dy) %s }", quem, destino)
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
	return v.ArrastaAPeca == id || v.Mestre
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
	if v.Rascunho {
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
	if v.Rascunho {
		return followsFinger(p.ID)
	}
	return fingerFollowsWithPreview(v, p)
}

// sceneBoardCommand escreve a chamada de abrir ou encerrar.
//
// Irmão do `moveCommand` e separado dele de propósito: aquele leva o id
// da PEÇA no caminho, e este não tem peça nenhuma — abrir acontece justamente
// quando não há tabuleiro.
func sceneBoardCommand(v BoardView, acao string) string {
	return fmt.Sprintf("@post('%s/%s')", v.Base, acao)
}

// campaignCollection traduz os lugares guardados para a tela.
//
// A DATA é encurtada para o dia: o acervo responde "quando joguei isto?", e a
// hora não ajuda a escolher entre a taverna de ontem e a cripta de março. O
// formato vem do banco em ISO, e cortar no `T` é mais honesto que reformatar —
// não inventa fuso que o servidor não guardou.
func campaignCollection(lugares []board.Place, abertos []*board.BoardState) []lugarDoAcervo {
	// O índice é montado UMA vez: comparar cada linha com cada aba é a lista
	// inteira multiplicada pelo número de cenas abertas, a cada carga da página
	// e a cada quadro do stream.
	naMesa := make(map[string]string, len(abertos))
	for _, aberto := range abertos {
		naMesa[aberto.Place] = aberto.ID
	}
	acervo := make([]lugarDoAcervo, 0, len(lugares))
	for _, l := range lugares {
		acervo = append(acervo, lugarDoAcervo{
			ID: l.ID, Nome: l.Name, Pecas: l.Tokens, Quando: diaDe(l.UpdatedAt),
			// Pelo NOME, que é a identidade que o `Archive` já dá ao lugar — ver
			// `placeTab`, onde o argumento inteiro está escrito.
			AbertaEm: naMesa[l.Name],
		})
	}
	return acervo
}

func diaDe(iso string) string {
	if dia, _, achou := strings.Cut(iso, "T"); achou {
		return dia
	}
	return iso
}

// placeCommand escreve a chamada de reabrir ou apagar um lugar do acervo.
func placeCommand(v BoardView, placeID int64, acao string) string {
	return fmt.Sprintf("@post('%s/lugares/%d/%s')", v.Base, placeID, acao)
}

// tabCommand escreve a troca de aba a partir do acervo.
//
// A MESMA rota que a barra de abas usa, e não uma "reabrir que só troca": o que
// se quer aqui é literalmente ir até a aba que já existe, e uma segunda porta
// para isso seria uma segunda regra sobre o que significa escolher uma cena.
func tabCommand(v BoardView, tabuleiroID string) string {
	return fmt.Sprintf("@post('%s/aba/%s')", v.Base, tabuleiroID)
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
// tabuleiro é escrito, e é isso que o `TestNoBoardRouteIsHandwritten` varre.

// tableBoardBase é o tabuleiro DA MESA: a cena que a sessão está jogando.
func tableBoardBase(campaignID, sessionID int64) string {
	return fmt.Sprintf("/campanhas/%d/sessoes/%d/tabuleiro", campaignID, sessionID)
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
func pickTool(qual string) string {
	return fmt.Sprintf("$tool = ($tool === %q ? '' : %q)", qual, qual)
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
func markerCommand(v BoardView, id, acao string) string {
	return fmt.Sprintf("@post('%s/marcadores/%s/%s')", v.Base, id, acao)
}

// markerName é o que o leitor de tela anuncia, e ele DIZ o estado.
//
// "Marcador A em 3, 2" não conta a única coisa que o mestre precisa saber antes
// de clicar: se a mesa já está vendo aquilo. O estado entra no nome porque é
// aqui que ele muda o que a pessoa vai fazer.
func markerName(m boardMarker) string {
	estado := "visível para a mesa"
	if m.Escondido {
		estado = "escondido da mesa"
	}
	return fmt.Sprintf("Marcador %s em %s, %s", m.Texto, m.Onde, estado)
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
func curtainCommand(v BoardView, estado string) string {
	return fmt.Sprintf("@post('%s/cortina/%s')", v.Base, estado)
}

// curtainTarget é para onde o botão do cabeçalho leva.
//
// O botão ALTERNA e a tira só ABRE, e são dois destinos e não um alternar cego —
// a razão está no `runsCurtain`. Aqui é só a tradução do estado atual para o
// verbo que falta.
func curtainTarget(fechada bool) string {
	if fechada {
		return "abrir"
	}
	return "fechar"
}
