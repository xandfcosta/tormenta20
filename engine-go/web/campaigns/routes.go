package campaigns

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"t20engine/campaign"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"

	"t20engine/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// A rota da cena de CAMPANHAS (ALE-234).
//
// UMA rota serve os dois casos, e é isso que a mantém pequena: a carga fria
// devolve a página inteira, e a busca — que chega pelo mesmo `GET` com os
// sinais do Datastar — devolve só o remendo da cena. Quem distingue os dois é o
// cabeçalho `datastar-request`, que o cliente põe.

// Routes registra os endereços das quatro telas desta cena.
func Routes(r chi.Router, s Scene) {
	r.Get("/campanhas", s.handleList)
	r.Get("/campanhas/nova", s.handleNew)
	r.Post("/campanhas/nova", s.handleNewPost)
	r.Get("/campanhas/entrar", s.handleJoin)
	r.Post("/campanhas/entrar", s.handleJoinPost)
	r.Get("/campanhas/{id}", s.handleOne)
	r.Post("/campanhas/{id}/editar", s.handleEdit)
	r.Post("/campanhas/{id}/excluir", s.handleDelete)
	r.Post("/campanhas/{id}/regras/{regra}", s.handleToggleRule)
}

func (s Scene) handleList(w http.ResponseWriter, r *http.Request) {
	busca, papel := filterFromRequest(r)
	view, err := s.LoadList(r.Context(), s.deps.CurrentUserID(r), s.deps.RequesterIsAdmin(r), busca, papel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Pedido do Datastar: remendo. A cena inteira, e não só a lista, porque a
	// barra também muda — o chip de papel aceso e o texto da busca.
	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := ui.RenderFragment(r.Context(), SceneBody(view))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: "Campanhas · Tormenta 20",
		// `cascaNua`: esta cena desenha o próprio cabeçalho, porque ele carrega a
		// busca e os filtros. A casca densa poria um segundo `<h1>` acima.
		Forma: ui.ShellBare,
	}, SceneBody(view))
}

