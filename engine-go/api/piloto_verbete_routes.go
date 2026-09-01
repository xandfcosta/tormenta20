package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// A rota do VERBETE CITADO (ALE-264): o conteúdo da caixa que um elo abre.
//
// Fora de `/mestre/catalogos` de propósito, e o endereço é da CASCA: o diálogo
// mora nela, e amanhã um elo pode sair do bestiário ou da Mesa. Uma rota por
// cena obrigaria cada uma a ter a sua cópia do mesmo handler.

// rotaDoVerbete é o endereço PÚBLICO, que é o que o `@get` do navegador pede.
const rotaDoVerbete = "/verbete"

func (s *Server) rotasDoVerbete(r chi.Router) {
	r.Get("/verbete", s.handleVerbeteDoElo)
}

// handleVerbeteDoElo devolve SÓ o miolo da caixa.
//
// Os parâmetros são os mesmos do endereço da cena (`aba`, `entrada`), e isso não
// é coincidência: o elo manda a consulta do próprio `href` para cá. Um segundo
// formato faria o link e o remendo poderem discordar sobre o que mostrar.
func (s *Server) handleVerbeteDoElo(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	grupo := grupoDaEntrada(abaConhecida(q.Get("aba")), q.Get("entrada"))

	// `parte` pede um PEDAÇO do verbete em vez do cartão inteiro — hoje só os
	// aprimoramentos da magia. Parâmetro e não segunda rota porque o que muda é
	// o miolo da mesma caixa: duas rotas obrigariam o cliente a saber qual
	// chamar, e o dia em que houver um terceiro pedaço seriam três.
	miolo := verbeteDoElo(grupo, s.livro.endereco)
	if q.Get("parte") == "aprimoramentos" && len(grupo.Magias) == 1 {
		miolo = osAprimoramentosDaMagia(grupo.Magias[0], s.livro.endereco)
	}

	sse := datastar.NewSSE(w, r)
	fragmento, err := renderFragmento(r.Context(), miolo)
	if err != nil {
		return
	}
	_ = sse.PatchElements(fragmento)
}
