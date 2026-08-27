package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/engine"
	"t20engine/tabuleiro"
)

// Os guardas do TRILHO DE FERRAMENTAS (ALE-203), em Datastar.
//
// A forma — vertical, sobreposto — é do CSS e não se afirma aqui: o que um teste
// de render pode dizer sobre leiaute é pouco, e o que importa dela está no
// navegador. O que se prende é o que sobrevive a um `Contains`: que a lista, os
// botões e o TECLADO concordam sobre quais ferramentas existem, e que a borracha
// deixou de apagar a coisa errada.

// TestOTecladoEOTRILHOconcordamSobreQuemExiste.
//
// Os dois saem do mesmo `oTrilhoDe`, e este guarda existe porque a alternativa —
// uma tabela de teclas escrita à mão — falharia em silêncio dos dois jeitos: uma
// tecla que liga uma ferramenta sem botão, e um botão que a tecla não alcança.
func TestOTecladoEOTrilhoConcordamSobreQuemExiste(t *testing.T) {
	for _, papel := range []struct {
		Nome   string
		Mestre bool
	}{{"o mestre", true}, {"o jogador", false}} {
		teclado := oTecladoDoTrilho(papel.Mestre)
		for _, f := range oTrilhoDe(papel.Mestre) {
			if !strings.Contains(teclado, `evt.key === "`+f.Atalho+`"`) {
				t.Errorf("%s tem o botão %q e não tem a tecla %s", papel.Nome, f.Rotulo, f.Atalho)
			}
		}
	}
	// E o inverso: o jogador NÃO pode ter a tecla de uma ferramenta do mestre.
	// Uma tecla que liga o pincel na tela de quem não pinta deixaria o clique
	// mudo — a camada não existe lá, e o gesto simplesmente não faria nada.
	doJogador := oTecladoDoTrilho(false)
	for _, f := range asFerramentasDoMapa() {
		if f.SoMestre && strings.Contains(doJogador, `$ferramenta = "`+f.ID+`"`) {
			t.Errorf("o jogador tem a tecla de %q, que é do mestre", f.Rotulo)
		}
	}
}

// TestOAtalhoEFIXOporFerramenta.
//
// Numerar por POSIÇÃO faria a mesma ferramenta trocar de tecla entre os dois
// papéis — o trilho do jogador tem três entradas e o do mestre tem nove. Quem
// aprendeu `3 = gabarito` mestrando tem de continuar com `3 = gabarito` jogando.
func TestOAtalhoEFixoPorFerramenta(t *testing.T) {
	doMestre := map[string]string{}
	for _, f := range oTrilhoDe(true) {
		doMestre[f.ID] = f.Atalho
	}
	for _, f := range oTrilhoDe(false) {
		if doMestre[f.ID] != f.Atalho {
			t.Errorf("%q é a tecla %s para o jogador e %s para o mestre",
				f.Rotulo, f.Atalho, doMestre[f.ID])
		}
	}
	// E nenhuma tecla serve a duas ferramentas: a segunda ganharia a disputa no
	// `?:` encadeado e a primeira ficaria inalcançável, sem erro nenhum.
	vistas := map[string]string{}
	for _, f := range asFerramentasDoMapa() {
		if antes, repetida := vistas[f.Atalho]; repetida {
			t.Errorf("a tecla %s serve a %q e a %q", f.Atalho, antes, f.Rotulo)
		}
		vistas[f.Atalho] = f.Rotulo
	}
}

// TestOAtalhoNaoROUBAaTeclaDeQuemEscreve.
//
// Sem a guarda, digitar "5" no PV de um combatente trocaria a ferramenta do mapa
// atrás do formulário. Já aconteceu com o `-` do zoom, e é por isso que a guarda
// é uma constante compartilhada em vez de três cópias.
func TestOAtalhoNaoRoubaATeclaDeQuemEscreve(t *testing.T) {
	teclado := oTecladoDoTrilho(true)
	for _, alvo := range []string{"INPUT", "TEXTAREA", "SELECT"} {
		if !strings.Contains(teclado, alvo) {
			t.Errorf("o atalho não se protege de %s", alvo)
		}
	}
	if !strings.Contains(teclado, "isContentEditable") {
		t.Error("o atalho não se protege de um campo `contenteditable`")
	}
	// E o ESC NÃO pode estar aqui. Ele tem dono — o `cena.js` mapeia Escape para
	// "voltar" e o para no `document`, então um `keydown__window` nunca o vê.
	// Provado no navegador com controle: `F2` no mesmo nó liga a ferramenta e
	// `Escape` não chega nem a um `addEventListener` cru na janela.
	//
	// O guarda é NEGATIVO de propósito: um ramo de Escape aqui não daria erro em
	// lugar nenhum — ele só ficaria prometendo uma saída que a tela não cumpre.
	if strings.Contains(teclado, "Escape") {
		t.Error("o trilho promete o Esc, que o `cena.js` engole antes de chegar à janela")
	}
}

