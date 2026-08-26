package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/engine"
	"t20engine/tabuleiro"
)

// Os guardas do GESTO CONTÍNUO do pincel (ALE-203, itens 8 e 9 do dono).
//
// A aritmética do traço não é medida aqui: `tabuleiro.CasasDoTraco` tem guarda
// próprio, e ele prende a regra ("o traço não tem buraco") no lugar mais barato.
// O que se prende deste lado é o que só existe deste lado — que a rota pinta o
// SEGMENTO numa gravação só, que a resposta não devolve a Mesa inteira, e que a
// tela liga os gestos que fazem o traço acontecer.

// TestOTracoPintaOSegmentoInteiro.
//
// O caso mede o SEGMENTO, e não o número de gravações. Eu tinha escrito uma
// segunda asserção sobre a `Version` do tabuleiro — "uma versão a mais, senão a
// mesa recebe um quadro por casa" — e ela media a coisa errada: o `PaintTerrain`
// sobe a versão POR CASA, então um traço de dez casas sobe dez, dentro de um
// `apply` só. Quem garante a gravação única é a estrutura (`PintaOTraco` chama
// `apply` uma vez, e o `comandoDoTabuleiro` publica uma vez), não um contador —
// e um teste que afirma o contrário fica vermelho sobre um app correto.
func TestOTracoPintaOSegmentoInteiro(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	rec := f.pede(t, f.mestre, http.MethodPost,
		f.urlDaMesa()+"/tabuleiro/terreno/dificil/2/2/ate/8/5", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o traço deu %d", rec.Code)
	}

	b := f.s.boards.Get(context.Background(), f.sessionID)
	casas := tabuleiro.QuadradosDe(b, "dificil")
	esperadas := tabuleiro.CasasDoTraco(engine.Square{X: 2, Y: 2}, engine.Square{X: 8, Y: 5})
	if len(casas) != len(esperadas) {
		t.Errorf("o traço (2,2)→(8,5) pintou %d casas, esperado as %d do segmento: %v",
			len(casas), len(esperadas), casas)
	}
	// E o traço não tem buraco na ponta que este lado controla: a primeira e a
	// última casa do segmento estão lá. O meio é problema do `CasasDoTraco`, que
	// tem guarda próprio.
	for _, ponta := range []engine.Square{{X: 2, Y: 2}, {X: 8, Y: 5}} {
		if !contem(casas, ponta) {
			t.Errorf("a casa %v não foi pintada: o traço não chega às pontas", ponta)
		}
	}
}

func contem(casas []engine.Square, alvo engine.Square) bool {
	for _, c := range casas {
		if c == alvo {
			return true
		}
	}
	return false
}

// TestOTracoDaBorrachaApagaOSegmentoInteiro: o irmão do de cima, e ele existe
// porque as duas rotas são caminhos diferentes — a da borracha não tem espécie,
// e foi justamente ela que ficou para trás na primeira versão desta superfície.
func TestOTracoDaBorrachaApagaOSegmentoInteiro(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.urlDaMesa()+"/tabuleiro/terreno/cobertura/0/0/ate/6/6", ""); rec.Code != http.StatusOK {
		t.Fatalf("pintar deu %d", rec.Code)
	}
	// O CONTROLE: havia o que apagar. Sem ele, "sobrou zero" é verdade também
	// sobre um tabuleiro em que nada foi pintado.
	b := f.s.boards.Get(context.Background(), f.sessionID)
	if len(tabuleiro.QuadradosDe(b, "cobertura")) < 7 {
		t.Fatalf("o traço de pintura só fez %d casas — não há o que a borracha apagar",
			len(tabuleiro.QuadradosDe(b, "cobertura")))
	}

	if rec := f.pede(t, f.mestre, http.MethodPost,
		f.urlDaMesa()+"/tabuleiro/terreno/limpar/0/0/ate/6/6", ""); rec.Code != http.StatusOK {
		t.Fatalf("apagar deu %d", rec.Code)
	}
	b = f.s.boards.Get(context.Background(), f.sessionID)
	if sobrou := tabuleiro.QuadradosDe(b, "cobertura"); len(sobrou) != 0 {
		t.Errorf("a borracha deixou %v pelo caminho", sobrou)
	}
}

