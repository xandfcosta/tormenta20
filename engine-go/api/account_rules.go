package api

import (
	"database/sql"
	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// AS REGRAS DE CONTA E DE SESSÃO, com casa própria (ALE-278, fatia 6).
//
// Elas eram doze métodos do `*Server`, e estavam nele por proximidade e não por
// pertencimento: quem as chamava era o `handleLogin`, o `handleRegister` e o
// `requireAuth`, todos do servidor, então elas nasceram lá. Os dois primeiros
// foram apagados na ALE-277 por não terem consumidor, e o que sobrou tornou a
// pergunta inevitável — quem chama `authenticate` hoje é a PORTA, uma cena.
//
// Elas não viraram métodos do adaptador da porta porque não são só da porta: o
// `sessionUser` e o `verifyToken` são de quem barra requisição, e barrar é do
// hospedeiro. Um tipo próprio é o que deixa as duas coisas verdadeiras ao mesmo
// tempo — a porta embute um `accountRules`, o `*Server` guarda um, e nenhum dos
// dois é dono do outro.
//
// O que ele carrega é o mínimo que essas regras leem: o segredo e a política do
// biscoito (`cfg`), as consultas, e o `*sql.DB` porque criar conta gasta o
// convite na MESMA transação — meia conta criada com convite gasto é a pior das
// duas metades.
type accountRules struct {
	cfg     plataforma.Config
	db      *sql.DB
	queries *sqlcgen.Queries
}

func (s *Server) accountRules() accountRules {
	return accountRules{cfg: s.cfg, db: s.db, queries: s.queries}
}
