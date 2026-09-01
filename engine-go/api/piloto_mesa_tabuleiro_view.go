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
	// A MOLDURA SAIU na ALE-203 (decisão do dono: o tabuleiro é infinito para o
	// usuário). O servidor não tem mais `X0`, `Colunas` nem `Linhas` — ele manda
	// o que EXISTE, em coordenada ABSOLUTA do plano, e quem decide o que aparece
	// é a JANELA, que mora no navegador ao lado do zoom.
	//
	// O que isso conserta, medido: a moldura CRESCIA ao pintar perto da borda, e
	// `X0` mudava — o mesmo ponto da tela virava outro quadrado entre dois
	// cliques. Foi uma das duas causas de "apaguei e não apagou".
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
	// Alcance são as casas que a peça alcança com a AÇÃO DE MOVIMENTO, para
	// PINTAR na cor de "cabe na ação de movimento". Vazio fora de combate: sem vez não há ação de movimento, não
	// há teto, e desenhar um seria inventá-lo.
	Alcance []quadradoDoTabuleiro
	// AlcanceSegundo são as casas que ela só alcança gastando também a AÇÃO
	// PADRÃO (T20 p233), pintadas na segunda cor.
	//
	// Faixa própria e não uma lista só, porque a pergunta que a mesa faz não é
	// "dá para chegar?" e sim "chegar aí me custa o turno inteiro?" — e essa é a
	// diferença entre as duas cores. O que passa das duas ações não é desenhado:
	// não há terceira ação de movimento para gastar.
	AlcanceSegundo []quadradoDoTabuleiro
	// Fantasma é a peça DESENHADA na origem do movimento proposto, ou nil.
	//
	// Ela é a peça inteira e não um marcador genérico porque é o monograma e o
	// selo que dizem QUEM saiu dali: com três zumbis em campo, um disco vazio na
	// casa não responde qual deles está a caminho.
	Fantasma *pecaDoTabuleiro
	// ArrastaAPeca liga o gesto na PEÇA. Com proposta aberta ela é a mesma peça,
	// desenhada no fim do caminho — ver `aPecaPousaOndeFoiSolta`.
	ArrastaAPeca string
	// CampaignID e SessionID moram aqui porque o tabuleiro escreve as próprias
	// rotas, como a `mesaView` faz com as dela.
	CampaignID, SessionID int64
	// Mestre é quem MONTA e DESMONTA a cena. Sai do mesmo `quem.Role` que a
	// redação usa, e não de um parâmetro novo: duas fontes para o papel é como
	// nasce a tela que esconde o botão de quem pode e o mostra para quem não.
	Mestre bool
	// Lente é o mestre vendo a cena COMO A MESA (ALE-193). Ela não muda o que ele
	// PODE — os controles continuam dele —, só o que ele VÊ: o tabuleiro chega
	// redigido pelo mesmo `BoardForRole` que a mesa recebe.
	Lente bool
	// PecasEscondidas é quantas peças a mesa NÃO vê, e é a pergunta que trouxe o
	// mestre até aqui ("a emboscada está mesmo invisível?"). Contar o que sobrou
	// na tela não responderia: ele não sabe o que não está vendo.
	PecasEscondidas int
	// Abas são os tabuleiros ABERTOS da sessão (ALE-205), e a barra só existe a
	// partir de dois: com um só não há o que trocar, e a tira de fichas seria
	// enfeite ocupando mapa. Ver `asAbasDaMesa`.
	Abas []abaDoTabuleiro
	// Puxado é a tira "o mestre trouxe você para cá", ou nil (ALE-205, fatia 2).
	//
	// É o único aviso desta cena que fala de uma mudança que quem lê NÃO fez: a
	// cortina e a lente são modos que o dono da tela ligou. Ver `aTiraDoPuxao`.
	Puxado *oPuxaoNaTela
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
	// AbertaEm é a aba em que este lugar JÁ ESTÁ na mesa, ou vazio (ALE-205,
	// fatia 3). É o que faz a lista distinguir o que se REABRE do que se VÊ.
	//
	// Sem ele o mestre não tinha como saber qual das 148 linhas é a cena que está
	// na tela dele agora — o papercut que o dono levantou —, e "Reabrir" a que já
	// está aberta abriria uma segunda aba da mesma cena, com duas verdades sobre
	// onde as peças estão.
	AbertaEm string
}

