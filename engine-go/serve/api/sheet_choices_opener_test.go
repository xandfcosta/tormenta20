package api

import (
	"html"
	"strings"
	"testing"
)

// O Escolher de um humano sem atributo gravado escrevia `$race_attributes = null`. O Datastar manda o
// sinal nulo como `""`, o `[]string` do servidor recusa o corpo inteiro, e TODOS os sinais do pedido
// caíam: com o Escolher preguiçoso, o diálogo se esvaziava no primeiro gesto dentro dele.
func TestTheChoicesOpenerNeverWritesANullList(t *testing.T) {
	f, id := humanBarbarian(t)
	page := html.UnescapeString(powerScreen(t, f, id))
	if strings.Contains(page, "$race_attributes = null") {
		t.Error("o botão Escolher escreve $race_attributes = null; a lista vazia é []")
	}
	if !strings.Contains(page, "$race_attributes = []") {
		t.Error("o botão Escolher não escreve a lista de atributos da raça")
	}
}
