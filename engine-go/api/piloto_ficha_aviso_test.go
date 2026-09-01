package api

import (
	"net/http"
	"strconv"
	"testing"
	"time"
)

// O AVISO DE "esta ficha mudou" chega a quem escuta (ALE-275).
//
// A ficha dentro da sessão não é região do stream: ela se mantém em dia porque
// alguém CUTUCA o `CharacterWatch` quando o personagem muda. Esse alguém tem de
// ser o gateway, e não cada comando — são mais de trinta passando pelo
// `comandoDaFicha`, e a linha esquecida num deles seria uma ficha que não
// atualiza só naquele gesto.
//
// É a família do gancho que o `characterChanged` documenta: um aviso que
// depende de cada chamador lembrar nasce meio desligado, e o Go segue verde.
func TestOComandoDaFichaAvisaQuemEscuta(t *testing.T) {
	f := novoPiloto(t)
	aviso, parar := f.s.fichas.Assinar(f.charID)
	defer parar()

	// Um comando qualquer que GRAVA: tirar 1 de PV.
	rec := f.pede(t, f.jogador, http.MethodPost,
		"/piloto/personagens/"+strconv.FormatInt(f.charID, 10)+"/vitais/pv/-1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o comando respondeu %d", rec.Code)
	}

	select {
	case <-aviso:
	case <-time.After(time.Second):
		t.Fatal("ninguém foi avisado: a ficha dentro da sessão ficaria com o estado velho")
	}
}

// E o gesto RECUSADO não avisa: uma regra que barrou o clique não mudou nada, e
// avisar faria toda tela que escuta refazer a ficha por causa do que não
// aconteceu.
//
// A recusa é uma PROFICIÊNCIA que não existe, e não um passo de vital fora da
// faixa: passo fora da faixa é aceito e PRENDIDO na borda, então ele grava — a
// primeira versão deste caso media isso e reprovava dizendo que faltava a
// recusa. Ela estava certa: não havia recusa nenhuma.
func TestOComandoRecusadoNaoAvisa(t *testing.T) {
	f := novoPiloto(t)
	aviso, parar := f.s.fichas.Assinar(f.charID)
	defer parar()

	rec := f.pede(t, f.jogador, http.MethodPost,
		"/piloto/personagens/"+strconv.FormatInt(f.charID, 10)+"/proficiencias/alterna/gargalhada", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("a recusa devia voltar 200 com a cena: veio %d", rec.Code)
	}
	if frase := aRecusaDaCena(rec.Body.String()); frase == "" {
		t.Fatal("a cena não trouxe a recusa — sem ela este caso mede outra coisa")
	}

	select {
	case <-aviso:
		t.Fatal("um gesto recusado avisou que a ficha mudou")
	case <-time.After(100 * time.Millisecond):
	}
}

// A BAIXA limpa o registro. Sem ela, cada aba fechada deixa um canal para
// sempre, e o `Avisar` passa a percorrer uma lista que só cresce.
func TestABaixaTiraOOuvinteDaFicha(t *testing.T) {
	f := novoPiloto(t)
	_, parar := f.s.fichas.Assinar(f.charID)
	if n := f.s.fichas.Ouvintes(f.charID); n != 1 {
		t.Fatalf("%d ouvintes depois de assinar, esperado 1", n)
	}
	parar()
	if n := f.s.fichas.Ouvintes(f.charID); n != 0 {
		t.Errorf("%d ouvintes depois da baixa, esperado 0", n)
	}
}
