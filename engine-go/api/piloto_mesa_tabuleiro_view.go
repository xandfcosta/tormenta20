package api

import (
	"fmt"
	"strings"

	"t20engine/aovivo"
	"t20engine/engine"
	"t20engine/tabuleiro"
)

// O TABULEIRO como dado (ALE-263) — a fatia que se OLHA.
//
// Puro de propósito, como a `mesaView`: o handler busca, este arquivo decide, o
// template só desenha.
//
// Nenhuma regra NOVA nasce aqui. A moldura e a aparência da peça moram no
// `tabuleiro`, e o estado já chega redigido pelo `BoardForRole` — que é o mesmo
// gargalo por papel que o `StateForRole` é para a fila. Um piloto que
// reescrevesse a redação mediria a reescrita.

// tabuleiroView é o tabuleiro de uma mesa, pronto para desenhar.
type tabuleiroView struct {
	// Aberto separa "não há tabuleiro" de "há um vazio": o primeiro é a cena
	// antes de o mestre abrir, e ele NÃO desenha grade nenhuma.
	Aberto bool
	// Cortina: o tabuleiro EXISTE para o mestre e a mesa vê uma cortina no lugar
	// dele (ALE-202). É diferente de "não há tabuleiro", e as duas telas precisam
	// se parecer o MENOS possível: são estados que o jogador resolve de formas
	// diferentes — um é esperar, o outro é cutucar o mestre.
	Cortina bool
	// AvisoDaCortina é a tira que o MESTRE vê quando ela está fechada, e ela não
	// é enfeite: o mapa dele fica IGUALZINHO com a cortina aberta ou fechada, e
	// sem a tira ele narra a taverna, move o taverneiro e pergunta o que a mesa
	// faz — para uma mesa que está olhando um aviso. É a única coisa na tela dele
	// que denuncia o modo.
	AvisoDaCortina bool
	Lugar          string
	// Chao é a APARÊNCIA do lugar (pedra, taverna, cripta…), e não o terreno
	// difícil, que é regra de movimento e vive no `Dificil`. Ver GLOSSARIO.md.
	Chao string
	// Colunas e Linhas são o tamanho do plano em QUADRADOS. O pixel por
	// quadrado é do navegador, num `--quadrado` que o dedo muda.
	Colunas, Linhas int
	// X0 e Y0 são a quina da moldura no plano, e podem ser NEGATIVOS — é assim
	// que o rótulo do eixo diz o número que o servidor guarda, em vez do "+1" de
	// planilha que mentiria sobre onde a peça está.
	X0, Y0     int
	Pecas      []pecaDoTabuleiro
	Marcadores []marcadorDoTabuleiro
	// Candidatos é a fila oferecida ao diálogo "Pôr no mapa", e ela é DO MESTRE:
	// a lista traz todo combatente, inclusive o assassino que ainda não entrou em
	// cena. Preenchê-la para o jogador escreveria no HTML dele os nomes que a
	// cortina e o `hidden` existem para não contar — o vazamento não apareceria na
	// tela, só no "ver código-fonte".
	Candidatos []candidatoAoMapa
	// Terreno são os quadrados pintados, de TODAS as espécies, já em coordenada
	// da tela e cada um sabendo qual é a sua (T20 p238).
	//
	// UMA lista com a espécie dentro, e não quatro irmãs como no estado: lá elas
	// são separadas porque só o difícil alimenta o motor e a assimetria tem de
	// ficar à vista; aqui todas fazem a MESMA coisa — viram um `<div>` com uma
	// classe —, e quatro laços idênticos no templ seriam a repetição sem a razão
	// que a justifica do outro lado.
	Terreno []quadradoDeTerreno
	// Movimento é o proposto e ainda não confirmado, ou nil.
	Movimento *movimentoView
	// AlvoDoMovimento é a peça que o clique numa casa vai mover: a que já tem
	// movimento proposto, ou a que quem olha pode COMEÇAR a mover agora. Vazio
	// quando não há o que mover — e aí a camada de casas nem existe, porque um
	// alvo que não faz nada é pior que a ausência dele.
	AlvoDoMovimento string
	RotuloDoAlvo    string
	// Alcance são as casas que a peça ainda alcança, para PINTAR. Vazio quando
	// não há orçamento (o mestre, e todo mundo fora de combate): não há teto, e
	// desenhar um seria inventá-lo.
	Alcance []quadradoDoTabuleiro
	// Destino é a última parada da proposta, e ele é o que se ARRASTA enquanto
	// há uma: a peça fica onde está porque o provisório a promete, não a move.
	Destino *quadradoDoTabuleiro
	// ArrastaAPeca liga o gesto na PEÇA, e só quando não há proposta aberta.
	ArrastaAPeca string
	// CampaignID e SessionID moram aqui porque o tabuleiro escreve as próprias
	// rotas, como a `mesaView` faz com as dela.
	CampaignID, SessionID int64
	// Mestre é quem MONTA e DESMONTA a cena. Sai do mesmo `quem.Role` que a
	// redação usa, e não de um parâmetro novo: duas fontes para o papel é como
	// nasce a tela que esconde o botão de quem pode e o mostra para quem não.
	Mestre bool
	// Acervo são os LUGARES guardados da campanha (ALE-124, fatia 5). Só o
	// mestre tem — a mesa não escolhe onde joga.
	//
	// Vem no RETRATO e não por pedido sob demanda, ao contrário da SPA: lá o
	// acervo estava fora do instantâneo do socket e custava uma ida ao
	// servidor; aqui a página inteira JÁ é servida pelo servidor, e uma segunda
	// viagem para buscar o que ele já tem na mão seria inventar a latência que a
	// migração existe para tirar. O custo por tique do fluxo está medido.
	Acervo []lugarDoAcervo
}

