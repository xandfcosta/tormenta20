package table

import (
	"net/url"
)

// Os guardas do BESTIÁRIO DENTRO DA MESA (ALE-263).
//
// O desenho é o mesmo da cena do mestre e tem guarda lá; o que se prende aqui é
// o que só existe por ser DENTRO da mesa — a trava do papel, o envio para a
// fila, e a regra de quem é dono do rascunho.

// TestSendingToTheTablePutsOneRowPerCopy.
//
// Uma entrada por cópia, e quem numera os repetidos é o SERVIDOR (ALE-192): a
// tela não pode adivinhar um número que outro cliente acabou de usar. Todas
// entram com a MESMA iniciativa — é o que a mesa faz com um bando.
// signals escreve os sinais do jeito que o Datastar os manda num GET: um
// parâmetro `datastar` com o JSON inteiro.
//
// A primeira versão deste teste usava query params soltos (`?criatura=zumbi`), e
// eles NÃO são a mesma coisa: o `criteriosDoPedido` lê os dois, mas o
// `rascunhode` só existe como sinal — então o teste mandava um pedido que o
// navegador nunca manda, e o painel semeava por não achar o rascunho. O teste
// acusou o código por um defeito que era dele.
func signals(json string) string {
	return "?datastar=" + url.QueryEscape(json)
}