// pecaDoTabuleiro é uma peça posicionada e já com a aparência resolvida.
type pecaDoTabuleiro struct {
	ID     string
	Rotulo string
	// X e Y são o lugar no PLANO, com sinal: é de onde o arrasto conta o
	// deslocamento, e o CSS os multiplica pelo `--quadrado` depois de descontar a
	// janela. Havia um segundo par (`Col`/`Lin`) relativo à moldura, e ele saiu
	// com ela na ALE-203 — duas coordenadas para a mesma peça eram duas chances
	// de usar a errada.
	//
	// O `Onde` é a mesma coisa escrita para gente ler, e é o que o nome acessível
	// diz.
	X, Y int
	Onde string
	// SaiuDe é a casa GRAVADA da peça enquanto há movimento proposto, escrita
	// para gente ler, e vazia quando não há. Ela existe porque desde a ALE-203
	// (item 4) o `X`/`Y` acima é onde a peça é DESENHADA — o fim do caminho —, e
	// sem esta linha o leitor de tela perderia de onde a peça saiu: ele receberia
	// a peça já na parada proposta, como se ela tivesse andado.
	//
	// Texto e não um segundo par de coordenadas, e a diferença é o que a torna
	// segura: ninguém calcula com ela. O par `Col`/`Lin` saiu com a moldura
	// justamente por ser a segunda chance de usar a coordenada errada.
	SaiuDe string
	Pegada int
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
	// DeOndeVeio é onde ela estava antes do último pouso, e é o que decide se o
	// menu oferece "voltar para onde estava" (ALE-206). Nil quando ela não foi
	// movida nesta cena — e aí o verbo não é desenhado, porque um botão que não
	// faz nada é pior que nenhum.
	DeOndeVeio *engine.Square
	// Oculta é a peça que o mestre escondeu da mesa. Ela só existe na view dele:
	// o `BoardForRole` já a tirou da do jogador.
	Oculta bool
}

type marcadorDoTabuleiro struct {
	ID    string
	Texto string
	Cor   string
	// X e Y são a casa no PLANO, em coordenada absoluta — como todo o resto do
	// tabuleiro desde a ALE-203. Eram `Col`/`Lin`, relativos à moldura, e a
	// moldura saiu.
	X, Y int
	Onde string
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

// quadradoDoTabuleiro é uma casa, em coordenada ABSOLUTA do plano.
//
// Ela tinha `Col`/`Lin` — o lugar dentro da moldura — e a moldura saiu na
// ALE-203. Agora há um par de números só, e ele é o mesmo que o servidor guarda:
// nada precisa ser traduzido para desenhar, e nada se desloca quando a cena
// cresce.
type quadradoDoTabuleiro struct {
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
	v := tabuleiroView{
		Aberto: true, AvisoDaCortina: b.Curtained,
		Lugar: b.Place, Chao: chaoConhecido(b.Terrain),
	}
	for i := range b.Tokens {
		v.Pecas = append(v.Pecas, pecaDoTabuleiroDe(&b.Tokens[i], saude, naVez))
	}
	for i := range b.Markers {
		m := &b.Markers[i]
		v.Marcadores = append(v.Marcadores, marcadorDoTabuleiro{
			ID: m.ID, Texto: m.Text, Cor: m.Color, Escondido: m.Hidden,
			X: m.X, Y: m.Y, Onde: coordenada(m.X, m.Y),
		})
	}
	// A ORDEM do laço é a de `EspeciesDeTerreno`, então o desenho de uma casa
	// com duas espécies é sempre o mesmo — folhagens são difícil E camuflagem
	// (p267), e uma ordem que variasse faria a mesma casa mudar de cara entre
	// dois remendos.
	for _, pincel := range tabuleiro.EspeciesDeTerreno {
		for _, q := range tabuleiro.QuadradosDe(b, pincel.ID) {
			v.Terreno = append(v.Terreno, quadradoDeTerreno{
				quadradoDoTabuleiro{X: q.X, Y: q.Y}, string(pincel.ID),
			})
		}
	}
	v.CampaignID, v.SessionID = campaignID, sessionID
	v.Mestre = quem.Role == "gm"
	if v.Mestre {
		v.Candidatos = candidatosAoMapa(b, st)
	}
	v.Movimento = movimentoDoTabuleiro(b, quem)
	alcance := oAlvoEOAlcance(b, st, quem, meus)
	v.AlvoDoMovimento, v.RotuloDoAlvo = alcance.Alvo, alcance.Rotulo
	v.Alcance, v.AlcanceSegundo = alcance.Dentro, alcance.Segundo
	if v.Movimento != nil && v.Movimento.Meu {
		v.Movimento.Restante = alcance.Restante
	}
	// A PEÇA POUSA ONDE FOI SOLTA (ALE-203, item 4) e por isso ela é a única
	// coisa que se arrasta: o losango de destino sumiu junto, porque com a peça
	// no fim do caminho ele era um segundo alvo em cima do primeiro.
	v.Fantasma = aPecaPousaOndeFoiSolta(v.Pecas, v.Movimento, v.Mestre)
	v.ArrastaAPeca = v.AlvoDoMovimento
	return v
}

// aPecaPousaOndeFoiSolta leva a peça proposta para o FIM do caminho e devolve o
// FANTASMA que fica na origem, ou nil quando não há movimento.
//
// As palavras do dono: *"ao soltar a peça, ela vai ser renderizada no lugar que
// foi solta e o início mostra a peça transparente para marcar o início do
// movimento."* Antes disto a peça voltava para o começo ao soltar, e o que
// marcava o destino era um losango — o gesto acabava desfazendo a si mesmo aos
// olhos de quem arrastou.
//
// A peça continua com UM par de coordenadas, e ele passa a ser onde ela é
// DESENHADA. Guardar os dois lugares nela seria refazer o par `Col`/`Lin` que
// saiu com a moldura, e pela mesma razão: quem lê escolhe o errado. Quem precisa
// da casa gravada é o fantasma, e ele é uma peça à parte.
//
// O que se ARRASTA é a peça no fim do caminho, e é por isso que o deslocamento
// continua contando do lugar certo sem ninguém somar nada: o `soltaEPara` recebe
// o `X`/`Y` desenhado.
//
// PARA O MESTRE É O CONTRÁRIO, e é decisão do dono: a peça SÓLIDA fica onde ela
// realmente está e o FANTASMA vai para o fim do caminho. A inversão diz de quem
// é a decisão — o jogador está mostrando onde ele QUER estar, e para ele o
// destino é o fato; o mestre está olhando a cena que ele ainda não mudou, e para
// ele o fato é onde a peça está. Quem confirma vê o mundo como ele é; quem pede
// vê o mundo como ele quer.
func aPecaPousaOndeFoiSolta(pecas []pecaDoTabuleiro, mov *movimentoView, mestre bool) *pecaDoTabuleiro {
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
			copia.Onde = coordenada(mov.Fim.X, mov.Fim.Y)
			copia.SaiuDe = pecas[i].Onde
			return &copia
		}
		pecas[i].X, pecas[i].Y = mov.Fim.X, mov.Fim.Y
		pecas[i].Onde = coordenada(mov.Fim.X, mov.Fim.Y)
		pecas[i].SaiuDe = copia.Onde
		return &copia
	}
	return nil
}

