package api

import (
	"strings"
	"testing"
)

func TestThePlayerDoesNotWriteInTheGmNotes(t *testing.T) {
	f := newSceneFixture(t)

	rec := f.pede(t, f.jogador, "POST", f.tableUrl()+"/notas", `{"notes":"eu escrevi isto"}`)

	if rec.Code != 403 {
		t.Errorf("o jogador escreveu nas notas do mestre: %d", rec.Code)
	}
	// O CONTROLE do 403: se a nota tivesse sido gravada, o status sozinho não
	// contaria — já houve rota que recusava DEPOIS de escrever.
	if nota := f.dbNote(t); nota != "" {
		t.Errorf("a recusa veio depois da escrita: o banco tem %q", nota)
	}
}

// O caminho feliz, ponta a ponta pelo roteador de verdade.
func TestTheNoteAutosaveReachesTheDatabase(t *testing.T) {
	f := newSceneFixture(t)

	f.posta(t, f.mestre, f.tableUrl()+"/notas", `{"notes":"# Cena 1\nO ogro fugiu"}`)

	if got := f.dbNote(t); got != "# Cena 1\nO ogro fugiu" {
		t.Errorf("a nota no banco é %q", got)
	}
}

// A nota NÃO é aparada no meio da digitação.
//
// Aparar faz sentido em quem salva UMA vez, ao fechar. Este salva a cada 1,2 s
// de pausa, e aparar aqui comeria a linha em branco que o mestre acabou de abrir
// para o próximo parágrafo — o cursor pularia para o fim da frase anterior no
// meio da noite.
func TestTheNoteIsNotTrimmedMidTyping(t *testing.T) {
	f := newSceneFixture(t)

	f.posta(t, f.mestre, f.tableUrl()+"/notas", `{"notes":"a cena acabou\n\n"}`)

	if got := f.dbNote(t); got != "a cena acabou\n\n" {
		t.Errorf("a nota foi aparada: %q", got)
	}
}

// O gesto que faz o checkbox valer.
//
// O estado do quadrinho mora NA NOTA, não ao lado dela: sem esta reescrita o
// controle seria enfeite e a marcação não sobreviveria a um F5. A linha viaja no
// CAMINHO, como os outros verbos de linha da Mesa.
func TestTheTaskCheckboxRewritesTheNote(t *testing.T) {
	f := newSceneFixture(t)
	nota := `{"notes":"- [ ] pagar o taverneiro\n- [x] dar o XP"}`

	corpo := f.posta(t, f.mestre, f.tableUrl()+"/notas/tarefa/0/marcar", nota)

	if got := f.dbNote(t); got != "- [x] pagar o taverneiro\n- [x] dar o XP" {
		t.Errorf("o quadrinho não reescreveu a nota: %q", got)
	}
	// A RESPOSTA redesenha a prévia, e é por ela que a tela do mestre muda: sem
	// o fragmento, a nota mudaria no banco e o quadrinho continuaria vazio na
	// tela até um F5 — que é a forma mais convincente de um botão parecer quebrado.
	if !strings.Contains(corpo, "table-notes-preview") {
		t.Error("a resposta não trouxe a prévia; o quadrinho mudaria só no banco")
	}
	// E o SINAL volta junto, que é o que atualiza a caixa de texto sem trocar o
	// nó — trocar o `<textarea>` por remendo apagaria o que o mestre digita.
	if !strings.Contains(corpo, "notas") {
		t.Error("a resposta não trouxe o sinal `notas`; a caixa ficaria com o texto velho")
	}
}

// O par do de cima. Sem ele o guarda mediria um interruptor de mão única e
// chamaria de alternância.
func TestUncheckingBringsTheCheckboxBack(t *testing.T) {
	f := newSceneFixture(t)

	f.posta(t, f.mestre, f.tableUrl()+"/notas/tarefa/0/desmarcar", `{"notes":"- [x] dar o XP"}`)

	if got := f.dbNote(t); got != "- [ ] dar o XP" {
		t.Errorf("desmarcar não voltou o quadrinho: %q", got)
	}
}

