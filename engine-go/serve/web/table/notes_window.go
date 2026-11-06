package table

import (
	"encoding/json"
	"fmt"
	"net/http"
	"t20engine/app"
	"t20engine/domain/markdown"
	"t20engine/serve/web/ui"
)

// AS NOTAS NUMA JANELA PRÓPRIA — a cena, e o pacto que a torna única.
//
// Arquivo próprio e não um pedaço do `notes.go`: o que está aqui não é o painel
// ao lado do mapa, é uma CENA com endereço, uma trava de acesso e um protocolo
// entre duas janelas. O vizinho desenha um painel; este resolve onde as notas
// moram.

// notesWindowPage serve a cena das notas.
//
// # Por que uma cena e não o painel mudado de lugar
//
// A forma óbvia — mover o nó vivo para uma janela de Document
// Picture-in-Picture — não funciona: o Datastar não segue o nó adotado por outro
// documento, e a ligação morre EM SILÊNCIO, com a faixa continuando a dizer
// "Salvo" enquanto nada mais é gravado. O que funciona é a janela receber um
// DOCUMENTO, com o runtime dela. Daí a cena.
//
// # Ela é UMA, e o nome da aba é o que a torna única
//
// As notas não são região do stream de propósito (ver o cabeçalho do `.templ`):
// são de um leitor só. Duas caixas sobre a mesma coluna `sessions.notes`
// seriam o último-a-salvar-ganha, sem aviso, com as duas faixas dizendo
// "Salvo". O elo do cabeçalho abre num alvo com NOME (`sidePageTarget`), e um
// segundo clique traz a aba que já está aberta em vez de abrir outra.
//
// A TRAVA é a mesma do comando: as notas são do mestre.
func (s Scene) notesWindowPage(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	sess, role, err := s.access.Session(r.Context(), app.Caller{ID: s.deps.CurrentUserID(r)}, campaignID, sessionID)
	status := statusOf(err)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	if role != "gm" {
		http.Error(w, "as notas da sessão são do mestre", http.StatusForbidden)
		return
	}
	// A VIEW É MÍNIMA, e é o que separa esta cena da Mesa: ela não carrega
	// tabuleiro, elenco, lente nem fila. O `LoadView` faria tudo isso para
	// desenhar uma caixa de texto — e a janela do segundo monitor recarrega a
	// cada vez que o mestre a chama de volta.
	v := View{
		CampaignID: campaignID, SessionID: sessionID, SessionNum: sess.Sessionnumber,
		Notes: sess.Notes.String, NoteBlocks: markdown.Parse(sess.Notes.String),
	}
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Title:   fmt.Sprintf("Notas · Sessão %d", v.SessionNum),
		Signals: notesWindowSignals(),
	}, notesWindow(v))
}

// notesWindowSignals é o estado que a página das notas tem, e ele é menor que o
// da Mesa.
func notesWindowSignals() string {
	return "{notes: '', notes_saved: '', notes_error: '', notes_mode: 'duplo'}"
}

// seedNotesWindow põe na página o que o servidor sabe, UMA vez.
//
// O texto é serializado por `json.Marshal` e não concatenado à mão: uma aspa ou
// uma quebra de linha na nota fecharia a expressão e derrubaria a página
// inteira — e nota de mesa é feita de aspas e quebras de linha.
//
// O MODO vem do `localStorage`: é preferência de trabalho, e o mestre não deve
// reescolher a cada sessão.
func seedNotesWindow(v View) string {
	text, err := json.Marshal(v.Notes)
	if err != nil {
		text = []byte(`""`)
	}
	return fmt.Sprintf(
		"$notes = %s; $notes_saved = %s; $notes_mode = localStorage.getItem('%s') || 'duplo'",
		text, text, notesModeKey,
	)
}
