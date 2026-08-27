package api

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// Os guardas da JANELA sobre o plano infinito (ALE-203).
//
// O que se prende aqui é o que a moldura protegia por construção e passou a
// depender de uma CONVENÇÃO: enquanto o servidor desenhava um retângulo, "onde o
// dedo caiu" era uma conta sobre um elemento de tamanho conhecido. Agora é uma
// conta sobre a JANELA — o ponto do clique mais a vista, dividido pelo zoom — e
// ela aparece em cinco camadas de clique.

// TestNenhumaCamadaLeOPontoSemSomarAJanela.
//
// Este é o guarda da FAMÍLIA e não de um defeito: a conta do clique tinha CINCO
// cópias, uma por camada, e o comentário de cada uma dizia que a repetição era
// deliberada. Sem moldura, todas as cinco ganharam a janela dentro delas — e uma
// que ficasse para trás clicaria no quadrado errado, sem erro nenhum, com o
// desvio crescendo à medida que a pessoa arrasta a vista.
//
// Ele varre o HTML SERVIDO e não o código: é a única forma de alcançar a camada
// que alguém escrever amanhã sem ler nada disto.
func TestNenhumaCamadaLeOPontoSemSomarAJanela(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, http.MethodPost, f.urlDaMesa()+"/tabuleiro/abrir",
		`{"novolugar":"Taverna do Javali","novochao":"taverna"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir o tabuleiro deu %d", rec.Code)
	}
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	// O CONTROLE vem primeiro: sem ele, "não achei nenhuma leitura crua" é
	// indistinguível de "não achei leitura nenhuma" — e as duas passariam verde.
	lidas := regexp.MustCompile(`evt\.offset[XY]`).FindAllString(tela, -1)
	if len(lidas) < 8 {
		t.Fatalf("a cena só tem %d leituras de ponto: o canal não está aberto, e a ausência abaixo não é evidência", len(lidas))
	}

	// Cada leitura tem de vir somada à vista. A expressão é sempre
	// `(evt.offsetX + $vistax)`, então basta olhar o que vem logo depois.
	for _, eixo := range []struct{ ponto, sinal string }{
		{"evt.offsetX", sinalDaVistaX},
		{"evt.offsetY", sinalDaVistaY},
	} {
		olho := regexp.MustCompile(regexp.QuoteMeta(eixo.ponto) + `(?: \+ \$` + eixo.sinal + `)?`)
		for _, achado := range olho.FindAllString(tela, -1) {
			if !strings.Contains(achado, eixo.sinal) {
				t.Errorf("uma camada lê %q sem somar a janela: com a vista arrastada ela clica no quadrado errado", achado)
			}
		}
	}
}

// TestOAtalhoNaoMudaQuandoUmaFerramentaDoMestreSomeDoTrilho.
//
// É a promessa que o cabeçalho do trilho faz: quem aprendeu `4 = gabarito`
// mestrando continua com `4 = gabarito` jogando. Ela vale porque o número sai da
// posição na lista COMPLETA, e o filtro por papel acontece DEPOIS.
//
// O caso é montado à mão de propósito. No trilho de hoje todas as ferramentas do
// mestre estão no FIM, então numerar depois do filtro daria o mesmo resultado — o
// guarda passaria sobre a sabotagem, que é o pior tipo de verde. O que quebra a
// promessa é uma ferramenta do mestre no MEIO, e é ela que este trilho tem.
func TestOAtalhoNaoMudaQuandoUmaFerramentaDoMestreSomeDoTrilho(t *testing.T) {
	trilho := numeraOTrilho([]ferramentaDoMapa{
		{ID: "mover", Rotulo: "Mover"},
		{ID: "pintar", Rotulo: "Pintar", SoMestre: true},
		{ID: "regua", Rotulo: "Régua"},
	})
	doMestre := asVisiveisPara(true, trilho)
	doJogador := asVisiveisPara(false, trilho)

	// O CONTROLE: os dois trilhos têm de DIFERIR, senão não há filtro medindo.
	if len(doMestre) == len(doJogador) {
		t.Fatalf("o filtro não tirou nada (%d entradas nos dois): não há o que comparar", len(doMestre))
	}
	if doJogador[len(doJogador)-1].Atalho != doMestre[len(doMestre)-1].Atalho {
		t.Errorf("a régua é a tecla %s para o mestre e %s para o jogador — a ferramenta que sumiu do meio renumerou o resto",
			doMestre[len(doMestre)-1].Atalho, doJogador[len(doJogador)-1].Atalho)
	}
}

// TestCadaFerramentaTemUmaTeclaSoDela.
//
// Duas ferramentas com o mesmo número não dão erro: a segunda simplesmente nunca
// liga, porque o `oTecladoDoTrilho` monta um ternário encadeado e o primeiro
// ramo vence. Foi o defeito que os números escritos à mão convidavam, e é o
// motivo de o `numeraOTrilho` existir.
func TestCadaFerramentaTemUmaTeclaSoDela(t *testing.T) {
	vistas := map[string]string{}
	for _, f := range asFerramentasDoMapa() {
		if f.Atalho == "" {
			t.Errorf("a ferramenta %q nasceu sem atalho", f.Rotulo)
			continue
		}
		if dono, tem := vistas[f.Atalho]; tem {
			t.Errorf("a tecla %s liga %q e %q — a segunda nunca acende", f.Atalho, dono, f.Rotulo)
		}
		vistas[f.Atalho] = f.Rotulo
	}
}

// TestAMaoDeArrastarAVistaEDosDoisPapeis.
//
// Sem moldura não há rolagem nativa, então arrastar a vista deixou de ser
// conforto: é o único gesto de ponteiro que leva alguém ao outro lado do plano.
// Um trilho de jogador sem a mão seria um jogador preso no enquadramento em que
// a página abriu.
func TestAMaoDeArrastarAVistaEDosDoisPapeis(t *testing.T) {
	for _, mestre := range []bool{true, false} {
		achou := false
		for _, f := range oTrilhoDe(mestre) {
			achou = achou || f.ID == FerramentaDaVista
		}
		if !achou {
			t.Errorf("mestre=%v não tem a mão de arrastar a vista, e sem ela não há como percorrer o plano", mestre)
		}
	}
}