// TestOTracoForjadoERecusado.
//
// O teto é do domínio e a recusa chega como FRASE, não como 500: um traço de dez
// milhões de casas só vem de um pedido montado à mão, e a resposta certa é dizer
// o que houve.
func TestOTracoForjadoERecusado(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	corpo := f.pede(t, f.mestre, http.MethodPost,
		f.urlDaMesa()+"/tabuleiro/terreno/dificil/0/0/ate/9999999/0", "").Body.String()
	if !strings.Contains(corpo, "longo demais") {
		t.Errorf("o traço forjado não foi recusado com frase: %q", corpo[max(0, len(corpo)-200):])
	}
	b := f.s.boards.Get(context.Background(), f.sessionID)
	if casas := tabuleiro.QuadradosDe(b, "dificil"); len(casas) != 0 {
		t.Errorf("o traço recusado pintou %d casas assim mesmo", len(casas))
	}
}

// TestOPincelNaoDevolveAMesaInteira — o guarda dos 353 KB.
//
// Medido no navegador antes do conserto: uma casa pintada devolvia **353 KB**,
// porque o `respondeAoMestre` repinta TODAS as regiões. Num gesto de clique isso
// era caro; num gesto CONTÍNUO é proibitivo — um traço de vinte casas mandaria
// sete megabytes, e o mestre está arrastando o dedo enquanto isso chega.
//
// A asserção nomeia as duas metades: a região do mapa TEM de vir (senão o traço
// não aparece) e a do acervo NÃO pode (é a maior da Mesa, com 147 lugares, e ela
// não muda quando alguém pinta uma casa).
func TestOPincelNaoDevolveAMesaInteira(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")

	corpo := f.pede(t, f.mestre, http.MethodPost,
		f.urlDaMesa()+"/tabuleiro/terreno/dificil/1/1/ate/1/1", "").Body.String()

	if !strings.Contains(corpo, `id="mesa-tabuleiro"`) {
		t.Error("a resposta do pincel não traz o mapa — a casa pintada não apareceria")
	}
	for _, region := range []string{"mesa-acervo", "mesa-fila", "mesa-grupo", "mesa-npcs"} {
		if strings.Contains(corpo, `id="`+region+`"`) {
			t.Errorf("a resposta do pincel repinta a região %q, que não muda ao pintar uma casa", region)
		}
	}
}

// TestATelaLigaOTracoEOBotaoDireito.
//
// Uma afirmação sobre a FORMA do que a página serve, e é o único jeito de
// alcançar os três gestos de uma vez: `pointerdown`/`pointermove`/`pointerup` na
// camada de pintura, a rota com `/ate/` (e não a de um ponto só), e o botão 2
// caindo no caminho da borracha.
//
// Se algum deles voltar a ser um `data-on:click`, o traço morre em silêncio — a
// tela continua pintando um quadrado por clique, que é exatamente o estado que o
// dono relatou.
func TestATelaLigaOTracoEOBotaoDireito(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	for _, pedaco := range []string{
		"data-on:pointerdown",
		"data-on:pointermove",
		"data-on:pointerup",
		"data-on:contextmenu",
		"/ate/",
		"evt.button === 2",
	} {
		if !strings.Contains(tela, pedaco) {
			t.Errorf("a cena não tem %q: o traço do pincel não acontece", pedaco)
		}
	}
	// A CAMADA DE PINTURA não pode ter voltado ao clique de um quadrado só.
	if strings.Contains(tela, `aria-label="Pintar terreno — escolha a casa"`) {
		t.Error("a camada de pintura voltou a ser um clique por casa")
	}
}