func pecaDoTabuleiroDe(t *tabuleiro.BoardToken, saude map[string]int, naVez string) pecaDoTabuleiro {
	a := tabuleiro.AparenciaDe(t.Label)
	pegada := t.Footprint
	if pegada < 1 {
		pegada = 1
	}
	p := pecaDoTabuleiro{
		ID: t.ID, Rotulo: t.Label,
		X: t.X, Y: t.Y, Onde: coordenada(t.X, t.Y),
		Pegada:    pegada,
		Monograma: a.Monograma, Instancia: a.Instancia, Matiz: a.Matiz,
		Oculta:     t.Hidden,
		DeOndeVeio: t.DeOndeVeio,
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
	// A PARADA PROPOSTA na frase, porque a peça está desenhada nela: sem esta
	// linha o leitor de tela ouviria a peça já no destino e concluiria que o
	// movimento aconteceu (ALE-203, item 4).
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
	// Paradas são as casas onde a pessoa CLICOU, sem a primeira nem a última:
	// elas viram um pingo na trilha, e é ele que faz o "Desfazer parada" ter o
	// que desfazer aos olhos de quem clica. As duas pontas ficam de fora porque
	// já têm desenho próprio — a origem é o FANTASMA e o fim é a PEÇA.
	Paradas []quadradoDoTabuleiro
	// PodeDesfazer é ter mais de UMA perna. Com uma só, desfazer é cancelar — e o
	// Cancelar está ali do lado, dizendo isso com a palavra certa.
	PodeDesfazer bool
	// Origem é a casa de onde a peça SAIU, e ela existe porque a peça deixou de
	// ficar lá: desde a ALE-203 (item 4) a peça é desenhada onde foi SOLTA, e
	// quem marca o começo do movimento é o fantasma nesta casa.
	Origem quadradoDoTabuleiro
	// Fim é a casa onde a peça pousa, que é o fim do caminho. É dela que o
	// arrasto da próxima parada conta o deslocamento — a peça está lá.
	Fim quadradoDoTabuleiro
	// Fio é o `d` da seta que liga o fantasma à peça, dobrando nas paradas. Vem
	// pronto do servidor porque o caminho é dele; ver
	// `piloto_mesa_movimento_desenho.go` para por que ela dobra na PARADA e não
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
	Pernas []pernaDoMovimento
}

// movimentoDoTabuleiro monta o movimento em curso, ou nil quando não há.
//
// O ALCANCE só é desenhado para quem PODE decidir: oferecer casas clicáveis a
// quem não vai poder confirmar é convidar para um beco.
func movimentoDoTabuleiro(b *tabuleiro.BoardState, m tabuleiro.Mover) *movimentoView {
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
		v.Trilha = append(v.Trilha, quadradoDoTabuleiro{X: q.X, Y: q.Y})
	}
	// As duas PONTAS do caminho, que desde a ALE-203 (item 4) têm desenho de
	// peça: o fantasma sai da origem e a peça pousa no fim.
	if len(p.Path) > 0 {
		inicio, fim := p.Path[0], p.Path[len(p.Path)-1]
		v.Origem = quadradoDoTabuleiro{X: inicio.X, Y: inicio.Y}
		v.Fim = quadradoDoTabuleiro{X: fim.X, Y: fim.Y}
	}
	// A SETA, os RÓTULOS em metros e a divisão dourado/vermelho saem das MESMAS
	// dobras e do MESMO terreno (ALE-203, item 13): é o que faz o número escrito
	// sobre a linha explicar a cor dela em vez de contradizê-la.
	dobras := asDobrasDoMovimento(p)
	custos := osCustosDasPernas(dobras, terrenoDeMovimento(b))
	v.Fio, v.FioSegundo, v.FioAlem = osFiosDoMovimento(dobras, custos, p.Budget)
	v.Pernas = asPernasDoMovimento(dobras, custos)
	// As paradas INTERMEDIÁRIAS: a última é onde a peça pousou e a primeira é de
	// onde ela saiu — as duas já são um disco na tela, e marcá-las de novo
	// contaria a mesma coisa duas vezes.
	if len(p.Stops) > 2 {
		v.PodeDesfazer = true
		for _, q := range p.Stops[1 : len(p.Stops)-1] {
			v.Paradas = append(v.Paradas, quadradoDoTabuleiro{X: q.X, Y: q.Y})
		}
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
func oAlvoEOAlcance(b *tabuleiro.BoardState, st *aovivo.SessionRuntimeState, quem tabuleiro.Mover, meus map[int64]bool) alcanceDoTabuleiro {
	if b == nil {
		return alcanceDoTabuleiro{}
	}
	var alvo, rotulo string
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
		return alcanceDoTabuleiro{}
	}
	dentro, segundo, restante := engine.AlcanceDaProximaParada(de, orcamento, terrenoDeMovimento(b))
	return alcanceDoTabuleiro{
		Alvo: alvo, Rotulo: rotulo, Restante: restante,
		Dentro: emCasasDaTela(dentro), Segundo: emCasasDaTela(segundo),
	}
}

// alcanceDoTabuleiro junta o que sai de UMA pergunta: qual peça o clique move, e
// até onde ela vai com cada uma das duas ações de movimento (T20 p233).
//
// Struct e não seis valores de retorno porque as partes só fazem sentido juntas
// — o `Restante` é medido a partir do mesmo caminho que produz as faixas —, e
// porque a lista de retornos já tinha passado de quatro.
type alcanceDoTabuleiro struct {
	Alvo     string
	Rotulo   string
	Dentro   []quadradoDoTabuleiro
	Segundo  []quadradoDoTabuleiro
	Restante int
}

func emCasasDaTela(casas []engine.Square) []quadradoDoTabuleiro {
	out := make([]quadradoDoTabuleiro, 0, len(casas))
	for _, q := range casas {
		out = append(out, quadradoDoTabuleiro{X: q.X, Y: q.Y})
	}
	return out
}

// oSaldoDoMovimento diz o que SOBRA — ou o que passou (ALE-203, item 13).
//
// O rodapé dizia "sobram %d" sempre, e desde que o caminho caro passou a ser
// aceito esse número parava em ZERO: o `AlcanceDaProximaParada` trava o restante
// em zero de propósito, porque ele alimenta o desenho das casas alcançáveis e
// alcance negativo não é lugar nenhum. O efeito era um rodapé que dizia "sobram
// 0" para um caminho que passou três quadrados do deslocamento — verdade sobre
// o alcance, silêncio sobre o excesso, e a pessoa sem saber quanto encurtar.
//
// Em METROS, e não em quadrados como o resto da frase: este número é a leitura
// do trecho VERMELHO da seta, e é ali que a pessoa vai conferi-lo. Duas unidades
// para a mesma grandeza obrigariam a converter de cabeça justamente na conta que
// decide o próximo clique.
//
// @example oSaldoDoMovimento(&movimentoView{Custo: 8, Orcamento: 6}) // "ação de movimento + ação principal"
func oSaldoDoMovimento(m *movimentoView) string {
	if m.Custo <= m.Orcamento {
		return fmt.Sprintf("sobram %d", m.Restante)
	}
	// PASSANDO DO DESLOCAMENTO não há saldo a dizer, e quem conta a história é a
	// LEGENDA logo abaixo. "15,0m além do deslocamento" media a mesma coisa que a
	// faixa acesa, e ainda ficou ambígua quando surgiu o segundo limiar — além de
	// QUAL dos dois? O metro por perna continua escrito sobre a seta, que é onde
	// ele explica a cor.
	return ""
}

// asAcoesGastas nomeia o que o caminho CUSTA em ações do turno (T20 p233).
//
// É a pergunta que o dono pôs em primeiro lugar — *"eles poderão ver o quanto a
// peça deles pode mover e se eles precisam gastar a ação de movimento e a ação
// principal"* — e ela não se responde com quadrados: "13 de 6" diz que estourou,
// não diz que estourar aqui é legítimo e custa o turno inteiro.
//
// A frase é a MESMA leitura das cores da seta, em palavras: quem não distingue o
// azul do vermelho no mapa lê aqui, e quem lê o mapa confirma aqui.
//
// @example asAcoesGastas(&movimentoView{Custo: 8, Orcamento: 6}) // "ação de movimento + ação principal"
func asAcoesGastas(m *movimentoView) string {
	return asTresFaixas[aFaixaDoCusto(m)].Texto
}

// faixaDoMovimento é uma linha da LEGENDA das cores, no rodapé do movimento.
//
// Ela existe porque as três cores não se explicavam: o dono reparou que "não tem
// um modo do usuário saber o que as cores indicam". Um mapa que ensina a regra
// pela cor só ensina se disser o que a cor quer dizer — senão ele pede que a
// mesa adivinhe, e adivinhar cor é pior que não ter cor.
type faixaDoMovimento struct {
	Classe string
	Texto  string
	// Ativa é a faixa em que o caminho INTEIRO cai, e é a que fica acesa. As
	// outras continuam na tela, apagadas: quem nunca viu o azul não descobriria
	// que ele existe se só a faixa da vez aparecesse.
	Ativa bool
}

// asTresFaixas são as faixas na ordem em que se gastam, e a ÚNICA lista delas.
//
// O texto daqui é o mesmo que o `asAcoesGastas` devolve, de propósito: a legenda
// e a frase do rodapé são a mesma leitura, e duas listas divergiriam no dia em
// que alguém reescrevesse uma — com a tela dizendo "gasta a ação principal" ao
// lado de uma bolinha que diz outra coisa.
var asTresFaixas = []faixaDoMovimento{
	{Classe: "tabuleiro-faixa-cabe", Texto: "ação de movimento"},
	{Classe: "tabuleiro-faixa-segundo", Texto: "ação de movimento + ação principal"},
	{Classe: "tabuleiro-faixa-alem", Texto: "não cabe no turno"},
}

// aFaixaDoCusto diz em qual das três faixas o caminho INTEIRO cai (T20 p233).
//
// É o índice em `asTresFaixas`, e é a mesma conta que parte a seta — o que muda é
// a granularidade: a seta corta perna a perna, e isto olha o total. Por isso a
// legenda acesa concorda com a cor da PONTA da seta, que é onde o caminho acaba.
func aFaixaDoCusto(m *movimentoView) int {
	switch {
	case m.Custo <= m.Orcamento:
		return 0
	case m.Custo <= 2*m.Orcamento:
		return 1
	default:
		return 2
	}
}

// aLegendaDoMovimento monta as três linhas, com a da vez acesa.
func aLegendaDoMovimento(m *movimentoView) []faixaDoMovimento {
	ativa := aFaixaDoCusto(m)
	legenda := make([]faixaDoMovimento, 0, len(asTresFaixas))
	for i, f := range asTresFaixas {
		f.Ativa = i == ativa
		legenda = append(legenda, f)
	}
	return legenda
}

// aPontaDoFioQueCabe é a ponta da seta, ou `none` quando ela não é dele.
//
// A seta tem UMA ponta e ela vai no FIM do caminho. Havendo faixa depois desta,
// o dourado termina no MEIO do plano — no ponto em que a ação de movimento
// acabou —, e uma ponta ali apontaria para o nada e pareceria um segundo
// destino.
func aPontaDoFioQueCabe(m *movimentoView) string {
	return aPontaSeForOFim(m, m.FioSegundo == "" && m.FioAlem == "", "movimento")
}

// aPontaDoFioSegundo e aPontaDoFioAlem completam a regra: a ponta vai em quem
// TERMINA o caminho, e cada faixa a carrega na cor dela.
func aPontaDoFioSegundo(m *movimentoView) string {
	return aPontaSeForOFim(m, m.FioAlem == "", "segundo")
}

func aPontaDoFioAlem(m *movimentoView) string {
	return aPontaSeForOFim(m, true, "alem")
}

// aPontaSeForOFim devolve a ponta da cor pedida, ou `none`.
//
// `none` e não atributo ausente: `marker-end` é escrito pela mesma linha do
// `.templ` nos dois casos, e o templ não aceita `else if` numa lista de
// atributos — os DOIS ramos sairiam e o navegador guardaria o primeiro.
func aPontaSeForOFim(m *movimentoView, eOFim bool, cor string) string {
	if !eOFim {
		return "none"
	}
	return "url(#tabuleiro-ponta-do-" + cor + ")"
}

// comandoDoMovimento escreve a chamada de confirmar ou cancelar.
func comandoDoMovimento(v tabuleiroView, acao string) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/%s/%s')", v.CampaignID, v.SessionID, v.Movimento.TokenID, acao)
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
		"@post('/mesa/%d/%d/tabuleiro/%s/parada/' + (%s) + '/' + (%s))",
		v.CampaignID, v.SessionID, v.AlvoDoMovimento, clicouEmX, clicouEmY,
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
// O QUE SE ARRASTA é sempre a PEÇA, e ela conta do lugar onde está DESENHADA.
// Havia um segundo alvo — o losango do destino —, porque a peça voltava para a
// origem ao soltar e o fim da trilha precisava de alguém que o representasse.
// Com a peça pousando onde foi solta (ALE-203, item 4) o losango virou um alvo
// em cima do outro, e a regra do `nextStepOrigin` passou a se cumprir sozinha: a
// próxima parada conta do fim da trilha porque é lá que a peça está.

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

// oDedoSegueComPrevia é o `segueODedo` da PEÇA, com a seta viva por cima.
//
// Ele pede a prévia ao servidor SÓ QUANDO O QUADRADO MUDA, e não a cada pixel:
// é a mesma trava do `aReguaSegueOPonteiro`, e ela transforma "um pedido por
// evento de ponteiro" em "um pedido por casa atravessada". Sem ela, um arrasto
// de dois segundos abriria centenas de requisições para desenhar a mesma linha.
//
// A conta do quadrado é a MESMA do `soltaEPara` (`Math.round` do deslocamento
// pelo `--quadrado`), e tem de ser: se a prévia arredondasse diferente do
// soltar, a pessoa leria um custo e receberia outro — o defeito mais caro que
// esta tela pode ter, porque ele só aparece depois da decisão.
func oDedoSegueComPrevia(v tabuleiroView, p pecaDoTabuleiro) string {
	return fmt.Sprintf(
		"if ($arrastando !== 'peca') return; "+
			"$arrastox = evt.clientX - $arrastoinix; $arrastoy = evt.clientY - $arrastoiniy; "+
			"const cx = %d + Math.round($arrastox / $quadrado), cy = %d + Math.round($arrastoy / $quadrado); "+
			"if (cx === $previax && cy === $previay) return; "+
			"$previax = cx; $previay = cy; "+
			"@post('/mesa/%d/%d/tabuleiro/%s/previa/' + cx + '/' + cy)",
		p.X, p.Y, v.CampaignID, v.SessionID, p.ID)
}

// apagaAPrevia limpa a seta viva. Vai no `pointerup`, junto do que solta.
//
// QUEM LIMPA É QUEM TERMINA O GESTO, e não quem começa o próximo: um desenho de
// prévia que sobrevivesse ao soltar ficaria por cima da seta de verdade, com o
// mesmo formato e outra medida — dois caminhos na tela e nenhum jeito de saber
// qual é o que vale. É a mesma regra do nó compartilhado que o diálogo de senha
// ensinou (ver o CLAUDE.md deste pacote).
//
// O `$previax` volta para um valor IMPOSSÍVEL e não para zero: zero é uma casa
// legítima do plano, e o próximo arrasto que começasse nela não pediria prévia
// nenhuma — a trava do "só quando o quadrado muda" o engoliria em silêncio.
const apagaAPrevia = "$previafiocabe = ''; $previafiosegundo = ''; $previafioalem = ''; " +
	"$previarotulos = []; $previatexto = ''; $previax = null; $previay = null"

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
// A PEÇA MARCADA move o GRUPO, e não propõe (ALE-203, item 10).
//
// A decisão fica AQUI, num lugar só, porque ela é sobre o que o gesto SIGNIFICA:
// arrastar a peça da vez propõe um movimento com custo, e arrastar uma peça
// marcada reposiciona o grupo. Sem esta linha, a peça que é as duas coisas —
// marcada E alvo do turno — proporia, e o mesmo arrasto significaria coisas
// diferentes conforme um estado que não está na ponta do dedo. Medido no
// navegador: marcar o Bandido e arrastá-lo abria a barra de "14 quadrados".
//
// MARCADA VENCE porque marcar é deliberado: ninguém marca sem querer.
func soltaEPara(v tabuleiroView, quem string, x, y int) string {
	parada := fmt.Sprintf("'/mesa/%d/%d/tabuleiro/%s/parada/' + (%d + dx) + '/' + (%d + dy)",
		v.CampaignID, v.SessionID, v.AlvoDoMovimento, x, y)
	destino := "@post(" + parada + ")"
	if quem == "peca" && v.Mestre && v.AlvoDoMovimento != "" {
		grupo := fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/grupo/mover/' + dx + '/' + dy)",
			v.CampaignID, v.SessionID)
		destino = fmt.Sprintf("%s ? %s : %s", aPecaEstaMarcada(v.AlvoDoMovimento), grupo, destino)
	}
	return fmt.Sprintf(
		"if ($arrastando === '%s') { "+
			"const dx = Math.round($arrastox / $quadrado), dy = Math.round($arrastoy / $quadrado); "+
			"$arrastando = ''; $arrastox = 0; $arrastoy = 0; "+
			"if (dx || dy) %s }", quem, destino)
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

// ── QUEM RECEBE O GESTO DA PEÇA, decidido em GO ──────────────────────────────
//
// As três funções abaixo existem por um defeito MUDO do templ, e ele custa caro
// porque o código-fonte parecia certo:
//
//	if v.ArrastaAPeca == p.ID {
//	    data-on:pointerdown={ pegaParaArrastar("peca") }
//	} else if v.Mestre {
//	    data-on:pointerdown={ pegaOGrupo(p.ID) }
//	}
//
// **O templ NÃO aceita `else if` numa lista de atributos.** Ele fecha o primeiro
// `if`, escreve a palavra ` else` como TEXTO no meio das aspas do elemento e abre
// um `if` INDEPENDENTE — então os dois ramos saíam juntos, e o HTML servido tinha
// um atributo literalmente chamado `else` e `data-on:pointerdown` DUAS VEZES.
//
// Nada estourava: atributo repetido não existe no DOM, o navegador guarda o
// PRIMEIRO e descarta o resto em silêncio. O ramo do grupo estava morto em toda
// peça que era alvo do movimento, e só não se percebeu porque o `soltaEPara` já
// escolhe o grupo por conta própria quando a peça está marcada — a decisão certa
// chegava pelo outro caminho. O aviso já estava escrito duas linhas abaixo, no
// `oVestidoDaPeca`: *"UM `data-class` só porque atributo repetido não existe"*.
//
// A escolha volta para o Go, onde `else if` é `else if`, e o elemento passa a ter
// UMA lista de atributos. É a mesma forma que a fatia 1 desta issue usou para as
// ferramentas: **exclusão por CONSTRUÇÃO**, e não por dois blocos que se
// prometem exclusivos.

// aPecaRecebeOGesto diz se ela escuta o ponteiro.
//
// O mestre entra sempre porque marcar é gesto dele: uma peça que não é alvo do
// movimento ainda pode estar num grupo marcado, e o `pegaOGrupo` é quem checa a
// marca. Para o jogador só a peça dele responde.
func aPecaRecebeOGesto(v tabuleiroView, id string) bool {
	return v.ArrastaAPeca == id || v.Mestre
}

// oPegarDaPeca escolhe entre começar o arrasto DA PEÇA e o DO GRUPO.
func oPegarDaPeca(v tabuleiroView, id string) string {
	if v.ArrastaAPeca == id {
		return pegaParaArrastar("peca")
	}
	return pegaOGrupo(id)
}

// oSoltarDaPeca é o par do `oPegarDaPeca`, e os dois têm de concordar: um
// `pointerdown` de grupo com um `pointerup` de parada proporia o movimento de
// uma peça que a pessoa nem estava movendo.
//
// As coordenadas são as DESENHADAS (`p.X`/`p.Y`), que com movimento proposto são
// o fim do caminho — é o que faz a próxima parada contar do lugar onde a peça
// está (ALE-203, item 4).
func oSoltarDaPeca(v tabuleiroView, p pecaDoTabuleiro) string {
	if v.ArrastaAPeca == p.ID {
		return apagaAPrevia + "; " + soltaEPara(v, "peca", p.X, p.Y)
	}
	return soltaOGrupo(v)
}

// oSeguirDaPeca é o par do `oPegarDaPeca` no `pointermove`: a peça que se
// arrasta ganha a PRÉVIA, e o grupo continua só empurrando pixels.
//
// A divisa é a mesma dos outros dois, e ela tem de ser: a prévia mede o custo de
// UMA peça, e o gesto do grupo move várias sem regra de deslocamento nenhuma —
// pedir prévia ali desenharia a seta de uma peça sobre o arrasto de todas.
func oSeguirDaPeca(v tabuleiroView, p pecaDoTabuleiro) string {
	if v.ArrastaAPeca == p.ID {
		return oDedoSegueComPrevia(v, p)
	}
	return segueODedo("peca")
}

// estaArrastando marca o elemento que o CSS deve deslocar.
func estaArrastando(quem string) string {
	return fmt.Sprintf("{'tabuleiro-arrastando': $arrastando === '%s'}", quem)
}

// comandoDoTabuleiroDaCena escreve a chamada de abrir ou encerrar.
//
// Irmão do `comandoDoMovimento` e separado dele de propósito: aquele leva o id
// da PEÇA no caminho, e este não tem peça nenhuma — abrir acontece justamente
// quando não há tabuleiro.
func comandoDoTabuleiroDaCena(v tabuleiroView, acao string) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/%s')", v.CampaignID, v.SessionID, acao)
}

