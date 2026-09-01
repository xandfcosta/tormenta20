package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// As rotas da MESA DO MESTRE (ALE-257).
//
// O prefixo é `/mestre/` e não `/mesa/` — a razão está no
// cabeçalho do `piloto_bestiario.templ`: `mesa` já nomeia a sessão ao vivo
// desde a fatia 1, e uma palavra com dois sentidos no mesmo espaço de endereço
// é o que o glossário existe para impedir.
func (s *Server) rotasDoMestre(r chi.Router) {
	// `/mestre` sozinho não é uma tela: a trilha sempre tem uma ferramenta em
	// cena. Ele leva à primeira, que é a mesma que a SPA abre.
	r.Get("/mestre", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/mestre/bestiario", http.StatusSeeOther)
	})
	r.Get("/mestre/bestiario", s.handleBestiario)
	r.Post("/mestre/bestiario/tipo/{tipo}", s.handleBestiarioTipo)
	// CADA CATÁLOGO tem endereço próprio desde a ALE-264: eles viraram paradas do
	// trilho, e parada de trilho é uma cena. `/mestre/condicoes` em vez de
	// `/mestre/catalogos?aba=condicoes` — o mesmo handler, com a aba vindo
	// do CAMINHO.
	//
	// O laço sobre `abasDoAcervo` e não nove linhas escritas: o catálogo que
	// entrar amanhã ganha rota sozinho, e uma lista de rotas à mão é a que fica
	// para trás em silêncio.
	for _, aba := range abasDoAcervo {
		r.Get("/mestre/"+aba.ID, s.handleCatalogos)
	}
	// O endereço VELHO continua respondendo, redirecionando: ele foi o único por
	// duas fatias desta issue, e pode estar colado no chat de alguma mesa.
	r.Get("/mestre/catalogos", s.handleCatalogosVelho)
	r.Get("/mestre/encontros", s.handleEncontros)
	r.Post("/mestre/encontros/adicionar/{id}", s.handleEncontroAdicionar)
	r.Post("/mestre/encontros/mais/{id}", s.handleEncontroAdicionar)
	r.Post("/mestre/encontros/menos/{id}", s.handleEncontroMenos)
	r.Post("/mestre/encontros/remover/{id}", s.handleEncontroRemover)
	r.Get("/mestre/improviso", s.handleImproviso)
	// A ferramenta DESCONHECIDA cai na primeira, e não em 404.
	//
	// Porte de comportamento: a `/gm/$tool` da SPA validava o slug e redirigia,
	// com o comentário "uma URL digitada à mão ou velha aterrissa na primeira
	// ferramenta em vez de num palco em branco". Com a virada, quem encaminha
	// não valida mais — se o servidor devolvesse 404, um link velho de mestre
	// viraria página de erro em vez de abrir a Mesa.
	//
	// No chi o segmento ESTÁTICO ganha do parâmetro, então as quatro rotas
	// acima continuam sendo as que atendem; esta só recolhe o resto.
	r.Get("/mestre/{ferramenta}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/mestre/bestiario", http.StatusSeeOther)
	})
	r.Post("/mestre/improviso/{tabela}", s.handleImprovisoRola)
	r.Post("/mestre/improviso/{tabela}/limpar", s.handleImprovisoLimpa)
}

// handleImproviso desenha a cena com os históricos que vieram nos sinais.
func (s *Server) handleImproviso(w http.ResponseWriter, r *http.Request) {
	s.respondeImproviso(w, r, "")
}

// handleImprovisoRola rola UMA tabela e empilha o resultado no histórico dela.
func (s *Server) handleImprovisoRola(w http.ResponseWriter, r *http.Request) {
	s.respondeImproviso(w, r, chi.URLParam(r, "tabela"))
}

// handleImprovisoLimpa zera o histórico de UMA tabela.
//
// Zera só a dela, e não as quatro: as tabelas são independentes, e limpar a
// ruína não pode levar junto o evento de perseguição que o mestre acabou de
// tirar.
func (s *Server) handleImprovisoLimpa(w http.ResponseWriter, r *http.Request) {
	s.respondeImproviso(w, r, "limpar:"+chi.URLParam(r, "tabela"))
}