// A linha vem de um CLIQUE, e o cliente pode estar um remendo atrás do
// servidor — a nota mudou noutra aba e a tela ainda mostra a lista antiga. Isso
// é caminho NORMAL, não ataque: a resposta certa é devolver a nota intacta, e a
// errada é um `index out of range` derrubando o handler que estava salvando o
// texto de alguém.
func TestAnOutOfRangeLineDoesNotBringTheHandlerDown(t *testing.T) {
	f := newSceneFixture(t)

	rec := f.pede(t, f.mestre, "POST", f.tableUrl()+"/notas/tarefa/99/marcar", `{"notes":"- [ ] a"}`)

	if rec.Code >= 500 {
		t.Fatalf("uma linha fora da faixa derrubou o handler: %d", rec.Code)
	}
	if got := f.dbNote(t); got != "- [ ] a" {
		t.Errorf("a nota foi mexida por um clique fora da faixa: %q", got)
	}
}

// dbNote lê a coluna direto, que é o único lugar que decide se a nota
// existe. Ler a resposta do próprio handler seria perguntar ao acusado.
func (f sceneFixture) dbNote(t *testing.T) string {
	t.Helper()
	sess, err := f.s.queries.GetSession(t.Context(), f.sessionID)
	if err != nil {
		t.Fatalf("ler a sessão %d: %v", f.sessionID, err)
	}
	if !sess.Notes.Valid {
		return ""
	}
	return sess.Notes.String
}

// A prévia REMENDADA carrega os ids da mesa.
//
// Montá-la a partir de uma `View` SINTÉTICA, criada só com o texto, faz cada
// quadrinho do fragmento apontar para `/campanhas/0/sessoes/0/notas/tarefa/N/marcar`: struct
// nova nasce com `CampaignID` e `SessionID` em ZERO.
//
// O sintoma é mudo, e é por isso que ele merece guarda: o PRIMEIRO clique
// funciona, porque acontece sobre o HTML da carga fria, que tem os ids certos.
// Do segundo em diante a tela não muda — botão no lugar, `aria-checked`
// desenhado, nenhum erro em canto nenhum, e o banco parando de mudar.
//
// Um guarda que só afirmasse "a resposta traz a prévia" passaria verde: o
// fragmento ESTÁ lá, e está errado por dentro.
func TestThePatchedPreviewCarriesTheTableIds(t *testing.T) {
	f := newSceneFixture(t)

	corpo := f.posta(t, f.mestre, f.tableUrl()+"/notas", `{"notes":"- [ ] pagar o taverneiro"}`)

	// O CONTROLE: a prévia tem de trazer um quadrinho, senão não há caminho
	// nenhum para conferir e o teste passaria dizendo nada.
	if !strings.Contains(corpo, "notas/tarefa/") {
		t.Fatal("a prévia não trouxe quadrinho de tarefa — não há rota para conferir")
	}
	esperado := f.tableUrl() + "/notas/tarefa/0/marcar"
	if !strings.Contains(corpo, esperado) {
		t.Errorf("o quadrinho remendado não aponta para %s", esperado)
	}
	if strings.Contains(corpo, "/campanhas/0/sessoes/0/") {
		t.Error("o quadrinho remendado aponta para a mesa 0/0: a view da prévia nasceu sem os ids")
	}
}

// ── A JANELA PRÓPRIA ────────────────────────────────────────────────────────