// lugarDoAcervo é uma cena guardada, pronta para listar.
//
// A CONTAGEM de peças e não a lista: o acervo serve para escolher onde jogar, e
// mandar a cena inteira de cada lugar seria mandar a crônica toda a cada
// abertura de menu. A cena chega ao REABRIR.
type lugarDoAcervo struct {
	ID     int64
	Nome   string
	Pecas  int
	Quando string
}

// pecaDoTabuleiro é uma peça posicionada e já com a aparência resolvida.
type pecaDoTabuleiro struct {
	ID     string
	Rotulo string
	// X e Y são o lugar no PLANO: é de onde o arrasto conta o deslocamento.
	X, Y int
	// Col e Lin são o lugar DENTRO da moldura, contados de zero: é o que o CSS
	// multiplica pelo `--quadrado`. A coordenada do plano fica no `Onde`, que é
	// o que o nome acessível diz.
	Col, Lin int
	Onde     string
	Pegada   int
	// Monograma, Instancia e Matiz vêm da regra da ALE-179: a cor é da ESPÉCIE e
	// o número é da INSTÂNCIA.
	Monograma string
	Instancia string
	Matiz     int
	// NaVez acende o anel dourado, que é o MESMO sinal que a fila usa. Duas
	// cores para "a vez" fariam a mesa procurar duas coisas.
	NaVez bool
	// PV é a porcentagem restante, ou nil quando não há número para mostrar —
	// inclusive para o JOGADOR quando o mestre ocultou os PV (ALE-188). É assim
	// que a redação por papel chega até a peça.
	PV *int
	// Oculta é a peça que o mestre escondeu da mesa. Ela só existe na view dele:
	// o `BoardForRole` já a tirou da do jogador.
	Oculta bool
}

type marcadorDoTabuleiro struct {
	ID       string
	Texto    string
	Cor      string
	Col, Lin int
	Onde     string
	// Escondido só chega ao MESTRE — para a mesa o marcador escondido nem existe
	// (o `BoardForRole` o retira). Sem este campo o mestre revelava e a tela dele
	// não mudava: ele não tinha como saber o que a mesa estava vendo, que é a
	// pergunta que o gesto de revelar existe para responder.
	Escondido bool
}

// quadradoDeTerreno é uma casa pintada que sabe de que espécie é. A espécie vai
// como STRING porque o que a tela faz com ela é virar nome de classe.
type quadradoDeTerreno struct {
	quadradoDoTabuleiro
	Especie string
}

type quadradoDoTabuleiro struct {
	Col, Lin int
	// X e Y são o lugar no PLANO, e só o destino arrastável precisa deles: é de
	// onde o deslocamento do dedo é contado.
	X, Y int
}