// respondeImproviso é o caminho único das seis rotas.
//
// `tabela` vazio significa "só redesenhe" — é a carga fria e o campo de salas.
// Com tabela, rola e empilha ANTES de montar a cena, porque o histórico é o que
// a cena desenha.
func (s *Server) respondeImproviso(w http.ResponseWriter, r *http.Request, tabela string) {
	v := improvisoDoPedido(r)

	if alvo, achou := strings.CutPrefix(tabela, "limpar:"); achou {
		if _, conhecida := asRolagens[alvo]; !conhecida {
			http.Error(w, "tabela de improviso desconhecida: "+alvo, http.StatusBadRequest)
			return
		}
		v = zeraTabela(v, alvo)
		tabela = ""
	}
	if tabela != "" {
		rolar, ok := asRolagens[tabela]
		if !ok {
			// Tabela inventada é 400 e não silêncio: a rota é montada a partir
			// da própria lista, então um nome errado aqui só chega por URL
			// digitada à mão — e devolver a cena intacta faria parecer que o
			// botão não funciona.
			http.Error(w, "tabela de improviso desconhecida: "+tabela, http.StatusBadRequest)
			return
		}
		sorteado, err := rolar()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		v = empilhaEm(v, tabela, sorteado)
	}

	pronta := carregaImproviso(v)

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := renderFragmento(r.Context(), cenaDoImproviso(pronta))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.escrevePagina(w, r, http.StatusOK, paginaPiloto{
		Titulo:        "Improviso · Mesa do Mestre · Tormenta 20",
		Forma:         cascaDensa,
		Voltar:        "/",
		VoltarRotulo:  "Hub",
		TituloVisivel: "Mesa do Mestre",
	}, mesaDoMestre("improviso", cenaDoImproviso(pronta)))
}

// asRolagens liga o nome da rota à função que rola. A tela e a rota leem a
// MESMA tabela, então um nome novo aparece nos dois lugares ou em nenhum.
var asRolagens = map[string]func() (sorteio, error){
	"ruina":       rolaRuina,
	"perseguicao": rolaPerseguicao,
	"recompensa":  rolaRecompensa,
	"ideias":      rolaIdeia,
}

// zeraTabela apaga o histórico de uma tabela e deixa as outras três intactas.
func zeraTabela(v improvisoView, tabela string) improvisoView {
	switch tabela {
	case "ruina":
		v.Ruina = nil
	case "perseguicao":
		v.Perseguicao = nil
	case "recompensa":
		v.Recompensa = nil
	case "ideias":
		v.Ideias = nil
	}
	return v
}

func empilhaEm(v improvisoView, tabela string, s sorteio) improvisoView {
	switch tabela {
	case "ruina":
		v.Ruina = empilha(v.Ruina, s)
	case "perseguicao":
		v.Perseguicao = empilha(v.Perseguicao, s)
	case "recompensa":
		v.Recompensa = empilha(v.Recompensa, s)
	case "ideias":
		v.Ideias = empilha(v.Ideias, s)
	}
	return v
}

// improvisoDoPedido lê os quatro históricos e o número de salas dos SINAIS.
//
// Aqui não há caminho pela URL, e é diferente das outras cenas de propósito: um
// histórico de rolagens não é endereço — ninguém cola "os cinco dados que eu
// tirei" no chat da mesa, e pôr isso na URL só encheria o histórico do
// navegador a cada clique no botão de rolar.
func improvisoDoPedido(r *http.Request) improvisoView {
	sinais := struct {
		Ruina       []sorteio `json:"ruina"`
		Perseguicao []sorteio `json:"perseguicao"`
		Recompensa  []sorteio `json:"recompensa"`
		Ideias      []sorteio `json:"ideias"`
		Salas       *int      `json:"salas"`
	}{}
	v := improvisoView{Salas: salasPadrao}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return v
	}
	v.Ruina, v.Perseguicao = sinais.Ruina, sinais.Perseguicao
	v.Recompensa, v.Ideias = sinais.Recompensa, sinais.Ideias
	if sinais.Salas != nil {
		v.Salas = *sinais.Salas
	}
	return v
}

// handleEncontros serve os dois casos numa rota. Sem autorização própria: o
// bestiário é o LIVRO, e o rascunho vive no navegador de quem monta.
func (s *Server) handleEncontros(w http.ResponseWriter, r *http.Request) {
	s.respondeEncontro(w, r, nil)
}

// handleEncontroAdicionar serve o "acrescentar" do painel de busca E o `[+]` da
// linha, porque a álgebra é a MESMA: `acrescenta` sobe a contagem quando a
// criatura já está no encontro. Duas rotas para uma função é o que evita a
// tela ter dois caminhos que podem divergir.
func (s *Server) handleEncontroAdicionar(w http.ResponseWriter, r *http.Request) {
	s.respondeEncontro(w, r, acrescenta)
}

func (s *Server) handleEncontroMenos(w http.ResponseWriter, r *http.Request) {
	s.respondeEncontro(w, r, diminui)
}

func (s *Server) handleEncontroRemover(w http.ResponseWriter, r *http.Request) {
	s.respondeEncontro(w, r, removeLinha)
}

