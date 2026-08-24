package api

import (
	"fmt"

	"t20engine/aovivo"
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
func tabuleiroViewOf(b *tabuleiro.BoardState, saude map[string]int, naVez string) tabuleiroView {
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