// tabuleiroViewOf monta o tabuleiro a partir do estado JÁ REDIGIDO.
//
// A saúde chega de fora, num mapa por `entryId`, porque ela não é do tabuleiro:
// é da FILA, e o tabuleiro só a mostra. Derivá-la aqui seria a segunda conta de
// PV do app, que é como a ALE-122 começou.
func tabuleiroViewOf(b *tabuleiro.BoardState, st *aovivo.SessionRuntimeState, saude map[string]int, naVez string, quem tabuleiro.Mover, meus map[int64]bool, campaignID, sessionID int64) tabuleiroView {
	// A cena VAZIA ainda precisa saber quem olha e onde ela está: é dela que
	// sai o "Abrir tabuleiro", e um botão sem rota não é botão. A primeira
	// versão devolvia o zero e o mestre via a moldura tracejada sem gesto
	// nenhum — o mesmo estado que o jogador vê, que é justamente o que as duas
	// telas não podem ter em comum.
	if b == nil {
		return tabuleiroView{Mestre: quem.Role == "gm", CampaignID: campaignID, SessionID: sessionID}
	}
	// A CORTINA sai antes de tudo: o que chega aqui já veio vazio do
	// `BoardForRole`, sem peça, sem terreno e sem o nome do lugar — "Covil do
	// Dragão" já contaria a cena que ela existe para esconder. Montar moldura e
	// peças sobre isso desenharia uma grade vazia, que é justamente a tela do
	// "ainda não abri um tabuleiro".
	// A cortina é o que a MESA vê, e não o que o mestre vê: para ele o
	// `BoardForRole` devolveu a cena inteira, e esconder aqui tiraria o mapa de
	// quem está montando a cena. A primeira versão disto não olhava o papel e o
	// guarda acusou — "o mestre perdeu a própria cena com a cortina fechada".
	if b.Curtained && quem.Role != "gm" {
		return tabuleiroView{Aberto: true, Cortina: true, CampaignID: campaignID, SessionID: sessionID}
	}
	e := tabuleiro.MolduraDe(b)
	v := tabuleiroView{
		Aberto: true, AvisoDaCortina: b.Curtained,
		Lugar: b.Place, Chao: chaoConhecido(b.Terrain),
		Colunas: e.Colunas, Linhas: e.Linhas, X0: e.X0, Y0: e.Y0,
	}
	for i := range b.Tokens {
		v.Pecas = append(v.Pecas, pecaDoTabuleiroDe(&b.Tokens[i], e, saude, naVez))
	}
	for i := range b.Markers {
		m := &b.Markers[i]
		v.Marcadores = append(v.Marcadores, marcadorDoTabuleiro{
			ID: m.ID, Texto: m.Text, Cor: m.Color, Escondido: m.Hidden,
			Col: m.X - e.X0, Lin: m.Y - e.Y0, Onde: coordenada(m.X, m.Y),
		})
	}
	// A ORDEM do laço é a de `EspeciesDeTerreno`, então o desenho de uma casa
	// com duas espécies é sempre o mesmo — folhagens são difícil E camuflagem
	// (p267), e uma ordem que variasse faria a mesma casa mudar de cara entre
	// dois remendos.
	for _, pincel := range tabuleiro.EspeciesDeTerreno {
		for _, q := range tabuleiro.QuadradosDe(b, pincel.ID) {
			v.Terreno = append(v.Terreno, quadradoDeTerreno{
				quadradoDoTabuleiro{Col: q.X - e.X0, Lin: q.Y - e.Y0}, string(pincel.ID),
			})
		}
	}
	v.CampaignID, v.SessionID = campaignID, sessionID
	v.Mestre = quem.Role == "gm"
	if v.Mestre {
		v.Candidatos = candidatosAoMapa(b, st)
	}
	v.Movimento = movimentoDoTabuleiro(b, quem, e)
	var restante int
	v.AlvoDoMovimento, v.RotuloDoAlvo, v.Alcance, restante = oAlvoEOAlcance(b, st, quem, meus, e)
	if v.Movimento != nil && v.Movimento.Meu {
		v.Movimento.Restante = restante
	}
	v.Destino, v.ArrastaAPeca = oQueSeArrasta(b, quem, v.AlvoDoMovimento, e)
	return v
}

func pecaDoTabuleiroDe(t *tabuleiro.BoardToken, e tabuleiro.Moldura, saude map[string]int, naVez string) pecaDoTabuleiro {
	a := tabuleiro.AparenciaDe(t.Label)
	pegada := t.Footprint
	if pegada < 1 {
		pegada = 1
	}
	p := pecaDoTabuleiro{
		ID: t.ID, Rotulo: t.Label,
		Col: t.X - e.X0, Lin: t.Y - e.Y0, X: t.X, Y: t.Y, Onde: coordenada(t.X, t.Y),
		Pegada:    pegada,
		Monograma: a.Monograma, Instancia: a.Instancia, Matiz: a.Matiz,
		Oculta: t.Hidden,
	}
	if t.EntryID != nil {
		p.NaVez = naVez != "" && *t.EntryID == naVez
		if pct, ok := saude[*t.EntryID]; ok {
			p.PV = &pct
		}
	}
	return p
}

// coordenada escreve o lugar COM SINAL, que é o número que o servidor guarda.
//
// Num plano sem bordas o "+1" de planilha mente sobre onde a peça está, e é este
// texto que o leitor de tela recebe — sem ele a peça é um disco anônimo.
func coordenada(x, y int) string { return fmt.Sprintf("%d, %d", x, y) }

// saudeDaFila é quanto de PV resta a cada combatente, em porcentagem (ALE-188).
//
// Lê o estado JÁ REDIGIDO: o combatente cujo PV o mestre ocultou chega sem
// `HpMax`, não entra no mapa, e a peça dele sai sem barra. É assim que a redação
// por papel alcança o tabuleiro sem uma segunda decisão sobre quem vê o quê.
func saudeDaFila(st *aovivo.SessionRuntimeState) map[string]int {
	saude := map[string]int{}
	if st == nil {
		return saude
	}
	for i := range st.Initiative {
		e := &st.Initiative[i]
		if e.HpMax == nil || *e.HpMax <= 0 {
			continue
		}
		saude[e.ID] = mesaBarraDe(aovivo.DerefOr(e.HpCurrent, 0), *e.HpMax, false).Pct
	}
	return saude
}

