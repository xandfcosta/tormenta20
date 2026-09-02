package door

import "testing"

// O REDIRECIONAMENTO ABERTO é a regra de segurança desta cena, e o teste dela
// mora aqui desde a ALE-278 — antes ele estava no `api`, junto do resto dos
// guardas da porta, e não podia continuar: a função não é exportada.
//
// A regra é de uma linha e o custo de errá-la é a conta de alguém: um
// `?redirect=` que aceitasse endereço absoluto tira o jogador do nosso domínio
// para uma página que pode imitar esta, com a confiança já dada. A barra dupla é
// o caso que engana, porque `//outro.site` é relativo a PROTOCOLO e o navegador
// o trata como absoluto.
func TestTheRequestedDestinationOnlyAcceptsAnInternalPath(t *testing.T) {
	casos := map[string]string{
		"/campaigns/7":        "/campaigns/7",
		"":                    "/",
		"https://outro.site":  "/",
		"//outro.site":        "/", // protocol-relative: o navegador trata como absoluto
		"javascript:alert(1)": "/",
	}
	for entrada, quer := range casos {
		if got := requestedDestination(entrada); got != quer {
			t.Errorf("requestedDestination(%q) = %q, queria %q", entrada, got, quer)
		}
	}
}
