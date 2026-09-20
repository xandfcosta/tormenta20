package api

import (
	"t20engine/app/boards"
	"t20engine/app/session"
)

// O FIM DA VIDA de uma sessão, e de tudo que ela deixou em memória.
//
// A mesa roda de MEMÓRIA: o tabuleiro num mapa por sessão no `boards.Store`, a
// fila noutro no `session.Store`. Apagar a linha do banco não esvazia nenhum dos
// dois, e o que sobra não é inerte — o `Persist` seguinte bate na chave
// estrangeira, acende o `Dirty`, e a marca não sai mais: só um `Persist` bem
// sucedido a apaga, e nenhum vai suceder.
//
// # Por que isto mora no hospedeiro
//
// Porque é COMPOSIÇÃO, e nenhum dos dois stores conhece o outro. Cada um sabe
// esquecer o que é dele (`SessionDeleted`); quem sabe que os dois têm de ser
// avisados juntos é quem os montou. Uma cena chamando os dois na mão seria a
// terceira cópia da mesma sequência — e é a terceira que esquece o segundo.
//
// O que NÃO está aqui é o banco: `open_boards` sai por CASCATA com a sessão
// (migração 00010), e a fila mora na própria linha da sessão.

// sessionDeleted é A SEQUÊNCIA, escrita uma vez.
//
// Função livre sobre os dois stores, e não método de um deles: o `Server` e o
// adaptador da Mesa chegam aqui por caminhos diferentes, e os dois precisam
// EXATAMENTE do mesmo par de avisos. Escrevê-lo nos dois lugares é como um
// deles passa a esquecer um store.
//
// Ela é chamada DEPOIS de a linha sair do banco, e a ordem importa numa direção
// só: avisar antes deixaria uma janela em que a sessão ainda responde e o
// estado em memória já não existe — uma requisição nesse instante recriaria o
// que se acabou de apagar.
func sessionDeleted(boards *boards.Store, sessions *session.Store, sessionID int64) {
	boards.SessionDeleted(sessionID)
	sessions.SessionDeleted(sessionID)
}

// SessionDeleted é a porta do hospedeiro para a sequência acima.
func (s *Server) SessionDeleted(sessionID int64) {
	sessionDeleted(s.boards, s.sessions, sessionID)
}

// Aqui morava a faxina de memória das sessões de uma campanha apagada, e ela
// não existe mais neste pacote: a ORDEM — esquecer antes de apagar — é parte do
// gesto, e gesto é do caso de uso. Hoje é o `campaign.Lifecycle.Delete`, e a
// sequência tem um teste que a acusa invertida (ALE-359).
