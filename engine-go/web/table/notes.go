package table

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"t20engine/markdown"
	"t20engine/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// AS NOTAS DA SESSÃO (ALE-269, superfície 5) — o caminho até o banco.
//
// O desenho mora no `.templ`; aqui ficam as expressões que o Datastar executa e
// as duas rotas que escrevem. A GRAMÁTICA do markdown mora no
// `markdown/markdown.go`, que é um port com paridade medida contra o JS.
//
// AS NOTAS SÃO DO MESTRE. A trava é o `gmCommand`, que devolve 403 a quem
// postar na mão — o botão escondido é cortesia para quem não pode, nunca a
// segurança.

func (s Scene) RoutesNote(r chi.Router) {
	base := "/mesa/{campaignId}/{sessionId}/notas"
	r.Post(base, s.saveNoteSession)
	r.Post(base+"/tarefa/{linha}/{estado}", s.toggleTask)
}

// notesSignals é o que a página manda: o texto em curso.
//
// O NOME DO SINAL É TODO MINÚSCULO porque ele é usado como CHAVE de atributo
// (`data-bind:notas`), e o analisador de HTML minuscula chave — um
// `data-bind:notasDaSessao` chegaria como `notasdasessao` e ligaria um sinal
// NOVO, com o servidor lendo o antigo para sempre vazio. Já custou uma sessão
// inteira no descanso de dia.
type notesSignals struct {
	Notas string `json:"notas"`
}

// readsNotesClient pega o texto que está na tela de quem pediu.
//
// É o RASCUNHO e não a linha do banco, e a escolha é deliberada: quem clicou
// está olhando o que digitou, e um comando que operasse sobre a versão salva
// desfaria as últimas palavras dele sem aviso.
func readsNotesClient(r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	var sinais notesSignals
	if err := datastar.ReadSignals(r, &sinais); err != nil {
		return "", fmt.Errorf("não entendi as notas enviadas: %v", err)
	}
	return sinais.Notas, nil
}

// saveNote escreve a coluna `notes` pelo MESMO `setBuilder` do handler JSON.
//
// Uma segunda forma de gravar a mesma coluna divergiria no dia em que o
// `execTouched` mudar — é ele quem carimba o `updatedAt`. Mesmo argumento que o
// título da sessão registra.
//
// NÃO PASSA POR `trimOrNull`, e essa é a diferença que importa aqui: aparar o
// texto a cada 1,2s comeria a linha em branco que o mestre acabou de abrir para
// escrever o próximo parágrafo. O handler JSON apara porque salva UMA vez, ao
// fechar; este salva no meio da digitação.
func (s Scene) saveNote(r *http.Request, sessionID int64, texto string) error {
	if err := s.deps.SaveNotes(r.Context(), sessionID, texto); err != nil {
		return fmt.Errorf("não deu para salvar as notas: %v", err)
	}
	return nil
}

func (s Scene) saveNoteSession(w http.ResponseWriter, r *http.Request) {
	s.notesCommand(w, r, func(texto string) (string, error) { return texto, nil })
}