// filterFromRequest lê a busca e o papel, venham de onde vierem.
//
// Da URL na carga fria (`?busca=...`), e dos SINAIS quando o Datastar chama —
// ele os manda no `?datastar=` como JSON. Ler os dois no mesmo lugar é o que
// deixa a tela filtrada ser um endereço que se guarda: recarregar `?busca=anao`
// devolve exatamente o que estava.
func filterFromRequest(r *http.Request) (busca, papel string) {
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

// listURL monta o endereço que a cena representa, para o histórico do
// navegador acompanhar a busca.
func listURL(busca, papel string) string {
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

// handleNew desenha o formulário vazio.
func (s Scene) handleNew(w http.ResponseWriter, r *http.Request) {
	s.writeNewPage(w, r, http.StatusOK, newView{})
}

// handleNewPost cria e vai para a crônica recém-aberta.
//
// A recusa REDESENHA a folha com o que foi digitado e o erro no campo — não
// redireciona. Redirecionar perderia o texto, e a descrição é o campo caro de
// reescrever. Status 422 e não 200 porque a resposta É uma recusa, e o
// navegador não trata os dois igual no histórico.
func (s Scene) handleNewPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.writeNewPage(w, r, http.StatusBadRequest, newView{Aviso: ui.NoticeInternal})
		return
	}
	v := newView{
		Nome:      r.PostFormValue("name"),
		Descricao: r.PostFormValue("description"),
		Erros:     plataforma.FieldErrorMap{},
	}
	// A MESMA regra da rota JSON, e não uma cópia dela — ver `campaign/rules.go`.
	nome, descricaoTexto, erros := campaign.ValidateText(v.Nome, &v.Descricao)
	for campo, frases := range erros {
		v.Erros[campo] = frases
	}
	if len(v.Erros) > 0 {
		s.writeNewPage(w, r, http.StatusUnprocessableEntity, v)
		return
	}

	agora := plataforma.NowISO()
	c, err := s.deps.Queries().CreateCampaign(r.Context(), sqlcgen.CreateCampaignParams{
		Ownerid: s.deps.CurrentUserID(r), Name: nome, Description: trimOrNull(descricaoTexto),
		Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		v.Aviso = ui.NoticeInternal
		s.writeNewPage(w, r, http.StatusInternalServerError, v)
		return
	}
	// 303 e não 302: depois de um POST, o `See Other` é o que garante que o
	// navegador siga com GET. Sem ele, recarregar a crônica reenviaria o
	// formulário e abriria uma segunda campanha igual.
	http.Redirect(w, r, "/campanhas/"+strconv.FormatInt(c.ID, 10), http.StatusSeeOther)
}

func (s Scene) writeNewPage(w http.ResponseWriter, r *http.Request, status int, v newView) {
	s.deps.WritePage(w, r, status, ui.Page{
		Titulo: "Abrir nova campanha",
		// `cascaDensa`: a tela da SPA usa o cabeçalho compacto com o "‹ Voltar",
		// e sem ele a folha nasce sem saída visível — o Esc existe, mas atalho
		// não é a única porta.
		Forma:  ui.ShellDense,
		Voltar: "/campanhas",
	}, newBody(v))
}

// ── a carta de convite: entrar na mesa (ALE-249) ─────────────────────────────

// handleJoin desenha a carta, com o convite JÁ RESOLVIDO.
func (s Scene) handleJoin(w http.ResponseWriter, r *http.Request) {
	v, err := s.LoadJoin(r.Context(), s.deps.CurrentUserID(r), r.URL.Query().Get("token"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.writeJoinPage(w, r, http.StatusOK, v)
}

// handleJoinPost senta o herói à mesa.
//
// As sete travas são do `entrarNaMesa` e não daqui — a mesma função que a rota
// JSON usa. O que este manipulador faz é TRADUZIR cada recusa para uma frase em
// português no campo certo, que é trabalho de tela e não de regra.
func (s Scene) handleJoinPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.writeJoinPage(w, r, http.StatusBadRequest, joinView{Aviso: ui.NoticeInternal})
		return
	}
	eu := s.deps.CurrentUserID(r)
	token := r.PostFormValue("token")
	v, err := s.LoadJoin(r.Context(), eu, token)
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
			s.writeJoinPage(w, r, http.StatusUnprocessableEntity, v)
			return
		}
		campanhaID = n
	}

	heroiID, erroHeroi := strconv.ParseInt(r.PostFormValue("characterId"), 10, 64)
	if erroHeroi != nil {
		v.Erros["characterId"] = []string{"Escolha o herói que entra na mesa."}
		s.writeJoinPage(w, r, http.StatusUnprocessableEntity, v)
		return
	}
	v.EscolhidoID = heroiID

	if recusa := s.deps.Join(r.Context(), campanhaID, heroiID, s.deps.CurrentUserID(r), token); recusa != JoinOK {
		v.Erros, v.Aviso = joinRefusalPhrase(recusa)
		s.writeJoinPage(w, r, http.StatusUnprocessableEntity, v)
		return
	}
	// 303, como a folha em branco: depois de um POST, recarregar a crônica não
	// pode reenviar o formulário.
	http.Redirect(w, r, "/campanhas/"+strconv.FormatInt(campanhaID, 10), http.StatusSeeOther)
}

