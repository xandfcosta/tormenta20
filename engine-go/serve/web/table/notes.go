package table

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"t20engine/app"
	"t20engine/domain/markdown"
	"t20engine/serve/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// AS NOTAS DA SESSÃO — o caminho até o banco. O desenho mora no `.templ`; aqui
// ficam as expressões que o Datastar executa e as duas rotas que escrevem. A
// GRAMÁTICA do markdown mora no `markdown/markdown.go`.
//
// AS NOTAS SÃO DO MESTRE. A trava é o `gmCommand`, que devolve 403 a quem postar
// na mão — o botão escondido é cortesia para quem não pode, nunca a segurança.

func (s Scene) RoutesNote(r chi.Router) {
	base := sessionPattern + "/notas"
	// O MESMO endereço serve a CENA e o comando. Não é economia de rota: a
	// janela própria existe para o mestre pôr as notas no segundo monitor, e um
	// endereço que ele possa favoritar é metade do que isso significa.
	r.Get(base, s.notesWindowPage)
	r.Post(base, s.saveNoteSession)
	r.Post(base+"/tarefa/{linha}/{estado}", s.toggleTask)
}

// notesSignals é o que a página manda: o texto em curso.
//
// O NOME DO SINAL É TODO MINÚSCULO porque ele é usado como CHAVE de atributo
// (`data-bind:notes`), e o analisador de HTML minuscula chave: caixa alta ali
// ligaria um sinal NOVO e deixaria o declarado intocado, com o servidor lendo
// para sempre um texto vazio.
type notesSignals struct {
	Notes string `json:"notes"`
}

// readsNotesClient pega o texto que está na tela de quem pediu.
//
// É o RASCUNHO e não a linha do banco, e a escolha é deliberada: quem clicou
// está olhando o que digitou, e um comando que operasse sobre a versão salva
// desfaria as últimas palavras dele sem aviso.
func readsNotesClient(r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var signals notesSignals
	if err := datastar.ReadSignals(r, &signals); err != nil {
		return "", fmt.Errorf("não entendi as notas enviadas: %v", err)
	}
	return signals.Notes, nil
}

// saveNote pede a gravação ao caso de uso e traduz a falha na frase que o mestre
// lê. A razão de o texto NÃO ser aparado está lá, junto da escrita.
func (s Scene) saveNote(r *http.Request, campaignID, sessionID int64, text string) error {
	if err := s.lifecycle.SaveNotes(r.Context(), s.callerOf(r), campaignID, sessionID, text); err != nil {
		return fmt.Errorf("não deu para salvar as notas: %v", err)
	}
	return nil
}

func (s Scene) saveNoteSession(w http.ResponseWriter, r *http.Request) {
	s.notesCommand(w, r, func(text string) (string, error) { return text, nil })
}

// toggleTask marca ou desmarca o quadrinho de UMA linha.
//
// A linha viaja no CAMINHO e não num sinal, como os outros verbos de linha da
// Mesa: o alvo é o que o clique carrega, e um sinal compartilhado por todos os
// quadrinhos seria um lugar a mais para o item errado sobreviver à troca.
func (s Scene) toggleTask(w http.ResponseWriter, r *http.Request) {
	row, err := strconv.Atoi(chi.URLParam(r, "linha"))
	if err != nil {
		http.Error(w, fmt.Sprintf("linha inválida: %q", chi.URLParam(r, "linha")), http.StatusBadRequest)
		return
	}
	marked := chi.URLParam(r, "estado") == "marcar"
	s.notesCommand(w, r, func(text string) (string, error) {
		return markdown.ToggleTask(text, row, marked), nil
	})
}

// notesCommand é o tronco das duas rotas: autoriza, transforma, grava e
// redesenha a PRÉVIA.
//
// Ele redesenha SÓ a prévia, e nunca a caixa de texto — ver o comentário longo
// no `.templ`. O sinal `notas` sai junto porque a alternância de tarefa REESCREVE
// o texto, e é por ele que a caixa se atualiza sem o nó ser trocado: `data-bind`
// reflete o valor sem mexer no cursor de quem digita.
func (s Scene) notesCommand(
	w http.ResponseWriter, r *http.Request,
	transforms func(string) (string, error),
) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	userID := s.deps.CurrentUserID(r)
	_, role, err := s.access.Session(r.Context(), app.Caller{ID: userID}, campaignID, sessionID)
	status := statusOf(err)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	if role != "gm" {
		http.Error(w, "as notas da sessão são do mestre", http.StatusForbidden)
		return
	}
	// LER OS SINAIS ANTES do `NewSSE`: o SDK assume a resposta e fecha o corpo
	// do pedido, então um `ReadSignals` depois dele encontra o corpo fechado.
	// A ordem inversa passa VERDE em teste de handler e falha no servidor de
	// verdade — o `httptest.NewRequest` não reproduz esse ciclo de vida.
	text, readErr := readsNotesClient(r)
	novo, ruleErr := text, error(nil)
	if readErr == nil {
		novo, ruleErr = transforms(text)
	}
	if readErr == nil && ruleErr == nil {
		ruleErr = s.saveNote(r, campaignID, sessionID, novo)
	}
	s.respondNotes(w, r, campaignID, sessionID, novo, primeiroErro(readErr, ruleErr))
}

