package api

import (
	"fmt"

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
	Aberto  bool
	Lugar   string
	Terreno string
	// Colunas e Linhas são o tamanho do plano em QUADRADOS. O pixel por
	// quadrado é do navegador, num `--quadrado` que o dedo muda.
	Colunas, Linhas int
	// X0 e Y0 são a quina da moldura no plano, e podem ser NEGATIVOS — é assim
	// que o rótulo do eixo diz o número que o servidor guarda, em vez do "+1" de
	// planilha que mentiria sobre onde a peça está.
	X0, Y0     int
	Pecas      []pecaDoTabuleiro
	Marcadores []marcadorDoTabuleiro
	// Dificil são os quadrados de terreno difícil (T20 p238), já em coordenada
	// da TELA.
	Dificil []quadradoDoTabuleiro
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
	// CampaignID e SessionID moram aqui porque o tabuleiro escreve as próprias
	// rotas, como a `mesaView` faz com as dela.
	CampaignID, SessionID int64
}

// pecaDoTabuleiro é uma peça posicionada e já com a aparência resolvida.
type pecaDoTabuleiro struct {
	ID     string
	Rotulo string
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
}

type quadradoDoTabuleiro struct {
	Col, Lin int
}

// tabuleiroViewOf monta o tabuleiro a partir do estado JÁ REDIGIDO.
//
// A saúde chega de fora, num mapa por `entryId`, porque ela não é do tabuleiro:
// é da FILA, e o tabuleiro só a mostra. Derivá-la aqui seria a segunda conta de
// PV do app, que é como a ALE-122 começou.
func tabuleiroViewOf(b *tabuleiro.BoardState, st *aovivo.SessionRuntimeState, saude map[string]int, naVez string, quem tabuleiro.Mover, meus map[int64]bool, campaignID, sessionID int64) tabuleiroView {
	if b == nil {
		return tabuleiroView{}
	}
	e := tabuleiro.MolduraDe(b)
	v := tabuleiroView{
		Aberto: true, Lugar: b.Place, Terreno: terrenoConhecido(b.Terrain),
		Colunas: e.Colunas, Linhas: e.Linhas, X0: e.X0, Y0: e.Y0,
	}
	for i := range b.Tokens {
		v.Pecas = append(v.Pecas, pecaDoTabuleiroDe(&b.Tokens[i], e, saude, naVez))
	}
	for i := range b.Markers {
		m := &b.Markers[i]
		v.Marcadores = append(v.Marcadores, marcadorDoTabuleiro{
			ID: m.ID, Texto: m.Text, Cor: m.Color,
			Col: m.X - e.X0, Lin: m.Y - e.Y0, Onde: coordenada(m.X, m.Y),
		})
	}
	for _, q := range b.Difficult {
		v.Dificil = append(v.Dificil, quadradoDoTabuleiro{Col: q.X - e.X0, Lin: q.Y - e.Y0})
	}
	v.CampaignID, v.SessionID = campaignID, sessionID
	v.Movimento = movimentoDoTabuleiro(b, quem, e)
	v.AlvoDoMovimento, v.RotuloDoAlvo, v.Alcance = oAlvoEOAlcance(b, st, quem, meus, e)
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
		Col: t.X - e.X0, Lin: t.Y - e.Y0, Onde: coordenada(t.X, t.Y),
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

// terrenosDoLivro são os seis chãos que a casa desenha, e o nome vira CLASSE
// (`chao-pedra`). Vindo do banco, ele é dado do cliente: um terreno inventado
// viraria uma classe que não existe e o chão sairia transparente — o que se
// parece com defeito de CSS e manda procurar no lugar errado.
var terrenosDoLivro = map[string]bool{
	"pedra": true, "taverna": true, "floresta": true,
	"ermo": true, "cripta": true, "papel": true,
}

func terrenoConhecido(t string) string {
	if terrenosDoLivro[t] {
		return t
	}
	return "pedra"
}

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

// coresDeMarcador são as que o mestre pode escolher. Vem do banco, então é dado
// do cliente: uma cor inventada iria direto para o `style`, e daí para uma
// injeção de CSS. Fora da lista, cai no dourado da casa.
var coresDeMarcador = map[string]bool{
	"gold": true, "red": true, "green": true, "blue": true, "violet": true,
}

func corDeMarcador(c string) string {
	if coresDeMarcador[c] {
		return "var(--marcador-" + c + ")"
	}
	return "var(--marcador-gold)"
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
	if v.Meu {
		_, v.Restante = engine.AlcanceDaProximaParada(p.Path, p.Budget, terrenoDeMovimento(b))
	}
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
func oAlvoEOAlcance(b *tabuleiro.BoardState, st *aovivo.SessionRuntimeState, quem tabuleiro.Mover, meus map[int64]bool, e tabuleiro.Moldura) (alvo, rotulo string, alcance []quadradoDoTabuleiro) {
	if b == nil {
		return "", "", nil
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
		return "", "", nil
	}
	casas, _ := engine.AlcanceDaProximaParada(de, orcamento, terrenoDeMovimento(b))
	for _, q := range casas {
		alcance = append(alcance, quadradoDoTabuleiro{Col: q.X - e.X0, Lin: q.Y - e.Y0})
	}
	return alvo, rotulo, alcance
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
