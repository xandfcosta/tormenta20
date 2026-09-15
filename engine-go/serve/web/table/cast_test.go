package table

import (
	"strings"
)

// O ELENCO (ALE-269, superfície 6a) — os guardas.
//
// O que se prende aqui é a TRAVA e o que ela impede, não o desenho: o id do
// personagem viaja no CAMINHO, e caminho é digitável.

// TestTheGmDoesNotTrackWhoIsNotInTheCampaign é a trava, e ela existe
// porque o id vem da URL.
//
// Sem a conferência contra o roster, o mestre de uma mesa poria na fila dele o
// personagem de OUTRA campanha — e o personagem é o SNAPSHOT daquela campanha
// (ALE-33), então a linha nasceria ligada a uma ficha que aquela mesa não deve
// nem enxergar.
//
// O guarda vale mais que "o botão não aparece": o botão é cortesia; isto é o
// que responde a um `curl`.
func firstRows(s string, n int) string {
	linhas := strings.Split(s, "\n")
	if len(linhas) > n {
		linhas = linhas[:n]
	}
	return strings.Join(linhas, "\n")
}
