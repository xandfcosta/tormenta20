package live

import "fmt"

// O CICLO DE VIDA DE UMA SESSÃO, como DECISÃO e nada mais.
//
// Nenhuma linha daqui sabe o que é um banco. "Iniciar" e "encerrar" não são um
// `UPDATE`: são uma pergunta sobre o status em que a sessão está e o que o
// mestre pediu — e a resposta é qual escrita fazer, que quem orquestra executa.
//
// Separado assim porque a decisão é a parte que tem CASO e ARMADILHA, e a
// escrita é a parte que tem encanamento. O teste desta decisão não semeia banco,
// não abre transação e não monta servidor.

// StatusChange é o que precisa acontecer com a linha da sessão.
type StatusChange string

const (
	// StatusUnchanged é "já está assim", e NÃO é erro.
	StatusUnchanged StatusChange = "unchanged"
	// StatusStartsFresh carimba o início de uma noite que nunca começou.
	StatusStartsFresh StatusChange = "starts-fresh"
	// StatusReopens devolve ao ar uma sessão encerrada, com a fila e o
	// tabuleiro dela.
	StatusReopens StatusChange = "reopens"
	// StatusEnds tira a partida do ar.
	StatusEnds StatusChange = "ends"
)

// Os três status que a coluna `status` da sessão guarda.
const (
	StatusPlanned = "planned"
	StatusActive  = "active"
	StatusEnded   = "ended"
)

// ChangeToStatus diz o que fazer para a sessão chegar ao status pedido.
//
// As duas decisões que não são óbvias, e que custaram escolha:
//
//   - **Pedir "ativa" numa sessão ENCERRADA reabre**, em vez de recusar. A noite
//     continuou, e obrigar a criar uma sessão nova perderia a fila e o
//     tabuleiro dela.
//   - **Pedir "encerrada" numa sessão PLANEJADA é recusado**, e isso é diferente
//     do clique repetido: encerrar o que nunca começou é um gesto sobre a coisa
//     errada, e carimbar um fim numa noite sem início deixaria o histórico
//     dizendo que ela aconteceu.
//
// Clicar duas vezes NUNCA é erro — é o gesto de quem não viu a tela mudar, e
// recusar seria punir a dúvida.
//
//	live.ChangeToStatus("ended", "active") // live.StatusReopens, nil
func ChangeToStatus(current, wanted string) (StatusChange, error) {
	switch wanted {
	case StatusActive:
		switch current {
		case StatusActive:
			return StatusUnchanged, nil
		case StatusEnded:
			return StatusReopens, nil
		default:
			return StatusStartsFresh, nil
		}
	case StatusEnded:
		switch current {
		case StatusEnded:
			return StatusUnchanged, nil
		case StatusPlanned:
			return "", fmt.Errorf("a sessão está %q e nunca foi iniciada; não há o que encerrar", current)
		default:
			return StatusEnds, nil
		}
	}
	return "", fmt.Errorf("status pedido %q: só %q e %q podem ser pedidos", wanted, StatusActive, StatusEnded)
}

// EmptyTrackerJSON é a fila VAZIA, como a coluna `runtimeState` a guarda.
//
// Ela mora aqui e não em quem escreve porque tem DOIS escritores — criar a
// sessão e reiniciar o combate —, e a forma serializada do estado vazio é do
// regime ao vivo, não de quem o grava. Duas cópias divergiriam no dia em que o
// estado ganhasse um campo, e o sintoma seria uma sessão nova com a fila de
// outra forma que a reiniciada.
const EmptyTrackerJSON = `{"initiative":[],"round":0,"turnIndex":-1}`
