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
// # Ela é EXCLUSIVA com a coluna, e isso não é gosto
//
// As notas não são região do stream de propósito (ver o cabeçalho do `.templ`):
// são de um leitor só. Duas caixas sobre a mesma coluna `sessions.notes`
// seriam o último-a-salvar-ganha, sem aviso, com as duas faixas dizendo
// "Salvo". Quem fecha a coluna é o `storage` da janela — ver `takesTheNotes`.
//
// A TRAVA é a mesma do comando: as notas são do mestre.
func (s Scene) notesWindowPage(w http.ResponseWriter, r *http.Request) {
	campaignID, sessionID, ok := tableParams(w, r)
	if !ok {
		return
	}
	sess, papel, err := s.access.Session(r.Context(), app.Caller{ID: s.deps.CurrentUserID(r)}, campaignID, sessionID)
	status := statusOf(err)
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}
	if papel != "gm" {
		http.Error(w, "as notas da sessão são do mestre", http.StatusForbidden)
		return
	}
	// A VIEW É MÍNIMA, e é o que separa esta cena da Mesa: ela não carrega
	// tabuleiro, elenco, lente nem fila. O `LoadView` faria tudo isso para
	// desenhar uma caixa de texto — e a janela do segundo monitor recarrega a
	// cada vez que o mestre a chama de volta.
	v := View{
		CampaignID: campaignID, SessionID: sessionID, SessionNum: sess.Sessionnumber,
		Notas: sess.Notes.String, NotasBlocos: markdown.Parse(sess.Notes.String),
	}
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: fmt.Sprintf("Notas · Sessão %d", v.SessionNum),
		Sinais: notesWindowSignals(),
	}, notesWindow(v))
}

// notesWindowSignals é o estado que a JANELA tem, e ele é menor que o da Mesa.
//
// Fora ficam `notes_open`, `notes_width`, `notes_dragging`, `notes_floating` e
// `notes_window`: os cinco descrevem ONDE a coluna vive ao lado do mapa, e aqui
// não há mapa nem coluna. Declarar sinal que ninguém lê é o que o
// `TestEverySignalDeclaredByValueHasAReader` recusa, e com razão — ele viaja em
// toda requisição.
func notesWindowSignals() string {
	return "{notes: '', notes_saved: '', notes_error: '', notes_mode: 'duplo'}"
}

// seedNotesWindow é o mesmo semeador SEM o que descreve a coluna: a janela não
// tem largura para guardar nem mapa para flutuar sobre.
//
// O MODO vem da mesma chave da coluna, de propósito: quem escreve em "Lado a
// lado" ao lado do mapa quer "Lado a lado" na janela também. É a mesma
// preferência de trabalho, e ela não deveria depender de onde o painel está.
func seedNotesWindow(v View) string {
	texto, err := json.Marshal(v.Notas)
	if err != nil {
		texto = []byte(`""`)
	}
	return fmt.Sprintf(
		"$notes = %s; $notes_saved = %s; $notes_mode = localStorage.getItem('%s') || 'duplo'; %s",
		texto, texto, notesModeKey, takesTheNotes(v),
	)
}

// notesWindowKey diz QUAL sessão está com as notas numa janela, e é por ela que
// as duas telas se excluem.
//
// `localStorage` e não `BroadcastChannel`: o evento `storage` chega a toda
// janela da origem sem ninguém segurar um objeto vivo, e é o mesmo mecanismo
// que as outras três preferências das notas já usam. O valor é o ID DA SESSÃO
// porque uma janela aberta na sessão 4 não tem nada a dizer sobre a 7.
const notesWindowKey = "t20:notas-janela"

// notesWindowName é o nome da janela, e ele é o que a torna ÚNICA: um segundo
// `window.open` com o mesmo nome acha a que já existe em vez de abrir outra.
const notesWindowName = "t20-notas"

// takesTheNotes é o anúncio da JANELA: "as notas agora são minhas".
//
// A coluna do lado do mapa não fecha porque quem clicou pediu; ela fecha porque
// a janela TOMOU. A diferença aparece quando o navegador bloqueia o pop-up: ali
// nada é tomado, e a coluna fica onde estava em vez de sumir sobre uma janela
// que não abriu.
func takesTheNotes(v View) string {
	return fmt.Sprintf("localStorage.setItem('%s', '%d')", notesWindowKey, v.SessionID)
}

// givesTheNotesBack roda no `pagehide`, que é o evento que sobrevive ao fechar
// a janela — `beforeunload` pede confirmação em alguns caminhos e `unload` é
// pulado quando a página vai para o cache de ida e volta.
//
// A guarda do valor existe para o caso de DUAS sessões: se a janela da sessão 4
// fechar depois de a da 7 ter tomado, ela não pode apagar a chave da outra.
func givesTheNotesBack(v View) string {
	return fmt.Sprintf("if (localStorage.getItem('%s') === '%d') { localStorage.removeItem('%s') }",
		notesWindowKey, v.SessionID, notesWindowKey)
}

// watchesTheNotesWindow é o outro lado: a aba principal ouve o `storage` e
// obedece.
//
// `evt.newValue` vem NULO quando a chave é removida, e `null === '4'` é falso —
// então fechar a janela devolve a coluna sem um ramo próprio.
func watchesTheNotesWindow(v View) string {
	return fmt.Sprintf(
		"if (evt.key === '%s') { $notes_window = evt.newValue === '%d'; if ($notes_window) { $notes_open = false } }",
		notesWindowKey, v.SessionID)
}

// opensTheNotesWindow: o gesto da faixa.
//
// Ele AFIRMA o bloqueio em vez de supor que abriu. `window.open` devolve nulo
// quando o navegador recusa o pop-up, e é a única pista que existe — sem esta
// frase o clique não faria nada e não diria nada, que é a família de defeito
// que esta base mais persegue.
func opensTheNotesWindow(v View) string {
	return fmt.Sprintf(
		"const janela = window.open('%s', '%s', 'popup,width=620,height=840'); "+
			"if (janela) { janela.focus(); $notes_error = '' } "+
			"else { $notes_error = 'O navegador bloqueou a janela das notas. Libere os pop-ups deste endereço.' }",
		notesAddress(v), notesWindowName)
}

// notesButtonGesture é o botão do trilho, e ele RAMIFICA pelo lugar das notas.
//
// Com a janela no ar, abrir a coluna poria duas caixas sobre a mesma linha do
// banco — então o botão vai buscar a janela. Ele a NAVEGA de novo (mesmo nome,
// mesmo endereço), e isso é aceito de olhos abertos: o custo é a janela
// recarregar, e o que ela poderia perder é o que ainda não passou pela pausa de
// 1,2s do autosave — texto que só existe enquanto alguém digita NAQUELA janela,
// que não é onde o dedo está neste clique.
func notesButtonGesture(v View) string {
	return fmt.Sprintf(
		"if ($notes_window) { window.open('%s', '%s')?.focus() } else { $notes_open = !$notes_open }",
		notesAddress(v), notesWindowName)
}

// marked devolve a STRING e não o booleano: o `data-attr` do Datastar trata
// valor booleano como ATRIBUTO BOOLEANO, e um `aria-checked=""` não anuncia
// estado nenhum. Aqui o valor é escrito direto no HTML, mas a palavra é a mesma
// pela mesma razão.