// joinRefusalPhrase traduz cada MOTIVO de recusa na frase que a pessoa lê.
//
// Uma frase por recusa, e não um "não foi possível entrar" para tudo: cada uma
// destas tem uma AÇÃO diferente do outro lado — pedir link novo, conferir o
// número, escolher outro herói, ou nada, porque já está lá dentro.
//
// Ela recebia o ERRO do hospedeiro e passou a receber o `JoinRefusal` desta cena
// (ALE-278). O que a tela diz não mudou; o que mudou é que os sentinelas de erro
// deixaram de atravessar a fronteira. Quem classifica é o hospedeiro, quem
// escolhe a frase é a cena — a decisão que a porta de entrar deixou escrita.
func joinRefusalPhrase(recusa JoinRefusal) (plataforma.FieldErrorMap, string) {
	switch recusa {
	case JoinNoSuchCampaign:
		return plataforma.FieldErrorMap{"campaignId": {"Não existe campanha com esse número."}}, ""
	case JoinNeedsInvite:
		return plataforma.FieldErrorMap{}, "Esta mesa é fechada. Peça um link de convite ao mestre."
	case JoinNotYourHero:
		return plataforma.FieldErrorMap{"characterId": {"Escolha um herói seu."}}, ""
	case JoinAlreadyHasHero:
		return plataforma.FieldErrorMap{"characterId": {"Você já tem um herói nesta mesa."}}, ""
	case JoinHeroAlreadyThere:
		return plataforma.FieldErrorMap{"characterId": {"Esse herói já está nesta mesa."}}, ""
	default:
		return plataforma.FieldErrorMap{}, ui.NoticeInternal
	}
}

func (s Scene) writeJoinPage(w http.ResponseWriter, r *http.Request, status int, v joinView) {
	if v.Erros == nil {
		v.Erros = plataforma.FieldErrorMap{}
	}
	s.deps.WritePage(w, r, status, ui.Page{
		Titulo: "Entrar na mesa",
		Forma:  ui.ShellDense,
		Voltar: "/campanhas",
	}, JoinBody(v))
}

// ── a crônica: a página de uma campanha (ALE-255) ────────────────────────────

// handleOne desenha a crônica inteira, com a aba escolhida pelo `?tab=`.
//
// UMA resposta, e não três: a tela da SPA dispara consultas separadas para
// campanha, sessões e membros, cada uma com o próprio estado de carregando —
// e a visão geral mostra números que só existem depois que as três voltam.
func (s Scene) handleOne(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return
	}
	v, err := s.LoadOne(r.Context(), s.deps.CurrentUserID(r), s.deps.RequesterIsAdmin(r), id, r.URL.Query().Get("tab"))
	if errors.Is(err, errNoSuchCampaign) {
		http.Error(w, "Campanha não encontrada", http.StatusNotFound)
		return
	}
	if err != nil {
		// O `roleIn` recusa quem não é da mesa, e a recusa dele é 403. Aqui ela
		// vira página e não JSON, mas continua sendo a MESMA regra.
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	s.writeOnePage(w, r, http.StatusOK, v)
}

// ── as ações da crônica (ALE-255) ────────────────────────────────────────────

