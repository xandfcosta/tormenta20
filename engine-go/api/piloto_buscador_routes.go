package api

import (
	"net/http"
	"t20engine/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// A rota do BUSCADOR DO LIVRO (ALE-264).
//
// Ela devolve SÓ o remendo dos achados: a caixa inteira é desenhada uma vez pela
// casca e nunca é trocada. É o que mantém o `data-bind` do campo fora do que o
// servidor reescreve — remendar o campo junto apagaria o que a pessoa está
// digitando a cada 200ms.
//
// Atrás do `requirePagina` como o resto: o livro é igual para todo mundo, e o
// único requisito é ter entrado.

// rotaDoBuscador é o endereço PÚBLICO, que é o que o `@get` do navegador pede. O
// `chi` registra sem o `/`, que o `buildMux` tira antes de entregar aqui.
const rotaDoBuscador = "/buscador"

func (s *Server) BookSearchRoutes(r chi.Router) {
	r.Get("/buscador", s.handleBuscador)
}

func (s *Server) handleBuscador(w http.ResponseWriter, r *http.Request) {
	// O termo é lido ANTES do `NewSSE`, e a ordem é regra da biblioteca: o
	// `ReadSignals` de um POST consome o CORPO, e o próprio datastar-go devolve
	// "are you sure you created the SSE ***AFTER*** the ReadSignals?" quando é
	// tarde demais. Num GET os sinais vêm na consulta e a ordem não morde — que
	// é justamente o que faria o defeito nascer no dia da mudança para POST.
	v := buscaNoLivro(termoDoBuscador(r))
	sse := datastar.NewSSE(w, r)
	fragmento, err := ui.RenderFragment(r.Context(), achadosDoBuscador(v))
	if err != nil {
		return
	}
	_ = sse.PatchElements(fragmento)
}

// termoDoBuscador lê o que foi digitado: do SINAL quando o Datastar chama, da
// URL quando alguém abre o endereço à mão.
//
// Ponteiro para separar "não veio" de "veio vazio", como o `criteriosDoAcervo`:
// apagar a busca é gesto legítimo, e tratá-lo como ausência ressuscitaria o
// termo anterior.
func termoDoBuscador(r *http.Request) string {
	sinais := struct {
		Buscador *string `json:"buscador"`
	}{}
	if err := datastar.ReadSignals(r, &sinais); err != nil || sinais.Buscador == nil {
		return r.URL.Query().Get("busca")
	}
	return *sinais.Buscador
}
