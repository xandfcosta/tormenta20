package master

import (
	"net/http"
	"slices"
	"strings"
	"t20engine/domain/book"
	"t20engine/serve/web/routes"

	"t20engine/serve/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// As rotas e os handlers do BESTIÁRIO.
//
// O REGISTRO das trinta rotas continua num lugar só, no `routes.go`: é ele que o
// `api` chama, e espalhá-lo faria a cena ter quatro portas de entrada. O que
// mora aqui são os handlers desta ferramenta — arquivo é unidade de
// RESPONSABILIDADE e de conflito de merge, não de leitura.

// O endereço desta cena mora em `web/routes` porque o buscador linka para ela, e
// o critério de entrada de lá é estreito: só endereço citado de OUTRA cena.
// O critério de entrada de lá é estreito — só endereço citado de OUTRA cena.

// handleBestiary serve os DOIS casos numa rota, como a cena de campanhas: a
// carga fria devolve a página inteira e o Datastar recebe só o remendo da cena.
//
// Não há autorização própria aqui, e isso é deliberado: o bestiário é o LIVRO,
// igual para todo mundo. O `requirePage` do grupo já exige sessão, que é o
// único requisito — não há dado de campanha nem de personagem nesta tela.
func (s Scene) handleBestiary(w http.ResponseWriter, r *http.Request) {
	v := s.loadBestiary(BestiaryCriteriaFromRequest(r))

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragment, err := ui.RenderFragment(r.Context(), bestiaryScene(v))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragment)
		return
	}

	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: "Bestiário · Mesa do Mestre · Tormenta 20",
		Forma:  ui.ShellDense,
		Voltar: "/",
		// "Hub" e não a seta genérica: a Mesa do Mestre não é filha óbvia de
		// nenhuma tela, e nomear o destino é o que a folha de especificação pede.
		VoltarRotulo: "Hub",
		// O título VISÍVEL é o da TELA, e a cena desenha o próprio "Bestiário"
		// como `h2` — a trilha troca a ferramenta, não a tela. É por isso que o
		// `<title>` carrega os dois: quem tem seis abas abertas precisa saber
		// qual ferramenta está em cada uma.
		TituloVisivel: "Mesa do Mestre",
	}, masterBody("bestiario", bestiaryScene(v)))
}

