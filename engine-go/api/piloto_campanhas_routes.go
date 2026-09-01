package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
	"t20engine/web/ui"
)

// A rota da cena de CAMPANHAS (ALE-234).
//
// UMA rota serve os dois casos, e é isso que a mantém pequena: a carga fria
// devolve a página inteira, e a busca — que chega pelo mesmo `GET` com os
// sinais do Datastar — devolve só o remendo da cena. Quem distingue os dois é o
// cabeçalho `datastar-request`, que o cliente põe.

func (s *Server) CampaignRoutes(r chi.Router) {
	r.Get("/campanhas", s.handleCampanhas)
	r.Get("/campanhas/nova", s.handleCampanhaNova)
	r.Post("/campanhas/nova", s.handleCampanhaNovaPost)
	r.Get("/campanhas/entrar", s.handleCampanhaEntrar)
	r.Post("/campanhas/entrar", s.handleCampanhaEntrarPost)
	r.Get("/campanhas/{id}", s.handleCronica)
	r.Post("/campanhas/{id}/editar", s.handleCronicaEditar)
	r.Post("/campanhas/{id}/excluir", s.handleCronicaExcluir)
	r.Post("/campanhas/{id}/regras/{regra}", s.handleCronicaAlternarRegra)
}

func (s *Server) handleCampanhas(w http.ResponseWriter, r *http.Request) {
	busca, papel := filtroDoPedido(r)
	view, err := s.carregaCampanhas(r.Context(), currentUser(r), busca, papel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Pedido do Datastar: remendo. A cena inteira, e não só a lista, porque a
	// barra também muda — o chip de papel aceso e o texto da busca.
	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := renderFragmento(r.Context(), cenaDeCampanhas(view))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: "Campanhas · Tormenta 20",
		// `cascaNua`: esta cena desenha o próprio cabeçalho, porque ele carrega a
		// busca e os filtros. A casca densa poria um segundo `<h1>` acima.
		Forma: ui.ShellBare,
	}, cenaDeCampanhas(view))
}

// filtroDoPedido lê a busca e o papel, venham de onde vierem.
//
// Da URL na carga fria (`?busca=...`), e dos SINAIS quando o Datastar chama —
// ele os manda no `?datastar=` como JSON. Ler os dois no mesmo lugar é o que
// deixa a tela filtrada ser um endereço que se guarda: recarregar `?busca=anao`
// devolve exatamente o que estava.
func filtroDoPedido(r *http.Request) (busca, papel string) {
	q := r.URL.Query()
	busca, papel = q.Get("busca"), q.Get("papel")
	sinais := struct {
		Busca string `json:"busca"`
		Papel string `json:"papel"`
	}{}
	if err := datastar.ReadSignals(r, &sinais); err == nil {
		if sinais.Busca != "" || sinais.Papel != "" {
			busca, papel = sinais.Busca, sinais.Papel
		}
	}
	return busca, papel
}

// urlDeCampanhas monta o endereço que a cena representa, para o histórico do
// navegador acompanhar a busca.
func urlDeCampanhas(busca, papel string) string {
	q := url.Values{}
	if busca != "" {
		q.Set("busca", busca)
	}
	if papel != "" && papel != "todas" {
		q.Set("papel", papel)
	}
	if len(q) == 0 {
		return "/campanhas"
	}
	return "/campanhas?" + q.Encode()
}

// ── a folha em branco: abrir campanha (ALE-246) ──────────────────────────────

// handleCampanhaNova desenha o formulário vazio.
func (s *Server) handleCampanhaNova(w http.ResponseWriter, r *http.Request) {
	s.escreveFolhaNova(w, r, http.StatusOK, campanhaNovaView{})
}