// combatenteDaVez é o `entryId` de quem está na vez, ou vazio fora de combate.
// A peça acende com o MESMO dourado da linha, porque é o mesmo fato.
func combatenteDaVez(st *aovivo.SessionRuntimeState) string {
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

// nomeDaPeca é o que o leitor de tela recebe: QUEM e ONDE.
func nomeDaPeca(p pecaDoTabuleiro) string {
	nome := p.Rotulo + " em " + p.Onde
	if p.NaVez {
		nome += " — na vez"
	}
	if p.Oculta {
		nome += " — escondida da mesa"
	}
	return nome
}

// corDeMarcador traduz a cor guardada para a variável que pinta.
//
// A lista vem do `tabuleiro` e NÃO é escrita aqui, e esta função é a prova de
// por quê: ela mantinha um conjunto próprio em inglês — `gold/red/green/blue/
// violet` — enquanto a autoridade sempre aceitou `ouro/carmim/azul/verde`.
// Nenhuma das cinco casava com nenhuma das quatro, então TODO marcador do piloto
// caía no dourado, inclusive o carmim escolhido na outra tela. Nada estourava.
//
// A cor vem do banco, então é dado de cliente: fora da lista ela cai no padrão,
// porque string livre daqui iria direto para o `style`.
func corDeMarcador(c string) string {
	if tabuleiro.CorDeMarcadorConhecida(c) {
		return "var(--marcador-" + c + ")"
	}
	return "var(--marcador-" + tabuleiro.CorPadraoDeMarcador() + ")"
}

// ── o MOVIMENTO em curso (ALE-266) ───────────────────────────────────────────
//
// O movimento é uma sequência de PARADAS, e não um destino: a pessoa move a peça
// para uma casa e pode mover de novo, contornando o que quiser. Mas nada disso
// precisa de uma lista guardada em lugar nenhum — o CAMINHO PROPOSTO já é o
// acumulado, e a última parada é o último quadrado dele. Acrescentar uma parada
// é estender o caminho; e o alcance sai do fim dele com o que sobrou.
//
// Foi essa observação que apagou metade do desenho que eu ia fazer: eu ia
// guardar as paradas num sinal do cliente, com o problema de sumirem num F5.

// movimentoView é o movimento proposto, do ponto de vista de quem olha.
type movimentoView struct {
	TokenID string
	Rotulo  string
	// Trilha são as casas por onde a peça passa, já em coordenada da tela.
	Trilha []quadradoDoTabuleiro
	Custo  int
	// Orcamento -1 é "sem orçamento": o mestre move qualquer peça a qualquer
	// hora, e fora de combate cada um anda com a sua. Nesses casos não há
	// alcance para desenhar, porque não há teto que ele desenharia.
	Orcamento int
	Restante  int
	// Meu diz se quem olha decide sobre este movimento. O mestre decide por
	// qualquer um — é ele quem toca a mesa.
	Meu bool
}

// movimentoDoTabuleiro monta o movimento em curso, ou nil quando não há.
//
// O ALCANCE só é desenhado para quem PODE decidir: oferecer casas clicáveis a
// quem não vai poder confirmar é convidar para um beco.
func movimentoDoTabuleiro(b *tabuleiro.BoardState, m tabuleiro.Mover, e tabuleiro.Moldura) *movimentoView {
	if b == nil || b.Pending == nil {
		return nil
	}
	p := b.Pending
	peca := tabuleiro.FindToken(b, p.TokenID)
	if peca == nil {
		return nil
	}
	v := &movimentoView{
		TokenID: p.TokenID, Rotulo: peca.Label, Custo: p.Cost, Orcamento: p.Budget,
		Meu: m.Role == "gm" || p.ByUserID == m.UserID,
	}
	for _, q := range p.Path {
		v.Trilha = append(v.Trilha, quadradoDoTabuleiro{Col: q.X - e.X0, Lin: q.Y - e.Y0})
	}
	// O `Restante` é preenchido pelo chamador: ele sai da MESMA chamada que
	// desenha o alcance, e recalculá-lo aqui seria a segunda conta da regra.
	return v
}

// terrenoDeMovimento traduz o terreno difícil do tabuleiro para o motor.
//
// Existe porque o `moveTerrainOf` do pacote `tabuleiro` é privado, e duplicar a
// TRADUÇÃO é barato — duplicar a REGRA não seria. Se um dia ela virar três
// linhas, ela sobe para lá.
func terrenoDeMovimento(b *tabuleiro.BoardState) engine.MoveTerrain {
	if len(b.Difficult) == 0 {
		return engine.MoveTerrain{}
	}
	dificil := make(map[engine.Square]bool, len(b.Difficult))
	for _, q := range b.Difficult {
		dificil[q] = true
	}
	return engine.MoveTerrain{Difficult: dificil}
}

// pecaQueEuPossoMover é a peça que quem olha pode COMEÇAR a mover agora, ou "".
//
// Uma só, e não uma lista, porque a Mesa move uma peça por vez: com um
// movimento em curso a resposta é vazia — quem tem um proposto confirma ou
// cancela antes de pegar outra.
//
// A pergunta é respondida pelo `tabuleiro` e não aqui: quem sabe se é a vez, se
// a peça é sua e quanto sobra de deslocamento é o `PodeMover`, que é o mesmo
// `assertMovable` que a escrita usa. Perguntar de outro jeito na TELA é como
// nasce um botão que existe e o servidor recusa.
//
// O ESTADO DA SESSÃO tem de ir junto, e a primeira versão mandava nil: sem ele o
// `assertMovable` lê "fora de combate" e devolve que pode: a tela ofereceria
// mover a peça do jogador FORA DA VEZ dele, e a recusa só viria no clique.
// O RESTANTE sai daqui junto com as casas porque é a MESMA conta: `Alcance` e
// `Restante` são os dois valores que `AlcanceDaProximaParada` devolve de uma
// chamada só. A primeira versão chamava a função duas vezes com os mesmos
// argumentos, cada sítio jogando fora a metade que não usava — e duas contas da
// mesma regra é como este repositório já mostrou dois números diferentes para o
// mesmo combatente em duas telas (ALE-122).
func oAlvoEOAlcance(b *tabuleiro.BoardState, st *aovivo.SessionRuntimeState, quem tabuleiro.Mover, meus map[int64]bool, e tabuleiro.Moldura) (alvo, rotulo string, alcance []quadradoDoTabuleiro, restante int) {
	if b == nil {
		return "", "", nil, 0
	}
	// COM movimento em curso o alvo é a peça dele, e o alcance sai do fim do
	// caminho com o que sobrou. Sem, é a primeira peça que quem olha pode mover,
	// e o alcance sai de onde ela está com o orçamento inteiro.
	de := []engine.Square(nil)
	orcamento := 0
	if p := b.Pending; p != nil && (quem.Role == "gm" || p.ByUserID == quem.UserID) {
		if peca := tabuleiro.FindToken(b, p.TokenID); peca != nil {
			alvo, rotulo, de, orcamento = p.TokenID, peca.Label, p.Path, p.Budget
		}
	}
	if alvo == "" && b.Pending == nil {
		for i := range b.Tokens {
			// A POSSE é por PEÇA e não por pessoa, e a primeira versão disto
			// esqueceu: o `Mover` carrega um booleano só, e eu o deixei em falso
			// no caminho de leitura enquanto o de escrita o resolvia. O efeito era
			// o jogador NA VEZ dele não ver alcance nenhum — a tela dizia que ele
			// não podia mover a própria peça, e só o guarda acusou.
			//
			// Quem responde é o `meus`, montado contra o banco pelo `mesaRoster`:
			// a ponte até a pessoa é o DONO do personagem (ALE-33).
			dela := quem
			if id := b.Tokens[i].CharacterID; id != nil {
				dela.OwnsCharacter = meus[*id]
			}
			podeMover, orcamentoDela := tabuleiro.PodeMoverCom(b, st, b.Tokens[i].ID, dela)
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
		return "", "", nil, 0
	}
	casas, restante := engine.AlcanceDaProximaParada(de, orcamento, terrenoDeMovimento(b))
	for _, q := range casas {
		alcance = append(alcance, quadradoDoTabuleiro{Col: q.X - e.X0, Lin: q.Y - e.Y0})
	}
	return alvo, rotulo, alcance, restante
}

// comandoDoMovimento escreve a chamada de confirmar ou cancelar.
func comandoDoMovimento(v tabuleiroView, acao string) string {
	return fmt.Sprintf("@post('/piloto/mesa/%d/%d/tabuleiro/%s/%s')", v.CampaignID, v.SessionID, v.Movimento.TokenID, acao)
}

// paradaNoPontoClicado traduz o PONTO do clique em quadrado do plano.
//
// A conta é do cliente e não do servidor porque ela é sobre PIXELS: o servidor
// não sabe o zoom, que é do navegador desde que o enquadramento saiu do HTML.
// Mas ela não é REGRA — é a mesma conversão que o `squareAt` da SPA faz —, e
// tudo o que decide (o caminho, o custo, se cabe) continua do outro lado.
//
// `offsetX/offsetY` são relativos à camada, que cobre o plano inteiro; a origem
// da moldura entra somada porque o quadrado 0 da tela é o `X0` do plano, e ele
// pode ser NEGATIVO.
func paradaNoPontoClicado(v tabuleiroView) string {
	return fmt.Sprintf(
		"@post('/piloto/mesa/%d/%d/tabuleiro/%s/parada/' + (Math.floor(evt.offsetX / $quadrado) + %d) + '/' + (Math.floor(evt.offsetY / $quadrado) + %d))",
		v.CampaignID, v.SessionID, v.AlvoDoMovimento, v.X0, v.Y0,
	)
}

// ── o ARRASTO, puramente em CSS (ALE-266) ────────────────────────────────────
//
// A escolha do dono: o arrasto é VISUAL até soltar, e não toca no DOM que o
// servidor governa. Enquanto o dedo está em cima, o que muda é um `transform`
// alimentado por SINAIS; a posição de verdade só muda quando a parada é aceita.
//
// Os sinais vivem no `#mesa`, que é a única raiz que o remendo NUNCA toca desde
// que a cena virou regiões — as variáveis CSS descem por herança até a peça. Se
// morassem no plano ou na peça, o primeiro remendo de outro jogador as apagaria
// no meio do gesto, que é exatamente o defeito que o dono nomeou.
//
// O QUE SE ARRASTA depende de haver proposta, e isso não é detalhe: o provisório
// não move a peça — ele a PROMETE (é o `nextStepOrigin` da SPA). Então sem
// proposta arrasta-se a PEÇA, de onde ela está; com proposta arrasta-se o
// DESTINO, do fim da trilha. Arrastar a peça com uma proposta aberta faria o
// deslocamento contar do lugar errado, e a parada cairia longe do dedo.

// pegaParaArrastar escreve o `pointerdown`: marca quem está sendo arrastado e
// guarda o ponto de partida.
//
// NÃO chama `setPointerCapture`, e a ausência é deliberada. Quem faz o gesto
// sobreviver ao dedo sair de cima do elemento aqui é a JANELA: `segueODedo` e
// `soltaEPara` entram como `pointermove__window`/`pointerup__window`, e a
// janela recebe o evento com ou sem captura. Captura seria redundante e é a
// única chamada da expressão que LANÇA — `NotFoundError` quando o `pointerId`
// não é de um ponteiro ativo. E uma expressão Datastar que lança aborta a
// propagação DEPOIS de já ter escrito os sinais: o `pointerup` calculava o
// deslocamento certo (a URL saía `parada/4/3`) enquanto `data-class` e
// `data-attr:style` nunca reagiam — o gesto funcionava e era invisível.
func pegaParaArrastar(quem string) string {
	return fmt.Sprintf(
		"$arrastando = '%s'; $arrastoinix = evt.clientX; $arrastoiniy = evt.clientY; "+
			"$arrastox = 0; $arrastoy = 0", quem)
}

// segueODedo escreve o `pointermove`. Só mexe nos sinais se for ESTE que está
// sendo arrastado: os dois alvos escutam a mesma janela.
func segueODedo(quem string) string {
	return fmt.Sprintf(
		"$arrastando === '%s' && ($arrastox = evt.clientX - $arrastoinix, $arrastoy = evt.clientY - $arrastoiniy)", quem)
}

// soltaEPara escreve o `pointerup`: converte o deslocamento em QUADRADOS e
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
func soltaEPara(v tabuleiroView, quem string, x, y int) string {
	url := fmt.Sprintf("'/piloto/mesa/%d/%d/tabuleiro/%s/parada/' + (%d + dx) + '/' + (%d + dy)",
		v.CampaignID, v.SessionID, v.AlvoDoMovimento, x, y)
	return fmt.Sprintf(
		"if ($arrastando === '%s') { "+
			"const dx = Math.round($arrastox / $quadrado), dy = Math.round($arrastoy / $quadrado); "+
			"$arrastando = ''; $arrastox = 0; $arrastoy = 0; "+
			"if (dx || dy) @post(%s) }", quem, url)
}

// As variáveis do arrasto moram SÓ no `#mesa`, e descem por herança até quem
// está sendo arrastado.
//
// A razão é que o `data-attr:style` SUBSTITUI o atributo inteiro. Eu escrevi
// este aviso e mesmo assim pus o `data-attr:style` no destino também: o
// resultado foi o `style` dele virar só as variáveis do arrasto, apagando o
// `--col`/`--lin` que o POSICIONAVAM — o marcador ia parar na quina do plano.
// Só a medição no navegador mostrou, porque o atributo continuava lá e com cara
// de certo.
//
// A expressão ficou inline no `piloto_mesa.templ`, num lugar só, para não haver
// um segundo elemento tentado a usá-la.

// estaArrastando marca o elemento que o CSS deve deslocar.
func estaArrastando(quem string) string {
	return fmt.Sprintf("{'tabuleiro-arrastando': $arrastando === '%s'}", quem)
}

// oQueSeArrasta decide o alvo do gesto: o DESTINO quando há proposta, a PEÇA
// quando não há.
//
// Os dois nunca coexistem, e é a regra do `nextStepOrigin` que manda: a próxima
// parada conta do fim da trilha quando ela existe, e da peça quando não. Oferecer
// os dois faria o mesmo gesto contar de dois lugares.
func oQueSeArrasta(b *tabuleiro.BoardState, quem tabuleiro.Mover, alvo string, e tabuleiro.Moldura) (*quadradoDoTabuleiro, string) {
	if b == nil || alvo == "" {
		return nil, ""
	}
	if p := b.Pending; p != nil && len(p.Path) > 0 && (quem.Role == "gm" || p.ByUserID == quem.UserID) {
		fim := p.Path[len(p.Path)-1]
		return &quadradoDoTabuleiro{Col: fim.X - e.X0, Lin: fim.Y - e.Y0, X: fim.X, Y: fim.Y}, ""
	}
	return nil, alvo
}

// comandoDoTabuleiroDaCena escreve a chamada de abrir ou encerrar.
//
// Irmão do `comandoDoMovimento` e separado dele de propósito: aquele leva o id
// da PEÇA no caminho, e este não tem peça nenhuma — abrir acontece justamente
// quando não há tabuleiro.
func comandoDoTabuleiroDaCena(v tabuleiroView, acao string) string {
	return fmt.Sprintf("@post('/piloto/mesa/%d/%d/tabuleiro/%s')", v.CampaignID, v.SessionID, acao)
}

// acervoDaCampanha traduz os lugares guardados para a tela.
//
// A DATA é encurtada para o dia: o acervo responde "quando joguei isto?", e a
// hora não ajuda a escolher entre a taverna de ontem e a cripta de março. O
// formato vem do banco em ISO, e cortar no `T` é mais honesto que reformatar —
// não inventa fuso que o servidor não guardou.
func acervoDaCampanha(lugares []tabuleiro.Place) []lugarDoAcervo {
	acervo := make([]lugarDoAcervo, 0, len(lugares))
	for _, l := range lugares {
		acervo = append(acervo, lugarDoAcervo{
			ID: l.ID, Nome: l.Name, Pecas: l.Tokens, Quando: diaDe(l.UpdatedAt),
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

// comandoDoLugar escreve a chamada de reabrir ou apagar um lugar do acervo.
func comandoDoLugar(v tabuleiroView, placeID int64, acao string) string {
	return fmt.Sprintf("@post('/piloto/mesa/%d/%d/tabuleiro/lugares/%d/%s')",
		v.CampaignID, v.SessionID, placeID, acao)
}

// pecasEmPortugues concorda o número com o substantivo.
//
// Existe porque "1 peças" apareceu na tela na primeira medição, e essa é a
// classe de erro que passa por todo teste que compara com `fmt.Sprintf` do mesmo
// jeito — o teste re-derivaria o defeito. O caso do ZERO é escrito por extenso
// porque "0 peças" descreve mal o que a linha é: cena aberta e abandonada, que é
// justamente o que o mestre está procurando quando abre o acervo para limpar.
func pecasEmPortugues(n int) string {
	switch n {
	case 0:
		return "cena vazia"
	case 1:
		return "1 peça"
	default:
		return fmt.Sprintf("%d peças", n)
	}
}

// ── o PINCEL de terreno (ALE-264, item 5) ────────────────────────────────────
//
// UM SINAL para a ferramenta ativa, `pincel`, e não um por espécie. A razão veio
// da sessão da main, que estava consertando o gêmeo disto na SPA: cinco
// alternadores independentes deixam ligar dois ao mesmo tempo, e o estado
// impossível não estoura — ele aparece como o clique indo para a ferramenta
// errada. Com um sinal só, escolher uma DESescolhe as outras por construção.
//
// Vazio é o pincel guardado, e aí o clique volta a mover a peça. É a mesma
// superfície disputada por dois gestos, e quem arbitra é o sinal.

// pinturaNoPontoClicado escreve o clique que PINTA.
//
// A conta do quadrado é a mesma do `paradaNoPontoClicado` — a origem da moldura
// somada ao ponto dividido pelo lado —, e ela é do cliente pelo mesmo motivo: é
// sobre PIXELS, e o servidor não sabe o zoom.
//
// A BORRACHA não é uma espécie: ela é o `ligado=false` de qualquer uma. Por isso
// o sinal do pincel guarda a espécie e o apagar vira um segundo sinal booleano —
// uma "espécie borracha" obrigaria a decidir o que ela apaga quando a casa tem
// duas, e a resposta certa (a que estiver selecionada) já é o que isto faz.
func pinturaNoPontoClicado(v tabuleiroView) string {
	return fmt.Sprintf(
		"@post('/piloto/mesa/%d/%d/tabuleiro/terreno/' + $ferramenta + '/' + (Math.floor(evt.offsetX / $quadrado) + %d) + '/' + (Math.floor(evt.offsetY / $quadrado) + %d) + ($apagando ? '?apagar=1' : ''))",
		v.CampaignID, v.SessionID, v.X0, v.Y0,
	)
}

// escolheAFerramenta liga uma ferramenta, ou a DESliga se ela já estava.
//
// Clicar de novo na ferramenta ativa guarda o pincel, que é o gesto que devolve
// o clique ao movimento sem precisar de mais um botão "nenhum".
//
// UM SINAL SÓ, e o valor É a ferramenta: as quatro espécies de terreno, o
// `marcador`, e vazio para mover. A exclusão fica POR CONSTRUÇÃO — não há como
// duas estarem ligadas, e ninguém precisa lembrar de desligar a vizinha ao
// acrescentar a sexta. É a mesma conclusão a que a ALE-203 chegou na SPA, onde
// cinco sinais soltos deixavam pincel e régua ligados ao mesmo tempo, com um
// roubando o clique do outro.
func escolheAFerramenta(qual string) string {
	return fmt.Sprintf("$ferramenta = ($ferramenta === %q ? '' : %q)", qual, qual)
}

// FerramentaDeMarcar é o valor do sinal quando o clique MARCA.
//
// Constante e não string solta porque ela aparece em quatro expressões e num
// `data-show`: escrita à mão, a quinta ocorrência é a que erra a letra e vira
// uma ferramenta que a tela liga e o mapa nunca escuta.
const FerramentaDeMarcar = "marcador"

// marcacaoNoPontoClicado põe um marcador na casa que o dedo acertou.
//
// Mesma aritmética da pintura — o ponto do clique dividido pelo tamanho da casa,
// mais a origem da moldura —, e ela é repetida porque o DESTINO é outro. Extrair
// a conta para um helper compartilhado economizaria uma linha e faria as duas
// rotas mudarem juntas no dia em que uma delas precisar do canto e não do centro.
func marcacaoNoPontoClicado(v tabuleiroView) string {
	return fmt.Sprintf(
		"@post('/piloto/mesa/%d/%d/tabuleiro/marcadores/novo/' + (Math.floor(evt.offsetX / $quadrado) + %d) + '/' + (Math.floor(evt.offsetY / $quadrado) + %d))",
		v.CampaignID, v.SessionID, v.X0, v.Y0,
	)
}

// comandoDoMarcador escreve o gesto sobre um marcador que já existe.
func comandoDoMarcador(v tabuleiroView, id, acao string) string {
	return fmt.Sprintf("@post('/piloto/mesa/%d/%d/tabuleiro/marcadores/%s/%s')",
		v.CampaignID, v.SessionID, id, acao)
}

// nomeDoMarcador é o que o leitor de tela anuncia, e ele DIZ o estado.
//
// "Marcador A em 3, 2" não conta a única coisa que o mestre precisa saber antes
// de clicar: se a mesa já está vendo aquilo. O estado entra no nome porque é
// aqui que ele muda o que a pessoa vai fazer.
func nomeDoMarcador(m marcadorDoTabuleiro) string {
	estado := "visível para a mesa"
	if m.Escondido {
		estado = "escondido da mesa"
	}
	return fmt.Sprintf("Marcador %s em %s, %s", m.Texto, m.Onde, estado)
}

// marcadorEscolhido é a pergunta que mostra as ações de UM marcador.
func marcadorEscolhido(id string) string {
	return fmt.Sprintf("$marcadorescolhido === %q", id)
}

// escolheOMarcador abre as ações, ou as fecha se já estavam abertas.
//
// Clicar de novo no mesmo marcador FECHA, que é o gesto que sai de lá sem
// precisar de um botão "fechar" — o mesmo padrão do trilho de ferramentas.
// Passar "" fecha sem abrir outro, e é o que o apagar usa: as ações de um
// marcador que deixou de existir ficariam penduradas na tela até o próximo
// clique.
func escolheOMarcador(id string) string {
	if id == "" {
		return "$marcadorescolhido = ''"
	}
	return fmt.Sprintf("$marcadorescolhido = ($marcadorescolhido === %q ? '' : %q)", id, id)
}

// ── O ZOOM do plano (ALE-264, item 6) ────────────────────────────────────────
//
// `--quadrado` É o zoom, e isso já estava escrito no CSS antes de haver gesto: o
// plano tem `width: calc(var(--colunas) * var(--quadrado))`, e a grade, as
// peças, os marcadores e o terreno derivam do mesmo número. Mudar UM valor
// reenquadra a cena inteira, e a rolagem nativa continua valendo porque o plano
// muda de tamanho DE VERDADE — não é um `transform` que mente sobre o leiaute.
//
// E a conta do clique acompanha de graça: ela já dividia por `$quadrado`, que é
// o mesmo número. Era isso que o comentário da camada de casas prometia com "o
// `$quadrado` acompanha o zoom quando ele chegar".

// Os LIMITES são os da SPA, com as razões dela (`board-viewport.ts`): abaixo de
// 20 a peça vira um ponto e o rótulo some; acima de 96 uma tela de 1024 mostra
// 10 quadrados, menos que dois deslocamentos padrão (9m = 6 quadrados, p106), e
// o mestre deixa de ver para onde dá para andar. Portar os números em vez de
// inventá-los é o que faz as duas telas enquadrarem igual.
const (
	quadradoMinimo = 20
	quadradoMaximo = 96
	quadradoPadrao = 44
	passoDoZoom    = 8
)

// ampliaOPlano soma um passo ao zoom, preso aos limites.
func ampliaOPlano(delta int) string {
	return fmt.Sprintf("$quadrado = Math.min(%d, Math.max(%d, $quadrado + (%d)))",
		quadradoMaximo, quadradoMinimo, delta)
}

// zoomNoLimite é a pergunta que desabilita o botão que não faria nada.
func zoomNoLimite(delta int) string {
	if delta < 0 {
		return fmt.Sprintf("$quadrado <= %d", quadradoMinimo)
	}
	return fmt.Sprintf("$quadrado >= %d", quadradoMaximo)
}

// estiloDoPalco é o que leva o zoom do sinal para o CSS.
//
// Vai no PALCO e não no plano porque é lá que a variável nasce, e porque o palco
// é um nó que o remendo não substitui — o zoom sobrevive a cada mudança na cena,
// que é a mesma razão de o enquadramento não estar no HTML.
const estiloDoPalco = "'--quadrado: ' + $quadrado + 'px'"

// zoomPelaRoda: só com CTRL, e a decisão é de não tirar nada de ninguém.
//
// A roda SOZINHA continua rolando o plano, que é como se percorre o mapa hoje —
// a SPA pôde tomar a roda para o zoom porque lá não há rolagem nativa para
// perder. `Ctrl+roda` é a convenção de mapa e o gesto que o navegador já ensina.
const zoomPelaRoda = "evt.ctrlKey && (evt.preventDefault(), " +
	"$quadrado = Math.min(96, Math.max(20, $quadrado + (evt.deltaY < 0 ? 8 : -8))))"

// zoomPeloTeclado: `+` e `-`, as mesmas teclas da SPA.
//
// A guarda de alvo de digitação é a mesma do atalho da barra: sem ela, digitar
// um "-" no nome de um combatente reenquadraria o tabuleiro atrás do formulário.
const zoomPeloTeclado = `!['INPUT', 'TEXTAREA', 'SELECT'].includes(document.activeElement?.tagName) && ` +
	`(evt.key === '+' || evt.key === '=' ? ` +
	`$quadrado = Math.min(96, $quadrado + 8) : ` +
	`evt.key === '-' ? $quadrado = Math.max(20, $quadrado - 8) : null)`
