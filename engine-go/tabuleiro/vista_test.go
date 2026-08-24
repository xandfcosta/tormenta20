package tabuleiro

import (
	"testing"

	"t20engine/engine"
)

// ── a aparência da peça (ALE-179, portada na ALE-263) ────────────────────────
//
// A regra: a cor é da ESPÉCIE e o número é da INSTÂNCIA. Os casos são os que a
// suíte da SPA já nomeava, porque as bordas não mudaram com o transporte.

func TestOsIguaisSaemIguaisEONumeroNaoEntraNaCor(t *testing.T) {
	um, tres := AparenciaDe("Zumbi 1"), AparenciaDe("Zumbi 3")

	if um.Matiz != tres.Matiz {
		t.Errorf("dois zumbis saíram em matizes diferentes: %d e %d", um.Matiz, tres.Matiz)
	}
	if um.Monograma != tres.Monograma {
		t.Errorf("monogramas diferentes: %q e %q", um.Monograma, tres.Monograma)
	}
	if um.Instancia != "1" || tres.Instancia != "3" {
		t.Errorf("selos: %q e %q", um.Instancia, tres.Instancia)
	}
}

// O monograma vem da ESPÉCIE. Ele comia as duas primeiras palavras, então
// "Zumbi Putrefato 2" virava "ZP" e o número — a única coisa que distingue as
// três peças na mesa — era justamente o que se perdia.
func TestOMonogramaVemDaEspecieEONumeroViraSelo(t *testing.T) {
	p := AparenciaDe("Zumbi Putrefato 2")
	if p.Monograma != "ZP" || p.Instancia != "2" {
		t.Errorf("ficou %q + %q, queria ZP + 2", p.Monograma, p.Instancia)
	}
}

// DUAS letras mesmo em nome de uma palavra: no tabuleiro um "O" solto tem
// metade da massa que a peça precisa para ser achada entre vinte vizinhas.
func TestSemNumeroNaoHaSeloEOMonogramaAindaTemDuasLetras(t *testing.T) {
	p := AparenciaDe("Ogro")
	if p.Instancia != "" {
		t.Errorf("apareceu selo %q num nome sem número", p.Instancia)
	}
	if p.Monograma != "OG" {
		t.Errorf("monograma %q, queria OG", p.Monograma)
	}
}

// "Nv1" está no MEIO do nome e não é instância: separar por qualquer dígito
// transformaria "Recruta Nv1 Simples" em outra espécie.
func TestNumeroNoMeioDoNomeNaoEInstancia(t *testing.T) {
	p := AparenciaDe("Recruta Nv1 Simples")
	if p.Instancia != "" {
		t.Errorf("o Nv1 virou selo %q", p.Instancia)
	}
	if p.Monograma != "RN" {
		t.Errorf("monograma %q", p.Monograma)
	}
}

func TestEspeciesDiferentesContinuamDistintas(t *testing.T) {
	if AparenciaDe("Zumbi 1").Matiz == AparenciaDe("Goblin 1").Matiz {
		t.Error("zumbi e goblin saíram no mesmo matiz — a cor deixou de dizer algo")
	}
}

// O MATIZ É O MESMO DA SPA, e tem de continuar sendo: a peça do tabuleiro e o
// retrato do herói mostram a mesma criatura, e duas fórmulas dariam duas cores
// para ela em duas telas.
//
// Os números foram RODADOS no `hueFromName` da SPA com node e transcritos, e é
// isso que faz deste um guarda de contrato entre as duas linguagens. Derivá-los
// de um hash reescrito na asserção compararia a função consigo mesma — que foi
// a primeira versão deste teste, e ela passava sem provar nada.
//
// O "Ácido" está aqui de propósito: o JavaScript percorre PONTOS DE CÓDIGO
// (`for ch of name`), e um port que iterasse BYTES daria outro número em todo
// nome acentuado. É a única entrada da lista que pega esse erro.
//
// A doc do `hueFromName` diz `hueFromName('Thorvald') // => 214`, e ELA ESTÁ
// ERRADA: rodado, o valor é 186. Foi este teste que descobriu, porque nada no
// front afirma aquele exemplo. Avisada a sessão que cuida da SPA.
func TestOMatizEOMesmoDoRetratoDaSPA(t *testing.T) {
	casos := map[string]int{
		"Thorvald": 186,
		"Ogro":     197,
		"Zumbi":    9,
		"Arwen":    119,
		"Ácido":    218,
		"Goblin":   183,
	}
	for nome, quero := range casos {
		if got := matizDe(nome); got != quero {
			t.Errorf("matiz de %q = %d, quero %d (rodado no hueFromName da SPA)", nome, got, quero)
		}
	}
}

// ── o extenso (ALE-263) ──────────────────────────────────────────────────────