// handleCampanhaNovaPost cria e vai para a crônica recém-aberta.
//
// A recusa REDESENHA a folha com o que foi digitado e o erro no campo — não
// redireciona. Redirecionar perderia o texto, e a descrição é o campo caro de
// reescrever. Status 422 e não 200 porque a resposta É uma recusa, e o
// navegador não trata os dois igual no histórico.
func (s *Server) handleCampanhaNovaPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.escreveFolhaNova(w, r, http.StatusBadRequest, campanhaNovaView{Aviso: avisoInterno})
		return
	}
	v := campanhaNovaView{
		Nome:      r.PostFormValue("name"),
		Descricao: r.PostFormValue("description"),
		Erros:     plataforma.FieldErrorMap{},
	}
	// A MESMA regra da rota JSON, e não uma cópia dela — ver `campanha_regras.go`.
	nome, err := nomeDeCampanha(v.Nome)
	if err != nil {
		v.Erros["name"] = []string{"O nome é obrigatório e cabe em 120 caracteres."}
	}
	descricao, errDesc := descricaoDeCampanha(&v.Descricao)
	if errDesc != nil {
		v.Erros["description"] = []string{"A descrição cabe em 2000 caracteres."}
	}
	if len(v.Erros) > 0 {
		s.escreveFolhaNova(w, r, http.StatusUnprocessableEntity, v)
		return
	}

	agora := plataforma.NowISO()
	c, err := s.queries.CreateCampaign(r.Context(), sqlcgen.CreateCampaignParams{
		Ownerid: currentUser(r).ID, Name: nome, Description: descricao,
		Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		v.Aviso = avisoInterno
		s.escreveFolhaNova(w, r, http.StatusInternalServerError, v)
		return
	}
	// 303 e não 302: depois de um POST, o `See Other` é o que garante que o
	// navegador siga com GET. Sem ele, recarregar a crônica reenviaria o
	// formulário e abriria uma segunda campanha igual.
	http.Redirect(w, r, "/campanhas/"+strconv.FormatInt(c.ID, 10), http.StatusSeeOther)
}

func (s *Server) escreveFolhaNova(w http.ResponseWriter, r *http.Request, status int, v campanhaNovaView) {
	s.WritePage(w, r, status, ui.Page{
		Titulo: "Abrir nova campanha",
		// `cascaDensa`: a tela da SPA usa o cabeçalho compacto com o "‹ Voltar",
		// e sem ele a folha nasce sem saída visível — o Esc existe, mas atalho
		// não é a única porta.
		Forma:  ui.ShellDense,
		Voltar: "/campanhas",
	}, campanhaNova(v))
}

// ── a carta de convite: entrar na mesa (ALE-249) ─────────────────────────────

