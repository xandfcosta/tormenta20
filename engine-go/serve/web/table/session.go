package table

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"

	"t20engine/app"
	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
)

// O CICLO DA SESSÃO na cena — iniciar, encerrar, renomear, reiniciar o combate
// e excluir.
//
// # Esta família não passa pelo `gmCommand`, e a diferença é o desenho (ALE-344)
//
// Os outros comandos do mestre autorizam AQUI, no embrulho, e só então chamam a
// mutação. Estes autorizam LÁ DENTRO: quem decide se pode é o `app/session`,
// junto com quem executa. É o que um caso de uso é — a unidade de autorização e
// de escrita —, e repetir a trava aqui seria ter duas opiniões sobre quem
// mestra, que é como uma delas passa a divergir.
//
// O que sobra para a cena é o que é dela: ler o pedido, TRADUZIR a recusa em
// número, e redesenhar.
//
// # Os métodos são REST porque o recurso é a sessão
//
// `PATCH` na própria sessão muda o que dela se pede — o status ou o título —, e
// `DELETE` a apaga. O `reiniciar` NÃO é estado da sessão: a partida continua no
// ar e o que se esvazia é o combate, então ele é um `POST` num sub-recurso.

func (s Scene) RoutesSession(r chi.Router) {
	r.Patch(sessionPattern, s.patchesTheSession)
	r.Delete(sessionPattern, s.deletesTheSession)
	r.Post(sessionPattern+"/combate/reiniciar", s.restartsTheCombat)
}

// commandSession relê a linha, que é onde o ciclo mora.
func commandSession(st Scene, c commandCtx) (sqlcgen.Session, error) {
	return st.deps.Queries().GetSession(c.R.Context(), c.SessionID)
}

// statusPatch é o corpo do `PATCH`, e os campos são PONTEIRO de propósito: só o
// que veio muda, que é o que um remendo significa. Mandar os dois num pedido só
// nunca acontece — cada botão manda o seu —, e se acontecesse os dois valeriam.
type statusPatch struct {
	Status *string `json:"status"`
	Titulo *string `json:"session_title"`
}

// patchesTheSession é o `PATCH` da sessão: o status, o título, ou os dois.
func (s Scene) patchesTheSession(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	quem := s.callerOf(r)

	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var remendo statusPatch
	if err := datastar.ReadSignals(r, &remendo); err != nil {
		http.Error(w, fmt.Sprintf("não entendi o remendo enviado: %v", err), http.StatusBadRequest)
		return
	}
	if remendo.Status == nil && remendo.Titulo == nil {
		http.Error(w, "o remendo não pede nada: mande `status` ou `session_title`", http.StatusBadRequest)
		return
	}

	var estado *live.SessionRuntimeState
	var recusa error
	if remendo.Titulo != nil {
		// APARADO aqui e não no caso de uso: espaço em branco é coisa de campo de
		// texto, e o vazio que sobra é legítimo — a sessão tem NÚMERO, que é a
		// identidade dela.
		recusa = s.lifecycle.Rename(r.Context(), quem, campaignID, sessionID, strings.TrimSpace(*remendo.Titulo))
	}
	if recusa == nil && remendo.Status != nil {
		estado, recusa = s.lifecycle.SetStatus(r.Context(), quem, campaignID, sessionID, *remendo.Status)
	}
	s.answersTheLifecycle(w, r, quem, campaignID, sessionID, estado, recusa)
}

// restartsTheCombat esvazia a fila e os turnos SEM tirar a partida do ar.
//
// Os dois verbos moram na mesma tela, um perto do outro, e é por isso que a
// frase de cada um diz o que ACONTECE em vez de repetir o nome do botão:
// "encerrar" tira a sessão do ar, "reiniciar" só apaga a ordem e os turnos.
func (s Scene) restartsTheCombat(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	quem := s.callerOf(r)
	estado, recusa := s.lifecycle.RestartCombat(r.Context(), quem, campaignID, sessionID)
	s.answersTheLifecycle(w, r, quem, campaignID, sessionID, estado, recusa)
}

// deletesTheSession apaga a sessão e MANDA O MESTRE PARA A CRÔNICA.
//
// O destino é a crônica da campanha, que é de onde se entra numa sessão —
// voltar para a mesa apagada seria mandar o mestre para uma porta que não existe
// mais.
//
// # Ele NAVEGA, e a navegação vem pelo fio (ALE-344)
//
// Aqui havia um `<form method="post">` com um `http.Redirect`, porque `DELETE`
// não cabe num formulário de HTML — o elemento só faz `GET` e `POST`. Com o
// método certo no lugar, quem navega é o `sse.Redirect`, que o SDK do Datastar
// tem para isto. **O preço é real e está escrito aqui**: o formulário funcionava
// sem JavaScript nenhum e este caminho não funciona.
func (s Scene) deletesTheSession(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	quem := s.callerOf(r)
	if err := s.lifecycle.Delete(r.Context(), quem, campaignID, sessionID); err != nil {
		http.Error(w, err.Error(), statusOf(err))
		return
	}
	sse := datastar.NewSSE(w, r)
	_ = sse.Redirect("/campanhas/" + strconv.FormatInt(campaignID, 10))
}

// callerOf é quem está pedindo, na forma que o caso de uso recebe.
//
// `IsAdmin` é FALSO aqui, e isso é a verdade de hoje e não um esquecimento: a
// cena recebe o id de quem pede e nada mais, e nenhum gesto do ciclo tem o
// desvio de administrador. Quando tiver, o que atravessa é o valor — não um
// segundo jeito de perguntar.
func (s Scene) callerOf(r *http.Request) app.Caller {
	return app.Caller{ID: s.deps.CurrentUserID(r)}
}

// answersTheLifecycle responde o que os três gestos de remendo respondem: a
// cena redesenhada, com a recusa escrita no rodapé do mestre.
func (s Scene) answersTheLifecycle(
	w http.ResponseWriter, r *http.Request,
	quem app.Caller, campaignID, sessionID int64,
	estado *live.SessionRuntimeState, recusa error,
) {
	// NÃO ENCONTRADO e NÃO É SEU saem como status, e não como frase no rodapé:
	// quem não alcança a sessão não tem rodapé para ler. A recusa da REGRA é a
	// que vira frase — ela é sobre o gesto, e quem a recebeu está olhando a tela.
	if recusa != nil && !errors.Is(recusa, app.ErrRefused) {
		http.Error(w, recusa.Error(), statusOf(recusa))
		return
	}
	if estado != nil {
		s.deps.PublishSessionState(sessionID, estado)
	}
	s.respondGm(w, r, quem.ID, campaignID, sessionID, recusa, map[string]any{})
}

// statusOf traduz a recusa TIPADA do caso de uso no número que o navegador
// entende.
//
// A tradução mora aqui porque ela é do TRANSPORTE: o `app/` não conhece HTTP, e
// é isso que o deixa ser chamado de outro lugar. Ver o `boundary` do grupo.
func statusOf(err error) int {
	switch {
	case errors.Is(err, app.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, app.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, app.ErrRefused):
		return http.StatusUnprocessableEntity
	}
	return http.StatusInternalServerError
}
