package finder

import (
	"net/http"
	"t20engine/serve/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// A rota do BUSCADOR DO LIVRO.
//
// Ela devolve SÓ o remendo dos achados: a caixa inteira é desenhada uma vez pela
// casca e nunca é trocada. É o que mantém o `data-bind` do campo fora do que o
// servidor reescreve — remendar o campo junto apagaria o que a pessoa está
// digitando a cada 200ms.
//
// Atrás do `requirePage` como o resto: o livro é igual para todo mundo, e o
// único requisito é ter entrado.

// finderRoute é o endereço PÚBLICO, que é o que o `@get` do navegador pede.
//
// Ele NÃO vai para `web/routes`: o critério de lá é estreito de propósito — só
// endereço que uma cena cita de OUTRA —, e este é escrito pelos arquivos desta
// cena e por mais ninguém.
const finderRoute = "/buscador"

// ESTA CENA NÃO TEM PORTA.
//
// Ela declara ZERO dependências: lê o livro embutido pelo `book`, pontua com o
// `search`, desenha com o `web/ui` e linka pelo `web/routes` — quatro pacotes
// que já eram folha. Não há banco, não há sessão, não há casca a montar.
//
// Por isso `Routes` recebe só o roteador, e não existe `Deps` nem `Scene`. Uma
// interface vazia declarada "por simetria" seria cerimônia pura: o que dá valor
// à porta é ela DIZER o que a cena alcança do hospedeiro, e aqui a resposta é
// nada.
func Routes(r chi.Router) {
	r.Get(finderRoute, handleFinder)
}

func handleFinder(w http.ResponseWriter, r *http.Request) {
	// O termo é lido ANTES do `NewSSE`, e a ordem é regra da biblioteca: o
	// `ReadSignals` de um POST consome o CORPO, e o próprio datastar-go devolve
	// "are you sure you created the SSE ***AFTER*** the ReadSignals?" quando é
	// tarde demais. Num GET os sinais vêm na consulta e a ordem não morde — que
	// é justamente o que faria o defeito nascer no dia da mudança para POST.
	v := searchTheBook(finderTerm(r))
	sse := datastar.NewSSE(w, r)
	fragment, err := ui.RenderFragment(r.Context(), finderResults(v))
	if err != nil {
		return
	}
	_ = sse.PatchElements(fragment)
}

// finderTerm lê o que foi digitado: do SINAL quando o Datastar chama, da
// URL quando alguém abre o endereço à mão.
//
// Ponteiro para separar "não veio" de "veio vazio", como o `finderTerm`:
// apagar a busca é gesto legítimo, e tratá-lo como ausência ressuscitaria o
// termo anterior.
func finderTerm(r *http.Request) string {
	signals := struct {
		Searcher *string `json:"finder"`
	}{}
	if err := datastar.ReadSignals(r, &signals); err != nil || signals.Searcher == nil {
		return r.URL.Query().Get("busca")
	}
	return *signals.Searcher
}