// handleCampanhaEntrar desenha a carta, com o convite JÁ RESOLVIDO.
func (s *Server) handleCampanhaEntrar(w http.ResponseWriter, r *http.Request) {
	v, err := s.carregaCartaDeConvite(r.Context(), currentUser(r), r.URL.Query().Get("token"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.escreveCarta(w, r, http.StatusOK, v)
}

// handleCampanhaEntrarPost senta o herói à mesa.
//
// As sete travas são do `entrarNaMesa` e não daqui — a mesma função que a rota
// JSON usa. O que este manipulador faz é TRADUZIR cada recusa para uma frase em
// português no campo certo, que é trabalho de tela e não de regra.
func (s *Server) handleCampanhaEntrarPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.escreveCarta(w, r, http.StatusBadRequest, campanhaEntrarView{Aviso: avisoInterno})
		return
	}
	eu := currentUser(r)
	token := r.PostFormValue("token")
	v, err := s.carregaCartaDeConvite(r.Context(), eu, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	v.NumeroDigitado = r.PostFormValue("campaignId")

	// O id da campanha vem do CONVITE quando há um, e do campo quando não há.
	// Nunca dos dois: com convite na mão, um número digitado seria uma segunda
	// fonte para a mesma coisa, e a tela nem mostra o campo.
	campanhaID := v.CampanhaID
	if !v.TemConvite {
		n, erroNum := strconv.ParseInt(strings.TrimSpace(v.NumeroDigitado), 10, 64)
		if erroNum != nil || n <= 0 {
			v.Erros["campaignId"] = []string{"Informe o número da campanha."}
			s.escreveCarta(w, r, http.StatusUnprocessableEntity, v)
			return
		}
		campanhaID = n
	}

	heroiID, erroHeroi := strconv.ParseInt(r.PostFormValue("characterId"), 10, 64)
	if erroHeroi != nil {
		v.Erros["characterId"] = []string{"Escolha o herói que entra na mesa."}
		s.escreveCarta(w, r, http.StatusUnprocessableEntity, v)
		return
	}
	v.EscolhidoID = heroiID

	_, err = s.entrarNaMesa(r.Context(), pedidoDeEntrada{
		CampanhaID: campanhaID, PersonagemID: heroiID,
		Convite: token, Papel: "player", QuemPede: eu.ID,
	})
	if err != nil {
		v.Erros, v.Aviso = recusaDeEntrada(err)
		s.escreveCarta(w, r, http.StatusUnprocessableEntity, v)
		return
	}
	// 303, como a folha em branco: depois de um POST, recarregar a crônica não
	// pode reenviar o formulário.
	http.Redirect(w, r, "/campanhas/"+strconv.FormatInt(campanhaID, 10), http.StatusSeeOther)
}

// recusaDeEntrada traduz cada erro da regra para a frase que a pessoa lê.
//
// Uma frase por recusa, e não um "não foi possível entrar" para tudo: cada uma
// destas tem uma AÇÃO diferente do outro lado — pedir link novo, conferir o
// número, escolher outro herói, ou nada, porque já está lá dentro.
func recusaDeEntrada(err error) (plataforma.FieldErrorMap, string) {
	switch {
	case errors.Is(err, errCampanhaInexistente):
		return plataforma.FieldErrorMap{"campaignId": {"Não existe campanha com esse número."}}, ""
	case errors.Is(err, errConviteExigido):
		return plataforma.FieldErrorMap{}, "Esta mesa é fechada. Peça um link de convite ao mestre."
	case errors.Is(err, errPersonagemInexistente), errors.Is(err, errPersonagemDeOutro):
		return plataforma.FieldErrorMap{"characterId": {"Escolha um herói seu."}}, ""
	case errors.Is(err, errJaTemPersonagem):
		return plataforma.FieldErrorMap{"characterId": {"Você já tem um herói nesta mesa."}}, ""
	case errors.Is(err, errAlreadyInCampaign):
		return plataforma.FieldErrorMap{"characterId": {"Esse herói já está nesta mesa."}}, ""
	default:
		return plataforma.FieldErrorMap{}, avisoInterno
	}
}

func (s *Server) escreveCarta(w http.ResponseWriter, r *http.Request, status int, v campanhaEntrarView) {
	if v.Erros == nil {
		v.Erros = plataforma.FieldErrorMap{}
	}
	s.WritePage(w, r, status, ui.Page{
		Titulo: "Entrar na mesa",
		Forma:  ui.ShellDense,
		Voltar: "/campanhas",
	}, campanhaEntrar(v))
}

// ── a crônica: a página de uma campanha (ALE-255) ────────────────────────────

// handleCronica desenha a crônica inteira, com a aba escolhida pelo `?tab=`.
//
// UMA resposta, e não três: a tela da SPA dispara consultas separadas para
// campanha, sessões e membros, cada uma com o próprio estado de carregando —
// e a visão geral mostra números que só existem depois que as três voltam.
func (s *Server) handleCronica(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	v, err := s.carregaCronica(r.Context(), currentUser(r), id, r.URL.Query().Get("tab"))
	if errors.Is(err, errCampanhaInexistente) {
		http.Error(w, "Campanha não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		// O `roleIn` recusa quem não é da mesa, e a recusa dele é 403. Aqui ela
		// vira página e não JSON, mas continua sendo a MESMA regra.
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	s.escrevePaginaDaCronica(w, r, http.StatusOK, v)
}

// ── as ações da crônica (ALE-255) ────────────────────────────────────────────

// handleCronicaEditar grava nome e descrição.
//
// A recusa REDESENHA a aba de configuração com o que foi digitado, como a folha
// em branco — e pela mesma razão: a descrição é o campo caro de reescrever.
func (s *Server) handleCronicaEditar(w http.ResponseWriter, r *http.Request) {
	id, eu, ok := donoDaCronica(w, r, s)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	nomeBruto, descricaoBruta := r.PostFormValue("name"), r.PostFormValue("description")

	// A MESMA regra da folha em branco e da rota JSON. Três telas, uma função.
	nome, err := nomeDeCampanha(nomeBruto)
	descricao, errDesc := descricaoDeCampanha(&descricaoBruta)
	if err != nil || errDesc != nil {
		v, erroAoLer := s.carregaCronica(r.Context(), eu, id, "config")
		if erroAoLer != nil {
			http.Error(w, erroAoLer.Error(), http.StatusInternalServerError)
			return
		}
		// O que a pessoa digitou vence o que está no banco: ela está olhando
		// para o próprio texto, e devolver o antigo apagaria a edição dela.
		v.Nome, v.Descricao = nomeBruto, descricaoBruta
		if err != nil {
			v.Erros["name"] = []string{"O nome é obrigatório e cabe em 120 caracteres."}
		}
		if errDesc != nil {
			v.Erros["description"] = []string{"A descrição cabe em 2000 caracteres."}
		}
		s.escrevePaginaDaCronica(w, r, http.StatusUnprocessableEntity, v)
		return
	}

	var set setBuilder
	set.Add("name = ?", nome)
	set.Add("description = ?", nullableArg(descricao))
	if err := set.execTouched(r.Context(), s.db, "UPDATE campaigns", id); err != nil {
		http.Error(w, avisoInterno, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/campanhas/%d?tab=config", id), http.StatusSeeOther)
}

// handleCronicaExcluir apaga a crônica e devolve ao livro.
func (s *Server) handleCronicaExcluir(w http.ResponseWriter, r *http.Request) {
	id, _, ok := donoDaCronica(w, r, s)
	if !ok {
		return
	}
	if err := s.queries.DeleteCampaign(r.Context(), id); err != nil {
		http.Error(w, avisoInterno, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/campanhas", http.StatusSeeOther)
}

// handleCronicaAlternarRegra liga ou desliga UMA regra opcional e devolve só o
// painel dela.
//
// Remendo e não navegação, e essa é a diferença desta ação para as outras duas:
// alternar um interruptor no meio de uma lista de ajustes e recarregar a página
// perderia a posição de quem está lendo. Excluir e salvar LEVAM embora a
// página, então lá o formulário de verdade é o certo.
func (s *Server) handleCronicaAlternarRegra(w http.ResponseWriter, r *http.Request) {
	id, eu, ok := donoDaCronica(w, r, s)
	if !ok {
		return
	}
	regra := chi.URLParam(r, "regra")
	sse := datastar.NewSSE(w, r)

	atuais := s.ignoredRulesOf(r.Context(), id)
	var desejadas []string
	if slices.Contains(atuais, regra) {
		// Estava DESLIGADA: religar é tirá-la do conjunto de exceções.
		for _, x := range atuais {
			if x != regra {
				desejadas = append(desejadas, x)
			}
		}
	} else {
		desejadas = append(append([]string{}, atuais...), regra)
	}
	// A validação é a MESMA da rota JSON: regra que o motor não conhece é
	// recusada mesmo vindo de um caminho de tela.
	normalizadas, msg := normalizeIgnoredRules(desejadas)
	if msg != "" {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erroDaRegra": msg})
		return
	}
	if err := s.gravaRegrasIgnoradas(r.Context(), id, normalizadas); err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erroDaRegra": avisoInterno})
		return
	}

	v, err := s.carregaCronica(r.Context(), eu, id, "config")
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erroDaRegra": avisoInterno})
		return
	}
	fragmento, err := renderFragmento(r.Context(), regrasDaCronica(v))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erroDaRegra": avisoInterno})
		return
	}
	_ = sse.PatchElements(fragmento)
	_ = sse.MarshalAndPatchSignals(map[string]string{"erroDaRegra": ""})
}

// donoDaCronica resolve o id e exige que quem pede seja o DONO.
//
// As três ações desta aba são de mestre, e a trava é aqui e não na tela: a tela
// não mostra a aba para jogador, mas isso é UX — quem postar na mão leva 403.
func donoDaCronica(w http.ResponseWriter, r *http.Request, s *Server) (int64, AuthUser, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return 0, AuthUser{}, false
	}
	eu := currentUser(r)
	c, err := s.queries.GetCampaign(r.Context(), id)
	if err != nil {
		http.Error(w, "Campanha não encontrada", http.StatusNotFound)
		return 0, AuthUser{}, false
	}
	if c.Ownerid != eu.ID {
		http.Error(w, "Só quem mestra pode mudar a crônica.", http.StatusForbidden)
		return 0, AuthUser{}, false
	}
	return id, eu, true
}

func (s *Server) escrevePaginaDaCronica(w http.ResponseWriter, r *http.Request, status int, v cronicaView) {
	s.WritePage(w, r, status, ui.Page{
		Titulo: v.Nome,
		Forma:  ui.ShellDense,
		Voltar: "/campanhas",
		// O rótulo nomeia o destino em vez da seta genérica: daqui se volta
		// para o livro, e "Campanhas" diz isso melhor que "Voltar".
		VoltarRotulo: "Campanhas",
		// Os sinais são só de INTERAÇÃO — o diálogo de excluir e o aviso do
		// interruptor. Nada de estado da aplicação: a aba vem da URL e o resto
		// vem desenhado.
		Sinais: "{erroDaRegra: ''}",
	}, cronica(v))
}