// Um tabuleiro VAZIO desenha o piso, e não o nada.
func TestOTabuleiroVazioDesenhaOPiso(t *testing.T) {
	e := ExtensoDe(&BoardState{})
	if e.Colunas != MinimoDeColunas || e.Linhas != MinimoDeLinhas {
		t.Errorf("vazio deu %dx%d, queria o piso %dx%d", e.Colunas, e.Linhas, MinimoDeColunas, MinimoDeLinhas)
	}
	// E o piso nasce CENTRADO na origem: quem abre um tabuleiro põe a primeira
	// peça perto do meio, não na quina.
	if e.X0 > 0 || e.Y0 > 0 || e.X0+e.Colunas < 1 || e.Y0+e.Linhas < 1 {
		t.Errorf("o piso vazio não contém a origem: %+v", e)
	}
}

// O extenso ENVOLVE tudo que tem lugar, com margem — e a peça grande entra
// inteira, não só a quina dela.
func TestOExtensoEnvolveTudoComMargemEContaAPegada(t *testing.T) {
	pv := &BoardState{
		Tokens: []BoardToken{
			{ID: "a", Label: "Ogro", X: 40, Y: 30, Footprint: 2},
			{ID: "b", Label: "Arwen", X: 38, Y: 28},
		},
		Markers:   []BoardMarker{{ID: "m", X: 45, Y: 33}},
		Difficult: []engine.Square{{X: 36, Y: 27}},
	}
	e := ExtensoDe(pv)

	// A asserção é de CONTINÊNCIA e não de números exatos, e a primeira versão
	// deste teste errou por isso: eu previ origem 33 (o mínimo do conteúdo, 36,
	// menos a margem 3) e o código deu 31 — porque o conteúdo mais a margem dá
	// 16 colunas, abaixo do piso de 20, e o piso é aplicado CENTRADO. O código
	// estava certo e a minha conta é que não tinha o piso.
	//
	// Continência é o que a regra promete de verdade: tudo que tem lugar cabe,
	// com pelo menos a margem sobrando de cada lado.
	dentro := func(x, y int, oque string) {
		t.Helper()
		if x < e.X0+MargemDoExtenso || x > e.X0+e.Colunas-1-MargemDoExtenso ||
			y < e.Y0+MargemDoExtenso || y > e.Y0+e.Linhas-1-MargemDoExtenso {
			t.Errorf("%s em %d,%d não tem a margem de %d dentro do extenso %+v", oque, x, y, MargemDoExtenso, e)
		}
	}
	dentro(36, 27, "o terreno difícil")
	dentro(45, 33, "o marcador")
}

// Pegada ZERO é peça de um quadrado: o campo é `omitempty` no fio, e tratá-la
// como zero encolheria o extenso para DENTRO da própria peça.
func TestPegadaZeroContaComoUmQuadrado(t *testing.T) {
	e := ExtensoDe(&BoardState{Tokens: []BoardToken{{ID: "a", Label: "Rato", X: 100, Y: 100}}})
	if e.X0+e.Colunas-1 < 100+MargemDoExtenso {
		t.Errorf("o extenso acaba em x=%d e não cobre a peça em 100 mais a margem", e.X0+e.Colunas-1)
	}
}

// COORDENADA NEGATIVA é lugar legítimo: o plano não tem bordas, e um extenso que
// se recusasse a ir para o negativo empurraria a peça para fora do desenho.
func TestOExtensoVaiParaOnegativo(t *testing.T) {
	e := ExtensoDe(&BoardState{Tokens: []BoardToken{{ID: "a", Label: "Ogro", X: -20, Y: -15}}})
	if e.X0 > -20 || e.Y0 > -15 {
		t.Errorf("a peça em -20,-15 ficou fora do extenso %+v", e)
	}
}

// A PEGADA entra inteira, e este teste precisou de duas tentativas para medir
// isso de verdade.
//
// A primeira punha a peça grande no meio de um tabuleiro onde um marcador estava
// mais longe — então ignorar a pegada não mudava o extenso, e a sabotagem passou
// VERDE. Aqui a peça grande é o extremo e nada está além dela; e as duas peças
// estão longe o bastante para o span passar do piso, senão o piso mascararia a
// diferença exatamente como o marcador mascarava.
func TestAPegadaEntraInteiraNoExtenso(t *testing.T) {
	e := ExtensoDe(&BoardState{Tokens: []BoardToken{
		{ID: "perto", Label: "Rato", X: 0, Y: 0},
		{ID: "longe", Label: "Dragão", X: 30, Y: 20, Footprint: 3},
	}})

	// O Dragão de pegada 3 em (30,20) ocupa até (32,22). Ignorar a pegada faria
	// o extenso acabar em 33, e a quina da peça ficaria sem margem — ou fora.
	limiteX, limiteY := e.X0+e.Colunas-1-MargemDoExtenso, e.Y0+e.Linhas-1-MargemDoExtenso
	if limiteX < 32 || limiteY < 22 {
		t.Errorf("a quina do Dragão (32,22) não tem margem dentro do extenso %+v (limite %d,%d)", e, limiteX, limiteY)
	}
}
