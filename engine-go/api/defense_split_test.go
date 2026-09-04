package api

import (
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// A DEFESA DO ALVO CAÍDO aparece PARTIDA nas duas telas que a mostram (ALE-274).
//
// O Caído dá −5 na Defesa contra ataques corpo a corpo e +5 contra ataques à
// distância (T20 p394), então um personagem caído não tem "a Defesa": tem duas.
// O motor sempre calculou as duas; nenhuma tela as dizia, e o jogador só as
// descobria abrindo o diálogo de decomposição — que ninguém abre para conferir
// um número que a tela já mostra com confiança.
//
// AS DUAS TELAS NO MESMO CASO, e é isso que o torna honesto: afirmar só a ficha
// não distingue "o mestre não vê" de "ninguém vê", e afirmar só o elenco
// passaria verde sobre uma ficha que continuou mentindo. Elas mostram o MESMO
// herói, e a ALE-122 já pagou o preço de duas telas discordando sobre ele.
func TestTheProneDefenseIsSplitOnTheSheetAndInTheCast(t *testing.T) {
	f := novoPiloto(t)
	heroi := strconv.FormatInt(f.charID, 10)

	// O CONTROLE vem primeiro: em pé, a Defesa é UM número, e é assim que se sabe
	// que o caso mede a mudança e não um texto que já estava lá.
	emPe := f.pede(t, f.jogador, "GET", "/personagens/"+heroi+"?tab=combat", "").Body.String()
	if strings.Contains(emPe, " CaC · ") {
		t.Fatal("a ficha já mostrava a Defesa partida em pé — o caso mediria o que não mudou")
	}

	if rec := f.pede(t, f.jogador, "POST",
		"/personagens/"+heroi+"/efeitos/condicao/caido", ""); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Caído deu %d: %s", rec.Code, rec.Body.String())
	}

	caido := f.pede(t, f.jogador, "GET", "/personagens/"+heroi+"?tab=combat", "").Body.String()
	if !strings.Contains(caido, " CaC · ") || !strings.Contains(caido, " Dist") {
		t.Errorf("a ficha do caído não partiu a Defesa: %s", "(a caixa de Defesa não trouxe o par)")
	}

	// E o DIÁLOGO DO ELENCO, que é onde o mestre confere a Defesa de um jogador
	// para decidir se o ataque acerta — o lugar onde o custo do erro é maior.
	naMesa := f.pede(t, f.mestre, "GET", f.tableUrl(), "").Body.String()
	if !strings.Contains(naMesa, " CaC · ") {
		t.Errorf("o elenco da Mesa não partiu a Defesa do caído: %s", "(o elenco não trouxe o par)")
	}
}

// A LISTA DE HERÓIS mantém o número único, e a assimetria é deliberada
// (ALE-274).
//
// Decisão do dono: o cartão é CATÁLOGO fora da sessão — ninguém está resolvendo
// ataque ali, e um par de números onde se comparam heróis é ruído. O crachá da
// ficha e o diálogo do elenco mostram o par porque os dois existem para
// responder "acerta?".
//
// Ele é prendido porque é o tipo de diferença que alguém "arruma" na primeira
// leitura, achando que é esquecimento — e é escolha.
func TestTheHeroListKeepsTheSingleDefenseNumber(t *testing.T) {
	f := novoPiloto(t)
	heroi := strconv.FormatInt(f.charID, 10)
	if rec := f.pede(t, f.jogador, "POST",
		"/personagens/"+heroi+"/efeitos/condicao/caido", ""); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Caído deu %d", rec.Code)
	}

	lista := f.pede(t, f.jogador, "GET", "/personagens", "").Body.String()

	// O CONTROLE: o herói TEM de estar na lista, senão o caso mede a ausência
	// dele e passa verde dizendo nada.
	if !strings.Contains(lista, "Arcanista") {
		t.Fatal("o herói não apareceu na lista — o caso mediria outra coisa")
	}
	if strings.Contains(lista, " CaC · ") {
		t.Error("a lista partiu a Defesa: ali o par é ruído, e a decisão foi mantê-la inteira")
	}
}
