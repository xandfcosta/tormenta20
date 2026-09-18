// Package app é o VOCABULÁRIO da camada de aplicação: quem pede, e como um caso
// de uso recusa.
//
// Ele não tem comportamento e não vai ter. O que mora aqui é o que TODO caso de
// uso precisa dizer e que, escrito em cada pacote, seria a mesma decisão tomada
// várias vezes — e, no dia em que uma delas mudasse, duas grafias para o mesmo
// conceito.
//
// A regra do grupo está no `boundary` (`TestNoLayerImportsUpwards`): `app/` pode
// `domain/` e `infra/`, e não pode `serve/`.
package app

import "errors"

// AS RECUSAS, tipadas. Quem traduz para um número é o transporte.
//
// Um caso de uso que devolvesse 403 não poderia ser chamado de outro transporte
// — que é a única coisa que esta camada compra. A tradução mora em quem
// responde: no `serve/api` para a API JSON, na cena para o Datastar.
var (
	// ErrNotFound é "não existe, ou não é seu de alcançar" — os dois casos
	// respondem a mesma coisa de propósito: distinguir contaria a quem não
	// pertence à campanha quais sessões existem nela.
	ErrNotFound = errors.New("não encontrado")
	// ErrForbidden é "existe, você o alcança, mas este gesto não é seu".
	ErrForbidden = errors.New("não é seu")
	// ErrRefused é a REGRA dizendo não — encerrar o que nunca começou.
	ErrRefused = errors.New("recusado pela regra")
)

// Caller é quem está pedindo, na forma que um caso de uso precisa: um id e se
// administra.
//
// Não é o usuário inteiro, e não é o `*http.Request`: os dois são do
// transporte, e um caso de uso que os recebesse não poderia ser chamado de
// outro lugar.
type Caller struct {
	ID      int64
	IsAdmin bool
}

// Os dois papéis de quem senta numa campanha.
const (
	RoleGM     = "gm"
	RolePlayer = "player"
)