// toggleTask marca ou desmarca o quadrinho de UMA linha.
//
// A linha viaja no CAMINHO e não num sinal, como os outros verbos de linha da
// Mesa: o alvo é o que o clique carrega, e um sinal compartilhado por todos os
// quadrinhos seria um lugar a mais para o item errado sobreviver à troca.
func (s Scene) toggleTask(w http.ResponseWriter, r *http.Request) {
	linha, err := strconv.Atoi(chi.URLParam(r, "linha"))
	if err != nil {
		http.Error(w, fmt.Sprintf("linha inválida: %q", chi.URLParam(r, "linha")), http.StatusBadRequest)
		return
	}
	marcada := chi.URLParam(r, "estado") == "marcar"
	s.notesCommand(w, r, func(texto string) (string, error) {
		return markdown.ToggleTask(texto, linha, marcada), nil
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
	transforma func(string) (string, error),
) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	userID := s.deps.CurrentUserID(r)
	_, papel, status, err := s.deps.SessionForCaller(r.Context(), userID, campaignID, sessionID)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	if papel != "gm" {
		http.Error(w, "as notas da sessão são do mestre", http.StatusForbidden)
		return
	}
	// LER OS SINAIS ANTES do `NewSSE`: o SDK assume a resposta e fecha o corpo
	// do pedido, então um `ReadSignals` depois dele encontra o corpo fechado.
	// A ordem inversa passa VERDE em teste de handler e falha no servidor de
	// verdade — o `httptest.NewRequest` não reproduz esse ciclo de vida.
	texto, erroDeLeitura := readsNotesClient(r)
	novo, erroDaRegra := texto, error(nil)
	if erroDeLeitura == nil {
		novo, erroDaRegra = transforma(texto)
	}
	if erroDeLeitura == nil && erroDaRegra == nil {
		erroDaRegra = s.saveNote(r, sessionID, novo)
	}
	s.respondNotes(w, r, campaignID, sessionID, novo, primeiroErro(erroDeLeitura, erroDaRegra))
}

func primeiroErro(erros ...error) error {
	for _, e := range erros {
		if e != nil {
			return e
		}
	}
	return nil
}

// respondNotes devolve a prévia e o estado do salvamento.
//
// `notassalvas` é escrito SÓ no acerto, e é ele que faz a faixa dizer "Salvo".
// Escrevê-lo no erro também faria a tela afirmar que está no banco o que o
// banco recusou — a mentira mais cara que esta superfície pode contar, porque o
// mestre fecha a aba confiando nela.
// OS IDS VIAJAM PARA A PRÉVIA, e esta linha existe por um defeito MEDIDO no
// navegador: a `View` sintética nascia com `CampaignID` e `SessionID` ZERO,
// e cada quadrinho do fragmento remendado saía apontando para
// `/mesa/0/0/notas/tarefa/N/marcar`.
//
// O sintoma é da pior família desta base: o PRIMEIRO clique funcionava — ele
// acontece sobre o HTML da carga fria, que tem os ids certos — e a partir do
// segundo a tela ficava muda, com o botão no lugar, o `aria-checked` desenhado e
// nenhum erro em canto nenhum. O guarda que o prende é
// `TestThePatchedPreviewCarriesTheTableIds`.
func (s Scene) respondNotes(
	w http.ResponseWriter, r *http.Request,
	campaignID, sessionID int64, texto string, recusa error,
) {
	sse := datastar.NewSSE(w, r)
	sinais := map[string]any{"notas": texto, "erroDasNotas": ""}
	if recusa != nil {
		sinais["erroDasNotas"] = recusa.Error()
	} else {
		sinais["notassalvas"] = texto
		previa := tableNotesPreview(View{
			CampaignID: campaignID, SessionID: sessionID,
			Notas: texto, NotasBlocos: markdown.Parse(texto),
		})
		if fragmento, err := ui.RenderFragment(r.Context(), previa); err == nil {
			_ = sse.PatchElements(fragmento)
		}
	}
	_ = sse.MarshalAndPatchSignals(sinais)
}

// ── as expressões que o Datastar executa ────────────────────────────────────

// seedNotes põe na página o que o servidor sabe, UMA vez.
//
// O modo vem do `localStorage` com a MESMA chave da SPA, para a escolha do
// mestre atravessar as duas telas enquanto as duas existirem.
//
// O texto é serializado por `json.Marshal` e não concatenado à mão: uma aspa ou
// uma quebra de linha na nota fecharia a expressão e derrubaria a página
// inteira — e nota de mesa é feita de aspas e quebras de linha.
func seedNotes(v View) string {
	texto, err := json.Marshal(v.Notas)
	if err != nil {
		texto = []byte(`""`)
	}
	return fmt.Sprintf(
		"$notas = %s; $notassalvas = %s; $notasmodo = localStorage.getItem('%s') || 'duplo'",
		texto, texto, notesModeKey,
	)
}

// notesModeKey é a MESMA do `notes-view.ts`. Duas chaves fariam o mestre
// reescolher o arranjo ao trocar de tela.
const notesModeKey = "t20:notas-view"

func escolheOModo(valor string) string {
	return fmt.Sprintf("$notasmodo = '%s'; localStorage.setItem('%s', '%s')", valor, notesModeKey, valor)
}

func saveNotes(v View) string {
	return fmt.Sprintf("@post('/mesa/%d/%d/notas')", v.CampaignID, v.SessionID)
}

func toggleTaskNote(v View, t markdown.Task) string {
	estado := "marcar"
	if t.Marcada {
		estado = "desmarcar"
	}
	return fmt.Sprintf("@post('/mesa/%d/%d/notas/tarefa/%d/%s')",
		v.CampaignID, v.SessionID, t.Linha, estado)
}

// marked devolve a STRING e não o booleano, e isso é conserto de defeito
// MEDIDO: o `data-attr` do Datastar trata valor booleano como ATRIBUTO
// BOOLEANO, e um `aria-checked=""` não anuncia estado nenhum. Aqui o valor é
// escrito direto no HTML, mas a palavra é a mesma pela mesma razão.
func marked(marcada bool) string {
	if marcada {
		return "true"
	}
	return "false"
}
