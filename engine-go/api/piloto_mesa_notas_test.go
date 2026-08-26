package api

import (
	"strings"
	"testing"
)

// AS NOTAS DA SESSÃO (ALE-269, superfície 5) — os guardas do caminho.
//
// A GRAMÁTICA já tem os dela, com paridade medida contra o JS
// (`piloto_markdown_test.go`). O que se prende AQUI é o que só a composição
// mostra: quem pode escrever, o que chega ao banco, e o que a resposta redesenha.

// TestOJogadorNaoEscreveNasNotasDoMestre é a trava, e ela é do SERVIDOR.
//
// O botão que não aparece é cortesia para quem não pode; a segurança é o 403.
// Um guarda que só afirmasse "o jogador não vê o painel" mediria UX e deixaria a
// rota aberta para quem postasse na mão.
func TestOJogadorNaoEscreveNasNotasDoMestre(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.jogador, "POST", f.urlDaMesa()+"/notas", `{"notas":"eu escrevi isto"}`)

	if rec.Code != 403 {
		t.Errorf("o jogador escreveu nas notas do mestre: %d", rec.Code)
	}
	// O CONTROLE do 403: se a nota tivesse sido gravada, o status sozinho não
	// contaria — já houve rota que recusava DEPOIS de escrever.
	if nota := f.notaNoBanco(t); nota != "" {
		t.Errorf("a recusa veio depois da escrita: o banco tem %q", nota)
	}
}

// TestASalvaguardaDaNotaChegaAoBanco é o caminho feliz, ponta a ponta pelo
// roteador de verdade.
func TestASalvaguardaDaNotaChegaAoBanco(t *testing.T) {
	f := novoPiloto(t)

	f.posta(t, f.mestre, f.urlDaMesa()+"/notas", `{"notas":"# Cena 1\nO ogro fugiu"}`)

	if got := f.notaNoBanco(t); got != "# Cena 1\nO ogro fugiu" {
		t.Errorf("a nota no banco é %q", got)
	}
}

// TestANotaNaoEhAparadaNoMeioDaDigitacao.
//
// O handler JSON da SPA passa o texto por `trimOrNull` porque salva UMA vez, ao
// fechar. Este salva a cada 1,2 s de pausa, e aparar aqui comeria a linha em
// branco que o mestre acabou de abrir para escrever o próximo parágrafo — o
// cursor pularia para o fim da frase anterior no meio da noite.
func TestANotaNaoEhAparadaNoMeioDaDigitacao(t *testing.T) {
	f := novoPiloto(t)

	f.posta(t, f.mestre, f.urlDaMesa()+"/notas", `{"notas":"a cena acabou\n\n"}`)

	if got := f.notaNoBanco(t); got != "a cena acabou\n\n" {
		t.Errorf("a nota foi aparada: %q", got)
	}
}

// TestOQuadrinhoDaTarefaReescreveANota é o gesto que faz o checkbox valer.
//
// O estado do quadrinho mora NA NOTA, não ao lado dela: sem esta reescrita o
// controle seria enfeite e a marcação não sobreviveria a um F5. A linha viaja no
// CAMINHO, como os outros verbos de linha da Mesa.
func TestOQuadrinhoDaTarefaReescreveANota(t *testing.T) {
	f := novoPiloto(t)
	nota := `{"notas":"- [ ] pagar o taverneiro\n- [x] dar o XP"}`

	corpo := f.posta(t, f.mestre, f.urlDaMesa()+"/notas/tarefa/0/marcar", nota)

	if got := f.notaNoBanco(t); got != "- [x] pagar o taverneiro\n- [x] dar o XP" {
		t.Errorf("o quadrinho não reescreveu a nota: %q", got)
	}
	// A RESPOSTA redesenha a prévia, e é por ela que a tela do mestre muda: sem
	// o fragmento, a nota mudaria no banco e o quadrinho continuaria vazio na
	// tela até um F5 — que é a forma mais convincente de um botão parecer quebrado.
	if !strings.Contains(corpo, "mesa-notas-previa") {
		t.Error("a resposta não trouxe a prévia; o quadrinho mudaria só no banco")
	}
	// E o SINAL volta junto, que é o que atualiza a caixa de texto sem trocar o
	// nó — trocar o `<textarea>` por remendo apagaria o que o mestre digita.
	if !strings.Contains(corpo, "notas") {
		t.Error("a resposta não trouxe o sinal `notas`; a caixa ficaria com o texto velho")
	}
}

