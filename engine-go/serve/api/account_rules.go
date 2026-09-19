package api

import (
	"t20engine/app/accounts"
	"t20engine/infra/config"
	"t20engine/infra/db/sqlcgen"
)

// O QUE SOBROU DAS REGRAS DE CONTA DESTE LADO: quem a REQUISIÇÃO carrega.
//
// Autenticar, cadastrar, assinar a sessão e redefinir a senha moravam aqui e
// hoje moram no `app/accounts` — eram regra pregada ao primeiro transporte que a
// alcançou, e a marca disso era um `issueSession` que recebia um
// `http.ResponseWriter` para escrever um 500 no meio da assinatura (ALE-349).
//
// O que fica é tradução de transporte: achar o token no biscoito ou no
// cabeçalho, perguntar ao caso de uso de quem ele é, e vestir a linha do banco
// no `AuthUser` que as telas leem. O `*sql.DB` saiu junto — nenhuma transação
// começa mais deste lado.
type accountRules struct {
	cfg     config.Config
	queries *sqlcgen.Queries
	gate    accounts.Gate
}

func (s *Server) accountRules() accountRules {
	return accountRules{cfg: s.cfg, queries: s.queries, gate: s.accountGate()}
}

// accountGate é a porta de entrar e cadastrar-se, e ela recebe o `*sql.DB`
// porque criar conta gasta o convite na MESMA transação.
func (s *Server) accountGate() accounts.Gate {
	return accounts.NewGate(s.db, s.queries, s.cfg)
}

// accountResets é a redefinição de senha por link.
//
// Ela se monta SOBRE o portão e não ao lado dele: o custo do bcrypt é um só, e
// dois lugares escolhendo esse número divergiriam sem ninguém notar — as senhas
// continuariam funcionando.
func (s *Server) accountResets() accounts.Resets {
	return accounts.NewResets(s.accountGate())
}