// acervoDaCampanha traduz os lugares guardados para a tela.
//
// A DATA é encurtada para o dia: o acervo responde "quando joguei isto?", e a
// hora não ajuda a escolher entre a taverna de ontem e a cripta de março. O
// formato vem do banco em ISO, e cortar no `T` é mais honesto que reformatar —
// não inventa fuso que o servidor não guardou.
func acervoDaCampanha(lugares []tabuleiro.Place, abertos []*tabuleiro.BoardState) []lugarDoAcervo {
	// O índice é montado UMA vez: com 148 lugares e oito abas, comparar cada
	// linha com cada aba é a lista inteira multiplicada pelo número de cenas
	// abertas, a cada carga da página e a cada quadro do stream.
	naMesa := make(map[string]string, len(abertos))
	for _, aberto := range abertos {
		naMesa[aberto.Place] = aberto.ID
	}
	acervo := make([]lugarDoAcervo, 0, len(lugares))
	for _, l := range lugares {
		acervo = append(acervo, lugarDoAcervo{
			ID: l.ID, Nome: l.Name, Pecas: l.Tokens, Quando: diaDe(l.UpdatedAt),
			// Pelo NOME, que é a identidade que o `Archive` já dá ao lugar — ver
			// `aAbaComOLugar`, onde o argumento inteiro está escrito.
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

// comandoDoLugar escreve a chamada de reabrir ou apagar um lugar do acervo.
func comandoDoLugar(v tabuleiroView, placeID int64, acao string) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/lugares/%d/%s')",
		v.CampaignID, v.SessionID, placeID, acao)
}

// comandoDaAba escreve a troca de aba a partir do acervo (ALE-205, fatia 3).
//
// A MESMA rota que a barra de abas usa, e não uma "reabrir que só troca": o que
// se quer aqui é literalmente ir até a aba que já existe, e uma segunda porta
// para isso seria uma segunda regra sobre o que significa escolher uma cena.
func comandoDaAba(v tabuleiroView, tabuleiroID string) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/aba/%s')",
		v.CampaignID, v.SessionID, tabuleiroID)
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
		"@post('/mesa/%d/%d/tabuleiro/marcadores/novo/' + (%s) + '/' + (%s))",
		v.CampaignID, v.SessionID, clicouEmX, clicouEmY,
	)
}

// comandoDoMarcador escreve o gesto sobre um marcador que já existe.
func comandoDoMarcador(v tabuleiroView, id, acao string) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/marcadores/%s/%s')",
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

// comandoDaCortina escreve o gesto que fecha ou abre (ALE-202, ALE-269).
func comandoDaCortina(v tabuleiroView, estado string) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/tabuleiro/cortina/%s')",
		v.CampaignID, v.SessionID, estado)
}

// destinoDaCortina é para onde o botão do cabeçalho leva.
//
// O botão ALTERNA e a tira só ABRE, e são dois destinos e não um alternar cego —
// a razão está no `correACortina`. Aqui é só a tradução do estado atual para o
// verbo que falta.
func destinoDaCortina(fechada bool) string {
	if fechada {
		return "abrir"
	}
	return "fechar"
}