// TestODesmarcarVoltaOQuadrinho — o par do de cima. Sem ele o guarda mediria um
// interruptor de mão única e chamaria de alternância.
func TestODesmarcarVoltaOQuadrinho(t *testing.T) {
	f := novoPiloto(t)

	f.posta(t, f.mestre, f.urlDaMesa()+"/notas/tarefa/0/desmarcar", `{"notas":"- [x] dar o XP"}`)

	if got := f.notaNoBanco(t); got != "- [ ] dar o XP" {
		t.Errorf("desmarcar não voltou o quadrinho: %q", got)
	}
}

// TestALinhaDeForaNaoDerrubaOHandler.
//
// A linha vem de um CLIQUE, e o cliente pode estar um remendo atrás do
// servidor — a nota mudou noutra aba e a tela ainda mostra a lista antiga. Isso
// é caminho NORMAL, não ataque: a resposta certa é devolver a nota intacta, e a
// errada é um `index out of range` derrubando o handler que estava salvando o
// texto de alguém.
func TestALinhaDeForaNaoDerrubaOHandler(t *testing.T) {
	f := novoPiloto(t)

	rec := f.pede(t, f.mestre, "POST", f.urlDaMesa()+"/notas/tarefa/99/marcar", `{"notas":"- [ ] a"}`)

	if rec.Code >= 500 {
		t.Fatalf("uma linha fora da faixa derrubou o handler: %d", rec.Code)
	}
	if got := f.notaNoBanco(t); got != "- [ ] a" {
		t.Errorf("a nota foi mexida por um clique fora da faixa: %q", got)
	}
}

// notaNoBanco lê a coluna direto, que é o único lugar que decide se a nota
// existe. Ler a resposta do próprio handler seria perguntar ao acusado.
func (f pilotoFixture) notaNoBanco(t *testing.T) string {
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

// TestAPreviaRemendadaCarregaOsIdsDaMesa prende um defeito MEDIDO no navegador.
//
// A prévia da resposta era montada a partir de uma `mesaView` SINTÉTICA, criada
// só com o texto — e uma struct nova nasce com `CampaignID` e `SessionID` em
// ZERO. Cada quadrinho do fragmento saía apontando para
// `/piloto/mesa/0/0/notas/tarefa/N/marcar`.
//
// O SINTOMA É DA PIOR FAMÍLIA DESTA BASE, e é por isso que ele merece guarda: o
// PRIMEIRO clique funcionava, porque acontece sobre o HTML da carga fria, que
// tem os ids certos. Do segundo em diante a tela ficava MUDA — botão no lugar,
// `aria-checked` desenhado, nenhum erro em canto nenhum, e o banco parando de
// mudar. Foi preciso ler o `data-on:click` do nó vivo para ver o `0/0`.
//
// Um guarda que só afirmasse "a resposta traz a prévia" passaria verde sobre
// isto: o fragmento ESTAVA lá, e estava errado por dentro.
func TestAPreviaRemendadaCarregaOsIdsDaMesa(t *testing.T) {
	f := novoPiloto(t)

	corpo := f.posta(t, f.mestre, f.urlDaMesa()+"/notas", `{"notas":"- [ ] pagar o taverneiro"}`)

	// O CONTROLE: a prévia tem de trazer um quadrinho, senão não há caminho
	// nenhum para conferir e o teste passaria dizendo nada.
	if !strings.Contains(corpo, "notas/tarefa/") {
		t.Fatal("a prévia não trouxe quadrinho de tarefa — não há rota para conferir")
	}
	esperado := f.urlDaMesa() + "/notas/tarefa/0/marcar"
	if !strings.Contains(corpo, esperado) {
		t.Errorf("o quadrinho remendado não aponta para %s", esperado)
	}
	if strings.Contains(corpo, "/mesa/0/0/") {
		t.Error("o quadrinho remendado aponta para a mesa 0/0: a view da prévia nasceu sem os ids")
	}
}
