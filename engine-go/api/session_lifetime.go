package api

import (
	"context"
	"log"

	"t20engine/aovivo"
	"t20engine/db/sqlcgen"
	"t20engine/tabuleiro"
)

// O FIM DA VIDA de uma sessão, e de tudo que ela deixou em memória (ALE-270).
//
// A mesa roda de MEMÓRIA: o tabuleiro num mapa por sessão no `BoardStore`, a
// fila noutro no `SessionStore`. Apagar a linha do banco não esvazia nenhum dos
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
// deles passa a esquecer um store — que é literalmente o defeito desta issue,
// onde a cena chamava só o `Sessions().Forget`.
//
// Ela é chamada DEPOIS de a linha sair do banco, e a ordem importa numa direção
// só: avisar antes deixaria uma janela em que a sessão ainda responde e o
// estado em memória já não existe — uma requisição nesse instante recriaria o
// que se acabou de apagar.
func sessionDeleted(boards *tabuleiro.BoardStore, sessions *aovivo.SessionStore, sessionID int64) {
	boards.SessionDeleted(sessionID)
	sessions.SessionDeleted(sessionID)
}

// SessionDeleted é a porta do hospedeiro para a sequência acima.
func (s *Server) SessionDeleted(sessionID int64) {
	sessionDeleted(s.boards, s.sessions, sessionID)
}

// CampaignDeleted faz o mesmo para TODAS as sessões da campanha.
//
// Ela é chamada ANTES de apagar a campanha, e aqui a ordem é a inversa da de
// cima — por um motivo prosaico: apagar a campanha leva as sessões por cascata,
// e depois disso não há mais como perguntar quais eram.
//
// Falhar em LISTAR não impede o apagar, e a escolha é a mesma do `endBoard` com
// o `Archive`: o mestre mandou apagar a campanha, e recusar isso porque a
// faxina de memória não pôde ser planejada seria prender a mesa numa campanha
// que ele já descartou. O custo do que sobra é um alarme travado até o
// reinício, e ele fica REGISTRADO — sem esta linha, ninguém saberia por quê.
func campaignDeleted(
	ctx context.Context, q *sqlcgen.Queries,
	boards *tabuleiro.BoardStore, sessions *aovivo.SessionStore, campaignID int64,
) {
	sessoes, err := q.ListSessions(ctx, campaignID)
	if err != nil {
		log.Printf("campaign %d: não deu para listar as sessões antes de apagar (%v); "+
			"o estado em memória delas fica até o reinício", campaignID, err)
		return
	}
	for _, sess := range sessoes {
		sessionDeleted(boards, sessions, sess.ID)
	}
}

// CampaignDeleted é a porta do hospedeiro para a sequência acima.
func (s *Server) CampaignDeleted(ctx context.Context, campaignID int64) {
	campaignDeleted(ctx, s.queries, s.boards, s.sessions, campaignID)
}