// TestABORRACHAlimpaACasaInteira — o conserto do defeito que o dono achou.
//
// Ela era um MODO que invertia o pincel selecionado: com `Cobertura` na mão,
// clicar num quadrado de `Difícil` mandava `terreno/cobertura/…?apagar=1`,
// apagava a cobertura que não estava ali, e a tela não dizia nada. Medido na
// bancada, clique a clique, antes de virar este teste.
//
// Agora a rota não tem espécie no caminho — não há como errar qual.
func TestABorrachaLimpaACasaInteira(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	casa := f.urlDaMesa() + "/tabuleiro/terreno"

	// Três espécies EMPILHADAS na mesma casa: é o caso que o modo antigo não
	// sabia resolver, porque ele tinha de escolher uma.
	for _, especie := range []string{"dificil", "cobertura", "elevado"} {
		if rec := f.pede(t, f.mestre, http.MethodPost, casa+"/"+especie+"/4/4/ate/4/4", ""); rec.Code != http.StatusOK {
			t.Fatalf("pintar %s deu %d", especie, rec.Code)
		}
	}
	b := f.s.boards.Get(context.Background(), f.sessionID)
	if len(b.Difficult) != 1 || len(b.Cover) != 1 || len(b.Elevated) != 1 {
		t.Fatalf("as três não foram pintadas: %d/%d/%d — sem o caso positivo o resto não mede nada",
			len(b.Difficult), len(b.Cover), len(b.Elevated))
	}

	if rec := f.pede(t, f.mestre, http.MethodPost, casa+"/limpar/4/4/ate/4/4", ""); rec.Code != http.StatusOK {
		t.Fatalf("limpar deu %d", rec.Code)
	}
	b = f.s.boards.Get(context.Background(), f.sessionID)
	if len(b.Difficult)+len(b.Cover)+len(b.Concealment)+len(b.Elevated) != 0 {
		t.Errorf("a borracha deixou terreno na casa: %d difícil, %d cobertura, %d camuflagem, %d elevado",
			len(b.Difficult), len(b.Cover), len(b.Concealment), len(b.Elevated))
	}
}

// TestABorrachaNaoDependeDoPincelSELECIONADO.
//
// É a metade do defeito que um teste de "limpa a casa" sozinho não pegaria: a
// rota antiga funcionava perfeitamente quando o pincel na mão era o certo. O que
// quebrava era o outro caso, e ele passava despercebido porque o servidor
// respondia 200.
//
// Aqui isso vira uma afirmação sobre a FORMA da rota: se a espécie voltar para o
// caminho, este teste cai.
func TestABorrachaNaoDependeDoPincelSelecionado(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	tela := f.pede(t, f.mestre, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if !strings.Contains(tela, "tabuleiro/terreno/limpar/") {
		t.Error("a borracha não usa a rota sem espécie")
	}
	if strings.Contains(tela, "apagar=1") {
		t.Error("a borracha voltou a ser um modo do pincel (`?apagar=1`)")
	}
	// E ela é FERRAMENTA: tem lugar no trilho, com tecla.
	if !strings.Contains(tela, "Borracha (tecla ") {
		t.Error("a borracha não é uma ferramenta do trilho")
	}
}

// TestLimparUmaCasaLIMPAdevolveFalso.
//
// A versão do tabuleiro é o que acorda a mesa inteira pelo stream. Subir por um
// clique em chão limpo mandaria um quadro para seis pessoas para dizer que nada
// mudou — e o `escreveMesa` compara o HTML depois, mas o trabalho de renderizar
// nove regiões já teria acontecido.
func TestLimparUmaCasaLimpaDevolveFalso(t *testing.T) {
	b := &tabuleiro.BoardState{}
	if tabuleiro.LimpaACasa(b, engine.Square{X: 1, Y: 1}) {
		t.Error("limpar chão limpo disse que mudou alguma coisa")
	}
	if b.Version != 0 {
		t.Errorf("a versão subiu para %d sem mudança nenhuma", b.Version)
	}
	b.Difficult = append(b.Difficult, engine.Square{X: 1, Y: 1})
	if !tabuleiro.LimpaACasa(b, engine.Square{X: 1, Y: 1}) {
		t.Error("limpar uma casa pintada disse que nada mudou")
	}
	if b.Version != 1 {
		t.Errorf("a versão ficou em %d depois de uma mudança", b.Version)
	}
}

// TestOTrilhoDoJogadorNaoTemOQueEleNaoPode.
//
// A trava de verdade é do servidor (`comandoDoMestreNoTabuleiro`); isto é a
// cortesia de não oferecer o que seria recusado. Mas ela também é o que impede um
// gesto MUDO: a camada de pintura não existe na cena do jogador, então uma
// ferramenta oferecida a ele seria um modo que liga e não faz nada.
func TestOTrilhoDoJogadorNaoTemOQueEleNaoPode(t *testing.T) {
	f := novoPiloto(t)
	f.abreTabuleiro(t, "pedra")
	tela := f.pede(t, f.jogador, http.MethodGet, f.urlDaMesa(), "").Body.String()

	if !strings.Contains(tela, "Régua (tecla ") {
		t.Fatal("o jogador não recebeu o trilho — a página não é o que este teste pensa que é")
	}
	for _, f := range asFerramentasDoMapa() {
		if f.SoMestre && strings.Contains(tela, f.Rotulo+" (tecla ") {
			t.Errorf("o jogador recebeu %q, que é do mestre", f.Rotulo)
		}
	}
}
