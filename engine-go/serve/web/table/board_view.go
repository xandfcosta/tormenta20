package table

import (
	"fmt"

	"t20engine/domain/board"
	"t20engine/domain/engine"
	"t20engine/domain/live"
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