// handleEdit grava nome e descrição.
//
// A recusa REDESENHA a aba de configuração com o que foi digitado, como a folha
// em branco — e pela mesma razão: a descrição é o campo caro de reescrever.
func (s Scene) handleEdit(w http.ResponseWriter, r *http.Request) {
	id, eu, ok := s.ownerOrRefuse(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	nomeBruto, descricaoBruta := r.PostFormValue("name"), r.PostFormValue("description")

	// A MESMA regra da folha em branco e da rota JSON. Três telas, uma função —
	// e desde a ALE-278 uma FRASE também: as mensagens moram no `campaign`,
	// porque quem lê é o mestre e não o programa.
	nome, descricaoTexto, erros := campaign.ValidateText(nomeBruto, &descricaoBruta)
	if len(erros) > 0 {
		v, erroAoLer := s.LoadOne(r.Context(), eu, s.deps.RequesterIsAdmin(r), id, "config")
		if erroAoLer != nil {
			http.Error(w, erroAoLer.Error(), http.StatusInternalServerError)
			return
		}
		// O que a pessoa digitou vence o que está no banco: ela está olhando
		// para o próprio texto, e devolver o antigo apagaria a edição dela.
		v.Nome, v.Descricao = nomeBruto, descricaoBruta
		for campo, frases := range erros {
			v.Erros[campo] = frases
		}
		s.writeOnePage(w, r, http.StatusUnprocessableEntity, v)
		return
	}

	// A GRAVAÇÃO é uma pergunta e não um SQL montado aqui (ALE-278).
	//
	// Esta cena compunha `setBuilder` + `execTouched` + `"UPDATE campaigns"` à
	// mão, e cena que compõe SQL é cena com o banco dentro. O hospedeiro sabe
	// que a coluna se chama `description`, que vazio é NULL e que a linha tem um
	// `updatedAt` a tocar; a cena sabe que o mestre renomeou a mesa.
	if err := s.deps.SaveText(r.Context(), id, nome, descricaoTexto); err != nil {
		http.Error(w, ui.NoticeInternal, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/campanhas/%d?tab=config", id), http.StatusSeeOther)
}

// handleDelete apaga a crônica e devolve ao livro.
func (s Scene) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, _, ok := s.ownerOrRefuse(w, r)
	if !ok {
		return
	}
	if err := s.deps.Queries().DeleteCampaign(r.Context(), id); err != nil {
		http.Error(w, ui.NoticeInternal, http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/campanhas", http.StatusSeeOther)
}

// handleToggleRule liga ou desliga UMA regra opcional e devolve só o
// painel dela.
//
// Remendo e não navegação, e essa é a diferença desta ação para as outras duas:
// alternar um interruptor no meio de uma lista de ajustes e recarregar a página
// perderia a posição de quem está lendo. Excluir e salvar LEVAM embora a
// página, então lá o formulário de verdade é o certo.
func (s Scene) handleToggleRule(w http.ResponseWriter, r *http.Request) {
	id, eu, ok := s.ownerOrRefuse(w, r)
	if !ok {
		return
	}
	regra := chi.URLParam(r, "regra")
	sse := datastar.NewSSE(w, r)

	atuais := s.deps.IgnoredRules(r.Context(), id)
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
	normalizadas, msg := campaign.NormalizeIgnoredRules(desejadas)
	if msg != "" {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erroDaRegra": msg})
		return
	}
	if err := s.deps.SaveIgnoredRules(r.Context(), id, normalizadas); err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erroDaRegra": ui.NoticeInternal})
		return
	}

	v, err := s.LoadOne(r.Context(), eu, s.deps.RequesterIsAdmin(r), id, "config")
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erroDaRegra": ui.NoticeInternal})
		return
	}
	fragmento, err := ui.RenderFragment(r.Context(), rulesPanel(v))
	if err != nil {
		_ = sse.MarshalAndPatchSignals(map[string]string{"erroDaRegra": ui.NoticeInternal})
		return
	}
	_ = sse.PatchElements(fragmento)
	_ = sse.MarshalAndPatchSignals(map[string]string{"erroDaRegra": ""})
}

// ownerOrRefuse resolve o id e exige que quem pede seja o DONO.
//
// As três ações desta aba são de mestre, e a trava é aqui e não na tela: a tela
// não mostra a aba para jogador, mas isso é UX — quem postar na mão leva 403.
func (s Scene) ownerOrRefuse(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "id inválido", http.StatusBadRequest)
		return 0, 0, false
	}
	eu := s.deps.CurrentUserID(r)
	c, err := s.deps.Queries().GetCampaign(r.Context(), id)
	if err != nil {
		http.Error(w, "Campanha não encontrada", http.StatusNotFound)
		return 0, 0, false
	}
	if c.Ownerid != eu {
		http.Error(w, "Só quem mestra pode mudar a crônica.", http.StatusForbidden)
		return 0, 0, false
	}
	return id, eu, true
}

func (s Scene) writeOnePage(w http.ResponseWriter, r *http.Request, status int, v oneView) {
	s.deps.WritePage(w, r, status, ui.Page{
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
	}, oneBody(v))
}