func primeiroErro(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// respondNotes devolve a prévia e o estado do salvamento.
//
// `notes_saved` é escrito SÓ no acerto, e é ele que faz a faixa dizer "Salvo".
// Escrevê-lo no erro faria a tela afirmar que está no banco o que o banco
// recusou — a mentira mais cara que esta superfície pode contar, porque o mestre
// fecha a aba confiando nela.
//
// OS IDS VIAJAM PARA A PRÉVIA: uma `View` sintética com `CampaignID` e
// `SessionID` ZERO faz cada quadrinho do fragmento remendado apontar para
// `/campanhas/0/sessoes/0/notas/tarefa/N/marcar`. O sintoma é da pior família desta base: o
// PRIMEIRO clique funciona — ele acontece sobre o HTML da carga fria, que tem os
// ids certos — e do segundo em diante a tela fica muda, com o botão no lugar, o
// `aria-checked` desenhado e nenhum erro em canto nenhum. O guarda que o prende
// é `TestThePatchedPreviewCarriesTheTableIds`.
func (s Scene) respondNotes(
	w http.ResponseWriter, r *http.Request,
	campaignID, sessionID int64, text string, refusal error,
) {
	sse := datastar.NewSSE(w, r)
	signals := map[string]any{"notes": text, "notes_error": ""}
	if refusal != nil {
		signals["notes_error"] = refusal.Error()
	} else {
		signals["notes_saved"] = text
		preview := tableNotesPreview(View{
			CampaignID: campaignID, SessionID: sessionID,
			Notes: text, NoteBlocks: markdown.Parse(text),
		})
		if fragment, err := ui.RenderFragment(r.Context(), preview); err == nil {
			_ = sse.PatchElements(fragment)
		}
	}
	_ = sse.MarshalAndPatchSignals(signals)
}

// ── as expressões que o Datastar executa ────────────────────────────────────

// seedNotes põe na página o que o servidor sabe, UMA vez.
//
// O texto é serializado por `json.Marshal` e não concatenado à mão: uma aspa ou
// uma quebra de linha na nota fecharia a expressão e derrubaria a página
// inteira — e nota de mesa é feita de aspas e quebras de linha.
func seedNotes(v View) string {
	text, err := json.Marshal(v.Notes)
	if err != nil {
		text = []byte(`""`)
	}
	return fmt.Sprintf(
		"$notes = %s; $notes_saved = %s; $notes_mode = localStorage.getItem('%s') || 'duplo'; "+
			"$notes_window = localStorage.getItem('%s') === '%d'",
		text, text, notesModeKey, notesWindowKey, v.SessionID,
	)
}

// notesModeKey guarda o arranjo escolhido: é preferência de trabalho, e o mestre
// não deve reescolher a cada sessão.
const notesModeKey = "t20:notas-view"

func pickNotesMode(value string) string {
	return fmt.Sprintf("$notes_mode = '%s'; localStorage.setItem('%s', '%s')", value, notesModeKey, value)
}

// notesAddress é o endereço das notas desta sessão, e ele tem UM lugar: três
// grafias do mesmo caminho é como nasce a quarta que diverge — e o `@post` tem
// guarda de endereço, mas o `window.open` não tem.
func notesAddress(v View) string {
	return v.SessionBase() + "/notas"
}

func saveNotes(v View) string {
	return fmt.Sprintf("@post('%s')", notesAddress(v))
}

func toggleTaskNote(v View, t markdown.Task) string {
	state := "marcar"
	if t.Marked {
		state = "desmarcar"
	}
	return fmt.Sprintf("@post('%s/tarefa/%d/%s')", notesAddress(v), t.Row, state)
}

func marked(checked bool) string {
	if checked {
		return "true"
	}
	return "false"
}