// respondeEncontro lê o rascunho, aplica UM gesto e devolve a cena.
//
// A álgebra chega como função porque as quatro rotas só diferem nisso, e o
// resto — ler sinais, recalcular, remendar — é idêntico. Sem o parâmetro,
// seriam quatro cópias do mesmo handler, que é onde uma delas esquece de
// recalcular.
func (s *Server) respondeEncontro(
	w http.ResponseWriter, r *http.Request,
	gesto func([]linhaDoEncontro, string) []linhaDoEncontro,
) {
	nivel, grupo, linhas, busca := rascunhoDoPedido(r)
	if gesto != nil {
		linhas = gesto(linhas, chi.URLParam(r, "id"))
	}
	v := carregaEncontros(nivel, grupo, linhas, busca)

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := renderFragmento(r.Context(), cenaDosEncontros(v))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.escrevePagina(w, r, http.StatusOK, paginaPiloto{
		Titulo:        "Encontros · Mesa do Mestre · Tormenta 20",
		Forma:         cascaDensa,
		Voltar:        "/",
		VoltarRotulo:  "Hub",
		TituloVisivel: "Mesa do Mestre",
	}, mesaDoMestre("encontros", cenaDosEncontros(v)))
}

// rascunhoDoPedido lê o encontro da URL na carga fria e dos SINAIS no remendo.
//
// A URL é o caminho do LINK COPIADO: `?nivel=3&grupo=4&c=goblin:4,ogro:1`. Os
// sinais são o caminho de todo o resto, e vencem quando existem — eles são o
// que o mestre acabou de clicar.
func rascunhoDoPedido(r *http.Request) (int, int, []linhaDoEncontro, string) {
	q := r.URL.Query()
	nivel := numeroDaURL(q.Get("nivel"), nivelPadrao)
	grupo := numeroDaURL(q.Get("grupo"), grupoPadrao)
	linhas := linhasDaURL(q.Get("c"))
	busca := q.Get("busca")

	sinais := struct {
		Nivel         *int               `json:"nivel"`
		Grupo         *int               `json:"grupo"`
		Encontro      *[]linhaDoEncontro `json:"encontro"`
		BuscaCriatura *string            `json:"buscaCriatura"`
	}{}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return nivel, grupo, linhas, busca
	}
	if sinais.Nivel != nil {
		nivel = *sinais.Nivel
	}
	if sinais.Grupo != nil {
		grupo = *sinais.Grupo
	}
	if sinais.Encontro != nil {
		linhas = *sinais.Encontro
	}
	if sinais.BuscaCriatura != nil {
		busca = *sinais.BuscaCriatura
	}
	return nivel, grupo, linhas, busca
}

func numeroDaURL(bruto string, padrao int) int {
	n, err := strconv.Atoi(strings.TrimSpace(bruto))
	if err != nil {
		return padrao
	}
	return n
}

// handleCatalogos serve os dois casos numa rota, como as outras cenas.
//
// Sem autorização própria e pelo mesmo motivo do bestiário: o catálogo é o
// LIVRO, igual para todo mundo. O `requirePagina` do grupo já exige sessão.
// handleCatalogosVelho manda o endereço antigo para a cena da aba pedida,
// preservando busca e entrada — um redirecionamento que perde a consulta
// devolveria a pessoa a uma tela que não é a que ela pediu.
func (s *Server) handleCatalogosVelho(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	destino := "/mestre/" + abaConhecida(q.Get("aba"))
	q.Del("aba")
	if resto := q.Encode(); resto != "" {
		destino += "?" + resto
	}
	http.Redirect(w, r, destino, http.StatusMovedPermanently)
}

func (s *Server) handleCatalogos(w http.ResponseWriter, r *http.Request) {
	v := carregaCatalogos(criteriosDoPedidoDoAcervo(r), s.livro.endereco)

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := renderFragmento(r.Context(), cenaDosCatalogos(v))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.escrevePagina(w, r, http.StatusOK, paginaPiloto{
		Titulo:        rotuloDaAba(v.Aba) + " · Mesa do Mestre · Tormenta 20",
		Forma:         cascaDensa,
		Voltar:        "/",
		VoltarRotulo:  "Hub",
		TituloVisivel: "Mesa do Mestre",
	}, mesaDoMestre(v.Aba, cenaDosCatalogos(v)))
}

// filtrosDaURL lê os crachás que a consulta pede, só os DA ABA: `?circulo=2&circulo=3`
// na cena das condições não filtra nada, e aceitá-lo faria a cena carregar um
// estado que ela não sabe desenhar.
func filtrosDaURL(q url.Values, aba string) map[string][]string {
	fora := map[string][]string{}
	for _, f := range filtrosDaAba(abaConhecida(aba)) {
		if valores := q[f.Chave]; len(valores) > 0 {
			fora[f.Chave] = valores
		}
	}
	return fora
}

