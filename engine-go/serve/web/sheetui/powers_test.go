package sheetui

import (
	"strings"
)

// Os guardas da aba PODERES (ALE-272, fatia 8).
//
// O que eles prendem é a REGRA — o que o personagem possui, o que ele pode
// ativar agora, e o que entrar numa postura custa — e a decisão de tela que
// separa o que se USA do que só está lá.

// powerPanel é a ficha das posturas: a Fúria é a postura de escala do livro e a
// Alma de Bronze é o único poder do catálogo que CONCEDE algo ao entrar nela.
// powerPanel corta a LISTA, deixando de fora os diálogos que vêm depois
// dela — o de escolher poderes mostra o catálogo inteiro de opções.
// powerPanel corta a LISTA, deixando de fora os diálogos que vêm depois
// dela — o de escolher poderes mostra o catálogo inteiro de opções.
func powerPanel(screen string) string {
	// O CORTE é no ABRIR do primeiro diálogo, e não no primeiro `</section>`: as
	// duas seções da lista são `<section>` ANINHADAS, e cortar no primeiro
	// fechamento deixaria de fora justamente as passivas. Os diálogos começam
	// depois do painel, e todos são sobreposições de tela cheia.
	end := strings.Index(screen, `class="fixed inset-0`)
	if end < 0 {
		return screen
	}
	return screen[:end]
}

func actionsSlice(screen string) string {
	start := strings.Index(screen, ">Ações</h3>")
	if start < 0 {
		return ""
	}
	end := strings.Index(screen[start:], "Passivas ·")
	if end < 0 {
		return screen[start:]
	}
	return screen[start : start+end]
}
