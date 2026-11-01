package table

import "fmt"

// O AVISO DE VERSÃO NOVA.
//
// Depois de um deploy no meio do jogo, a página aberta segue com o CSS e o JS antigos
// enquanto recebe HTML novo, e o que está novo na tela pode sair sem estilo. Ela nunca
// recarrega sozinha, porque isso apagaria o que alguém está digitando: quem decide a
// hora é a pessoa, pelo botão.

const (
	pageBuildSignal   = "page_build"
	serverBuildSignal = "server_build"
)

// buildSignals declara a versão com que a página nasceu. Ela entra na raiz `#table`, que o
// remendo nunca toca, e por isso continua sendo a antiga depois do deploy.
func buildSignals(revision string) string {
	return fmt.Sprintf("%s: '%s', %s: ''", pageBuildSignal, revision, serverBuildSignal)
}

// pageIsOutdated acende o aviso. Vazio em qualquer ponta não acende: um binário sem
// carimbo não sabe dizer se é outra versão.
var pageIsOutdated = fmt.Sprintf("$%s !== '' && $%s !== '' && $%s !== $%s",
	pageBuildSignal, serverBuildSignal, pageBuildSignal, serverBuildSignal)
