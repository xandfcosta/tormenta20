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
// `data-bind:notesSignals` chegaria como `notasdasessao` e ligaria um sinal
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
		"$notas = %s; $notassalvas = %s; $notasmodo = localStorage.getItem('%s') || 'duplo'; "+
			"$notaslargura = Number(localStorage.getItem('%s')) || 0",
		texto, texto, notesModeKey, notesWidthKey,
	)
}

// notesModeKey é a MESMA do `notes-view.ts`. Duas chaves fariam o mestre
// reescolher o arranjo ao trocar de tela.
const notesModeKey = "t20:notas-view"

func escolheOModo(valor string) string {
	return fmt.Sprintf("$notasmodo = '%s'; localStorage.setItem('%s', '%s')", valor, notesModeKey, valor)
}

// A LARGURA DA COLUNA (ALE-218), e ela GRUDA como os modos grudam.
//
// É preferência de trabalho e não estado da sessão — o mestre escolhe uma vez o
// quanto de mapa quer ver ao lado das notas, e não deve reescolher a cada
// sessão. Chave própria porque é outra escolha que a do arranjo.
const notesWidthKey = "t20:notas-largura"

// O PISO é o do `clamp` que a coluna tinha fixo: abaixo de 22rem o "lado a lado"
// não cabe e a coluna vira uma tira inútil.
const notesMinWidth = 352 // 22rem

// O TETO é RELATIVO ao palco, e o de 44rem do `clamp` não servia.
//
// Lá ele limitava uma PORCENTAGEM, então nunca era alcançado numa janela
// pequena. Copiado para uma divisa explícita ele vira uma parede: medido a
// 1920, 40% já dá exatamente 704px, e a divisa nascia sem PARA ONDE CRESCER —
// morta numa das duas direções, na tela em que ela mais serve.
//
// 70% do palco deixa o mapa com quase um terço em qualquer janela, que é o que
// mantém as notas ao lado do tabuleiro em vez de no lugar dele.
func oTetoDaLargura() string {
	return "(document.getElementById('mesa-notas').parentElement.getBoundingClientRect().width * 0.7)"
}

// oPassoDaLargura é a seta do teclado, e ela existe porque **gesto nunca é o
// único caminho**: uma divisa que só responde a arrasto é uma preferência que
// quem não usa ponteiro não tem.
//
// 32px por seta, e o `Home` devolve ao padrão — o número redondo é escolha, e o
// que importa é ele ser grande o bastante para atravessar a faixa em poucos
// toques e pequeno o bastante para ajustar.
func oPassoDaLargura() string {
	return fmt.Sprintf(
		"if (evt.key === 'ArrowLeft' || evt.key === 'ArrowRight') { evt.preventDefault(); "+
			"$notaslargura = Math.min(%s, Math.max(%d, %s + (evt.key === 'ArrowLeft' ? 32 : -32))); %s } "+
			"if (evt.key === 'Home') { evt.preventDefault(); $notaslargura = 0; localStorage.removeItem('%s') }",
		oTetoDaLargura(), notesMinWidth, aLarguraDeAgora(), guardaALargura(), notesWidthKey,
	)
}

// aLarguraDeAgora é o valor de PARTIDA de um ajuste, e ele é medido na tela em
// vez de cair num padrão.
//
// Enquanto o mestre não escolhe, `$notaslargura` é zero e quem manda é o
// `clamp` da folha — que depende da janela. Um padrão escrito aqui faria a
// PRIMEIRA seta SALTAR: medido, a coluna ia de 704px para 384 num toque, porque
// o piso de 22rem não é o que está na tela. A divisa tem de continuar de onde a
// coluna está.
func aLarguraDeAgora() string {
	return "($notaslargura || document.getElementById('mesa-notas').getBoundingClientRect().width)"
}

// oArrastoDaLargura é o gesto de ponteiro. A conta é sobre a borda DIREITA da
// coluna, que não se move: arrastar para a esquerda cresce as notas.
func oArrastoDaLargura() string {
	return "evt.preventDefault(); $notasarrastando = true; el.setPointerCapture(evt.pointerId)"
}

func oMoverDaLargura() string {
	return fmt.Sprintf(
		"if ($notasarrastando) { const c = document.getElementById('mesa-notas').getBoundingClientRect(); "+
			"$notaslargura = Math.min(%s, Math.max(%d, c.right - evt.clientX)) }",
		oTetoDaLargura(), notesMinWidth,
	)
}

func oSoltarDaLargura() string {
	return "if ($notasarrastando) { $notasarrastando = false; " + guardaALargura() + " }"
}

func guardaALargura() string {
	return fmt.Sprintf("localStorage.setItem('%s', $notaslargura)", notesWidthKey)
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
