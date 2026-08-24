package api

import (
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// As rotas da MESA DO MESTRE (ALE-257).
//
// O prefixo é `/piloto/mestre/` e não `/piloto/mesa/` — a razão está no
// cabeçalho do `piloto_bestiario.templ`: `mesa` já nomeia a sessão ao vivo
// desde a fatia 1, e uma palavra com dois sentidos no mesmo espaço de endereço
// é o que o glossário existe para impedir.
func (s *Server) rotasDoMestre(r chi.Router) {
	// `/mestre` sozinho não é uma tela: a trilha sempre tem uma ferramenta em
	// cena. Ele leva à primeira, que é a mesma que a SPA abre.
	r.Get("/mestre", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/piloto/mestre/bestiario", http.StatusSeeOther)
	})
	r.Get("/mestre/bestiario", s.handleBestiario)
	r.Post("/mestre/bestiario/tipo/{tipo}", s.handleBestiarioTipo)
}

// handleBestiario serve os DOIS casos numa rota, como a cena de campanhas: a
// carga fria devolve a página inteira e o Datastar recebe só o remendo da cena.
//
// Não há autorização própria aqui, e isso é deliberado: o bestiário é o LIVRO,
// igual para todo mundo. O `requirePagina` do grupo já exige sessão, que é o
// único requisito — não há dado de campanha nem de personagem nesta tela.
func (s *Server) handleBestiario(w http.ResponseWriter, r *http.Request) {
	v := carregaBestiario(criteriosDoPedido(r))

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := renderFragmento(r.Context(), cenaDoBestiario(v))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.escrevePagina(w, r, http.StatusOK, paginaPiloto{
		Titulo: "Bestiário · Mesa do Mestre · Tormenta 20",
		Forma:  cascaDensa,
		Voltar: "/piloto/",
		// "Hub" e não a seta genérica: a Mesa do Mestre não é filha óbvia de
		// nenhuma tela, e nomear o destino é o que a folha de especificação pede.
		VoltarRotulo: "Hub",
		// O título VISÍVEL é o da TELA, e a cena desenha o próprio "Bestiário"
		// como `h2` — a trilha troca a ferramenta, não a tela. É por isso que o
		// `<title>` carrega os dois: quem tem seis abas abertas precisa saber
		// qual ferramenta está em cada uma.
		TituloVisivel: "Mesa do Mestre",
	}, mesaDoMestre("bestiario", cenaDoBestiario(v)))
}

// handleBestiarioTipo liga ou desliga UM crachá de tipo.
//
// Remendo e não navegação, pela mesma razão do interruptor de regra da crônica:
// o mestre está lendo a lista, e recarregar a página perderia a posição dela.
//
// O conjunto de tipos vive nos SINAIS e não no servidor, porque não há o que
// gravar — é um filtro de leitura, não um estado da campanha. O que a rota faz
// é a álgebra do conjunto e devolver a cena inteira com os sinais novos.
func (s *Server) handleBestiarioTipo(w http.ResponseWriter, r *http.Request) {
	tipo := chi.URLParam(r, "tipo")
	// Tipo que o catálogo não conhece é recusado ANTES de qualquer coisa: a URL
	// é editável à mão, e um tipo inventado no conjunto filtraria tudo fora e a
	// tela leria "Nenhuma criatura casa com os filtros" sem explicar por quê.
	if !slices.Contains(tiposDeCriatura, tipo) {
		http.Error(w, "tipo de criatura desconhecido: "+tipo, http.StatusBadRequest)
		return
	}
	// `ReadSignals` ANTES do `NewSSE`: depois da resposta começar, o corpo do
	// pedido já não se lê.
	criterios := criteriosDoPedido(r)
	sse := datastar.NewSSE(w, r)

	if i := slices.Index(criterios.Tipos, tipo); i >= 0 {
		criterios.Tipos = slices.Delete(slices.Clone(criterios.Tipos), i, i+1)
	} else {
		criterios.Tipos = append(slices.Clone(criterios.Tipos), tipo)
	}

	fragmento, err := renderFragmento(r.Context(), cenaDoBestiario(carregaBestiario(criterios)))
	if err != nil {
		return
	}
	_ = sse.PatchElements(fragmento)
}