// A cena tem a MESMA trava do comando.
//
// Ela é um endereço que o mestre pode favoritar, então ela é um endereço que
// qualquer um pode digitar — e as notas da sessão não são do jogador. O guarda
// existe porque uma cena NOVA não herda a trava de ninguém: a do `notesCommand`
// protege o POST, e o GET nasceu com a sua própria.
func TestTheNotesWindowIsTheGmsAlone(t *testing.T) {
	f := newSceneFixture(t)
	f.posta(t, f.mestre, f.tableUrl()+"/notas", `{"notes":"# O que o jogador não vê"}`)

	rec := f.pede(t, f.jogador, "GET", f.tableUrl()+"/notas", "")

	if rec.Code != 403 {
		t.Errorf("o jogador abriu as notas do mestre: %d", rec.Code)
	}
	// O CONTROLE do 403: um status certo com o corpo vazando o texto seria a
	// mesma coisa que não ter trava nenhuma.
	if strings.Contains(rec.Body.String(), "O que o jogador não vê") {
		// SEM O CORPO na mensagem: ele é a página inteira, e um `%q` de 8 KB
		// empurra o veredito para fora da tela de quem está lendo o vermelho.
		t.Errorf("a recusa veio com a nota dentro do corpo")
	}
}

// Três coisas na mesma asserção, e nenhuma é redundante: a cena traz o TEXTO
// (senão a janela abre vazia sobre uma nota que existe), traz a PRÉVIA já
// desenhada (é o markdown, não o cru), e traz o endereço de SALVAR com os ids
// certos — a mesma armadilha do `TestThePatchedPreviewCarriesTheTableIds`.
func TestTheNotesWindowDrawsTheNoteAndTheWayToSaveIt(t *testing.T) {
	f := newSceneFixture(t)
	f.posta(t, f.mestre, f.tableUrl()+"/notas", `{"notes":"# Cena 1\nO ogro **fugiu**"}`)

	rec := f.pede(t, f.mestre, "GET", f.tableUrl()+"/notas", "")

	if rec.Code != 200 {
		t.Fatalf("a janela das notas respondeu %d", rec.Code)
	}
	corpo := rec.Body.String()
	if !strings.Contains(corpo, "O ogro **fugiu**") {
		t.Errorf("a caixa de texto não veio com a nota")
	}
	if !strings.Contains(corpo, "<strong") {
		t.Errorf("a prévia não veio desenhada: o markdown chegou cru")
	}
	if !strings.Contains(corpo, f.tableUrl()+"/notas") {
		t.Errorf("a janela não sabe para onde salvar — o endereço com os ids não está no HTML")
	}
}

// As duas escrevem a MESMA coluna `sessions.notes`, e o autosave é de 1,2 s:
// com as duas abertas, quem salvar por último apaga o parágrafo do outro sem
// aviso, com as duas faixas dizendo "Salvo". A exclusão é um pacto entre
// documentos, e o que este guarda prende é a METADE que o servidor desenha —
// a janela ANUNCIA que tomou, e a Mesa ESCUTA.
//
// Ele afirma a chave literal de propósito: as duas pontas a escrevem em
// arquivos diferentes, e um renome que alcançasse só uma quebraria a exclusão
// sem quebrar compilação nenhuma. É a mesma forma dos sete canais de um sinal.
func TestTheNotesWindowAndTheColumnCannotBothHoldTheNotes(t *testing.T) {
	f := newSceneFixture(t)

	janela := f.pede(t, f.mestre, "GET", f.tableUrl()+"/notas", "").Body.String()
	mesa := f.pede(t, f.mestre, "GET", f.tableUrl(), "").Body.String()

	if !strings.Contains(janela, "t20:notas-janela") {
		t.Errorf("a janela não anuncia que tomou as notas: a coluna não teria como fechar")
	}
	if !strings.Contains(janela, "pagehide") {
		t.Errorf("a janela não devolve as notas ao fechar: a coluna ficaria trancada para sempre")
	}
	if !strings.Contains(mesa, "storage") || !strings.Contains(mesa, "t20:notas-janela") {
		t.Errorf("a Mesa não escuta o anúncio da janela: as duas caixas ficariam vivas juntas")
	}
}
