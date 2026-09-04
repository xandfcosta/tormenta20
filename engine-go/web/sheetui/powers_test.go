package sheetui

import (
	"strings"
)

// Os guardas da aba PODERES (ALE-272, fatia 8).
//
// O que eles prendem é a REGRA — o que o personagem possui, o que ele pode
// ativar agora, e o que entrar numa postura custa — e a decisão de tela que
// separa o que se USA do que só está lá.

// oBarbaro é a ficha das posturas: a Fúria é a postura de escala do livro e a
// Alma de Bronze é o único poder do catálogo que CONCEDE algo ao entrar nela.
// oPainelDosPoderes corta a LISTA, deixando de fora os diálogos que vêm depois
// dela — o de escolher poderes mostra o catálogo inteiro de opções.
// oPainelDosPoderes corta a LISTA, deixando de fora os diálogos que vêm depois
// dela — o de escolher poderes mostra o catálogo inteiro de opções.
func powerPanel(tela string) string {
	// O CORTE é no ABRIR do primeiro diálogo, e não no primeiro `</section>`: as
	// duas seções da lista são `<section>` ANINHADAS, e cortar no primeiro
	// fechamento deixaria de fora justamente as passivas. Os diálogos começam
	// depois do painel, e todos são sobreposições de tela cheia.
	fim := strings.Index(tela, `class="fixed inset-0`)
	if fim < 0 {
		return tela
	}
	return tela[:fim]
}

func actionsSlice(tela string) string {
	inicio := strings.Index(tela, ">Ações</h3>")
	if inicio < 0 {
		return ""
	}
	fim := strings.Index(tela[inicio:], "Passivas ·")
	if fim < 0 {
		return tela[inicio:]
	}
	return tela[inicio : inicio+fim]
}