// handleBestiaryType liga ou desliga UM crachá de tipo.
//
// Remendo e não navegação, pela mesma razão do interruptor de regra da crônica:
// o mestre está lendo a lista, e recarregar a página perderia a posição dela.
//
// O conjunto de tipos vive nos SINAIS e não no servidor, porque não há o que
// gravar — é um filtro de leitura, não um estado da campanha. O que a rota faz
// é a álgebra do conjunto e devolver a cena inteira com os sinais novos.
func (s Scene) handleBestiaryType(w http.ResponseWriter, r *http.Request) {
	// `ReadSignals` ANTES do `NewSSE`: depois da resposta começar, o corpo do
	// pedido já não se lê.
	criteria := BestiaryCriteriaFromRequest(r)
	// A recusa vem ANTES da resposta começar, senão o 400 chega depois dos
	// cabeçalhos de SSE e o cliente vê um stream vazio em vez de um erro.
	kinds, err := ToggleType(criteria.Types, chi.URLParam(r, "tipo"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	criteria.Types = kinds
	sse := datastar.NewSSE(w, r)

	fragment, err := ui.RenderFragment(r.Context(), bestiaryScene(s.loadBestiary(criteria)))
	if err != nil {
		return
	}
	_ = sse.PatchElements(fragment)
}

// BestiaryCriteria são os cinco valores que definem a cena.
type BestiaryCriteria struct {
	Term   string
	Types  []string
	CRMin  float64
	CRMax  float64
	Chosen string
	// Abrir: este pedido veio de um clique numa LINHA, e a ficha tem de nascer
	// aberta com a criatura certa.
	Open bool
}

// BestiaryCriteriaFromRequest lê os critérios da URL na carga fria e dos SINAIS quando o
// Datastar chama.
//
// Ler os dois no mesmo lugar é o que faz `?busca=ogro&nd-max=3` ser um ENDEREÇO
// que se recarrega e se cola no chat da mesa — a mesma decisão do
// `termFromRequest` dos personagens e do `filterFromRequest` das campanhas.
//
// Os sinais VENCEM a URL quando existem, e não o contrário: quando o Datastar
// chama, os sinais são o que o mestre acabou de digitar, e a URL é o que ele
// digitou da última vez que a página carregou.
func BestiaryCriteriaFromRequest(r *http.Request) BestiaryCriteria {
	q := r.URL.Query()
	c := BestiaryCriteria{
		Term:   q.Get("busca"),
		Chosen: q.Get("criatura"),
		Types:  typesFromURL(q.Get("tipos")),
		// Só da URL, e NUNCA dos sinais: os sinais viajam em todo pedido desta
		// rota — busca, filtro de tipo, escolha de criatura — e não separariam
		// "cliquei numa linha" de "digitei uma letra".
		Open: q.Get("abrir") != "",
	}
	c.CRMin, c.CRMax = book.CRRange(q.Get("nd-min"), q.Get("nd-max"))

	signals := struct {
		Term     *string   `json:"search"`
		Types    *[]string `json:"tipos"`
		CRMin    *float64  `json:"ndMin"`
		CRMax    *float64  `json:"ndMax"`
		Creature *string   `json:"creature"`
	}{}
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return c
	}
	// Ponteiro em cada campo para separar "o sinal não veio" de "o sinal veio
	// vazio": uma busca APAGADA é um valor legítimo, e tratá-la como ausente
	// faria o texto antigo da URL ressuscitar no primeiro remendo.
	if signals.Term != nil {
		c.Term = *signals.Term
	}
	if signals.Creature != nil {
		c.Chosen = *signals.Creature
	}
	if signals.Types != nil {
		c.Types = knownTypes(*signals.Types)
	}
	if signals.CRMin != nil || signals.CRMax != nil {
		c.CRMin, c.CRMax = numericCRRange(signals.CRMin, signals.CRMax, c.CRMin, c.CRMax)
	}
	return c
}

// typesFromURL lê `?tipos=animal,planar`. Vírgula e não repetição do parâmetro
// porque é o que cabe numa URL que alguém digita.
func typesFromURL(raw string) []string {
	if raw == "" {
		return nil
	}
	return knownTypes(strings.Split(raw, ","))
}

// knownTypes descarta o que o catálogo não tem, venha da URL ou do sinal.
//
// Descartar e não recusar: um tipo inventado na lista é um filtro que não casa
// com nada, e esvaziar a tela por causa de uma vírgula sobrando seria punir o
// mestre por um erro de digitação. A recusa dura fica no POST do crachá, que é
// onde alguém está agindo em vez de navegando.
func knownTypes(raw []string) []string {
	var outside []string
	for _, t := range raw {
		if t = strings.TrimSpace(t); slices.Contains(book.CreatureTypes, t) && !slices.Contains(outside, t) {
			outside = append(outside, t)
		}
	}
	return outside
}

// numericCRRange aperta os números que vieram dos SINAIS, onde eles já são
// número e não texto. Mesma faixa do livro, mesma razão do `book.CRRange`.
func numericCRRange(min, max *float64, standardMin, standardMax float64) (float64, float64) {
	exitMin, exitMax := standardMin, standardMax
	if min != nil {
		exitMin = clampCR(*min, book.CRMin)
	}
	if max != nil {
		exitMax = clampCR(*max, book.CRMax)
	}
	return exitMin, exitMax
}

func clampCR(n, standard float64) float64 {
	if n < book.CRMin || n > book.CRMax {
		return standard
	}
	return n
}

// loadBestiary com os critérios já lidos. Envelope fino sobre a função da
// camada de dados, para o handler não repetir a ordem dos cinco argumentos.
func (s Scene) loadBestiary(c BestiaryCriteria) BestiaryView {
	v := LoadBestiaryFrom(routes.MasterBestiary, s.deps.BookAddress(), c.Term, c.Types, c.CRMin, c.CRMax, c.Chosen)
	v.Open = c.Open
	return v
}
