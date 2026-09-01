package api

import (
	"net/http"
	"strconv"
	"t20engine/events"
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
	aviso, parar := f.s.bus.Subscribe(events.OfCharacter(f.charID))
	defer parar()

	// Um comando qualquer que GRAVA: tirar 1 de PV.
	rec := f.pede(t, f.jogador, http.MethodPost,
		"/piloto/personagens/"+strconv.FormatInt(f.charID, 10)+"/vitais/pv/-1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("o comando respondeu %d", rec.Code)
	}

	select {
	case ev := <-aviso.C:
		if _, ok := ev.(events.CharacterChanged); !ok {
			t.Fatalf("chegou %T, e quem escuta a ficha espera um CharacterChanged", ev)
		}
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
	aviso, parar := f.s.bus.Subscribe(events.OfCharacter(f.charID))
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
	case ev := <-aviso.C:
		t.Fatalf("um gesto recusado publicou %T", ev)
	case <-time.After(100 * time.Millisecond):
	}
}

// Aqui morava `TestABaixaTiraOOuvinteDaFicha`. A baixa deixou de ser do
// `CharacterWatch` — ela é do barramento, e está medida onde mora, em
// `events.TestABaixaTiraOOuvinte`. O que este arquivo protege é outra coisa: que
// o GATEWAY publique, e que a recusa não publique.