// criteriosDoPedidoDoAcervo lê a busca, a aba e a ENTRADA da URL na carga fria e
// dos SINAIS quando o Datastar chama — mesma decisão das outras cenas, e é ela
// que faz `?busca=fogo`, `?aba=magias` e `?entrada=medo` serem endereços que se
// recarregam.
//
// `entrada` NÃO vem de sinal, e é deliberado: ela é um endereço para UM verbete,
// escrito por um elo ou colado por alguém. Vindo de sinal, ela sobreviveria à
// próxima tecla digitada na busca e a cena ficaria presa num verbete só.
func criteriosDoPedidoDoAcervo(r *http.Request) criteriosDoAcervo {
	q := r.URL.Query()
	// A ABA vem do CAMINHO desde que cada catálogo virou uma cena
	// (`/mestre/condicoes`). A consulta continua sendo lida para o
	// endereço velho e para quem digitar `?aba=` à mão.
	aba := path.Base(r.URL.Path)
	if abaConhecida(aba) != aba {
		aba = q.Get("aba")
	}
	c := criteriosDoAcervo{
		Busca: q.Get("busca"), Aba: aba, Entrada: q.Get("entrada"),
		Filtros: filtrosDaURL(q, aba),
	}

	// Os FILTROS vêm num mapa cru e não numa struct: as chaves dependem da aba
	// (`circulo` só existe em magias), e uma struct com os seis campos faria
	// toda cena declarar os filtros das outras.
	var todos map[string]json.RawMessage
	if err := datastar.ReadSignals(r, &todos); err == nil {
		for _, f := range filtrosDaAba(abaConhecida(aba)) {
			var valores []string
			if bruto, tem := todos[f.Chave]; tem && json.Unmarshal(bruto, &valores) == nil {
				c.Filtros[f.Chave] = valores
			}
		}
	}

	sinais := struct {
		Busca *string `json:"busca"`
		Aba   *string `json:"aba"`
	}{}
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return c
	}
	// Ponteiro para separar "não veio" de "veio vazio": busca APAGADA é valor
	// legítimo, e tratá-la como ausente ressuscitaria o texto da URL.
	if sinais.Busca != nil {
		c.Busca = *sinais.Busca
	}
	if sinais.Aba != nil {
		c.Aba = *sinais.Aba
	}
	return c
}

// rotaDoBestiarioDoMestre é a base da CENA do mestre. O painel da Mesa tem a
// sua, e as duas dividem o mesmo desenho — ver `bestiarioView.Base`.
const rotaDoBestiarioDoMestre = "/mestre/bestiario"

// handleBestiario serve os DOIS casos numa rota, como a cena de campanhas: a
// carga fria devolve a página inteira e o Datastar recebe só o remendo da cena.
//
// Não há autorização própria aqui, e isso é deliberado: o bestiário é o LIVRO,
// igual para todo mundo. O `requirePagina` do grupo já exige sessão, que é o
// único requisito — não há dado de campanha nem de personagem nesta tela.
func (s *Server) handleBestiario(w http.ResponseWriter, r *http.Request) {
	v := s.carregaBestiario(criteriosDoPedido(r))

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
		Voltar: "/",
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
	// `ReadSignals` ANTES do `NewSSE`: depois da resposta começar, o corpo do
	// pedido já não se lê.
	criterios := criteriosDoPedido(r)
	// A recusa vem ANTES da resposta começar, senão o 400 chega depois dos
	// cabeçalhos de SSE e o cliente vê um stream vazio em vez de um erro.
	tipos, err := alternaOTipo(criterios.Tipos, chi.URLParam(r, "tipo"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	criterios.Tipos = tipos
	sse := datastar.NewSSE(w, r)

	fragmento, err := renderFragmento(r.Context(), cenaDoBestiario(s.carregaBestiario(criterios)))
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
	// Abrir: este pedido veio de um clique numa LINHA, e a ficha tem de nascer
	// aberta com a criatura certa.
	Abrir bool
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
		// Só da URL, e NUNCA dos sinais: os sinais viajam em todo pedido desta
		// rota — busca, filtro de tipo, escolha de criatura — e não separariam
		// "cliquei numa linha" de "digitei uma letra".
		Abrir: q.Get("abrir") != "",
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
func (s *Server) carregaBestiario(c criteriosDoBestiario) bestiarioView {
	v := carregaBestiarioDe(rotaDoBestiarioDoMestre, s.livro.endereco, c.Busca, c.Tipos, c.NDMin, c.NDMax, c.Escolhida)
	v.Abrir = c.Abrir
	return v
}