// criteriosDoBestiario são os cinco valores que definem a cena.
type criteriosDoBestiario struct {
	Busca     string
	Tipos     []string
	NDMin     float64
	NDMax     float64
	Escolhida string
}

// criteriosDoPedido lê os critérios da URL na carga fria e dos SINAIS quando o
// Datastar chama.
//
// Ler os dois no mesmo lugar é o que faz `?busca=ogro&nd-max=3` ser um ENDEREÇO
// que se recarrega e se cola no chat da mesa — a mesma decisão do
// `buscaDoPedido` dos personagens e do `filtroDoPedido` das campanhas.
//
// Os sinais VENCEM a URL quando existem, e não o contrário: quando o Datastar
// chama, os sinais são o que o mestre acabou de digitar, e a URL é o que ele
// digitou da última vez que a página carregou.
func criteriosDoPedido(r *http.Request) criteriosDoBestiario {
	q := r.URL.Query()
	c := criteriosDoBestiario{
		Busca:     q.Get("busca"),
		Escolhida: q.Get("criatura"),
		Tipos:     tiposDaURL(q.Get("tipos")),
	}
	c.NDMin, c.NDMax = faixaDeND(q.Get("nd-min"), q.Get("nd-max"))

	sinais := struct {
		Busca    *string   `json:"busca"`
		Tipos    *[]string `json:"tipos"`
		NDMin    *float64  `json:"ndMin"`
		NDMax    *float64  `json:"ndMax"`
		Criatura *string   `json:"criatura"`
	}{}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return c
	}
	// Ponteiro em cada campo para separar "o sinal não veio" de "o sinal veio
	// vazio": uma busca APAGADA é um valor legítimo, e tratá-la como ausente
	// faria o texto antigo da URL ressuscitar no primeiro remendo.
	if sinais.Busca != nil {
		c.Busca = *sinais.Busca
	}
	if sinais.Criatura != nil {
		c.Escolhida = *sinais.Criatura
	}
	if sinais.Tipos != nil {
		c.Tipos = tiposConhecidos(*sinais.Tipos)
	}
	if sinais.NDMin != nil || sinais.NDMax != nil {
		c.NDMin, c.NDMax = faixaDeNDNumerica(sinais.NDMin, sinais.NDMax, c.NDMin, c.NDMax)
	}
	return c
}

// tiposDaURL lê `?tipos=animal,planar`. Vírgula e não repetição do parâmetro
// porque é o que cabe numa URL que alguém digita.
func tiposDaURL(bruto string) []string {
	if bruto == "" {
		return nil
	}
	return tiposConhecidos(strings.Split(bruto, ","))
}

// tiposConhecidos descarta o que o catálogo não tem, venha da URL ou do sinal.
//
// Descartar e não recusar: um tipo inventado na lista é um filtro que não casa
// com nada, e esvaziar a tela por causa de uma vírgula sobrando seria punir o
// mestre por um erro de digitação. A recusa dura fica no POST do crachá, que é
// onde alguém está agindo em vez de navegando.
func tiposConhecidos(brutos []string) []string {
	var fora []string
	for _, t := range brutos {
		if t = strings.TrimSpace(t); slices.Contains(tiposDeCriatura, t) && !slices.Contains(fora, t) {
			fora = append(fora, t)
		}
	}
	return fora
}

// faixaDeNDNumerica aperta os números que vieram dos SINAIS, onde eles já são
// número e não texto. Mesma faixa do livro, mesma razão do `faixaDeND`.
func faixaDeNDNumerica(min, max *float64, padraoMin, padraoMax float64) (float64, float64) {
	saiMin, saiMax := padraoMin, padraoMax
	if min != nil {
		saiMin = apertaND(*min, ndMinimo)
	}
	if max != nil {
		saiMax = apertaND(*max, ndMaximo)
	}
	return saiMin, saiMax
}

func apertaND(n, padrao float64) float64 {
	if n < ndMinimo || n > ndMaximo {
		return padrao
	}
	return n
}

// carregaBestiario com os critérios já lidos. Envelope fino sobre a função da
// camada de dados, para o handler não repetir a ordem dos cinco argumentos.
func carregaBestiario(c criteriosDoBestiario) bestiarioView {
	return carregaBestiarioDe(c.Busca, c.Tipos, c.NDMin, c.NDMax, c.Escolhida)
}
