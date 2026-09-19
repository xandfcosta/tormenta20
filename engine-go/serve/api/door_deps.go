package api

import (
	"net/http"

	"t20engine/infra/db/sqlcgen"
)

// A PORTA, com adaptador próprio — e ele encolheu de NOVE métodos para DOIS.
//
// Os outros sete eram conta e senha, e hoje a cena os chama direto no
// `app/accounts`: autenticar, cadastrar, classificar a recusa do cadastro,
// conferir o link e redefinir. Um deles, o `SignUpRefusal`, existia só para
// traduzir sentinelas deste pacote num vocabulário que a cena declarava — e com
// as recusas exportadas pelo caso de uso não sobrou tradução (ALE-349).
//
// O `Queries()` saiu junto: ele estava na porta para UMA leitura, o e-mail do
// dono do link de redefinição, e essa leitura é do `accounts.Resets`.
type doorHost struct {
	sceneCore
	accounts accountRules
}

func (s *Server) doorHost() doorHost {
	return doorHost{sceneCore: s.sceneCore(), accounts: s.accountRules()}
}

// HasSession é `sessionUser` reduzida à pergunta que a cena faz.
//
// A cena só quer saber SE há sessão, para mandar quem já entrou para longe da
// tela de login. Devolver o `AuthUser` obrigaria a cena a conhecer um tipo do
// `api` — e ela não pode importar o `api`, que a importa de volta para montar
// rota.
func (h doorHost) HasSession(r *http.Request) bool {
	_, err := h.accounts.sessionUser(r)
	return err == nil
}

// IssueSession escreve o biscoito. Ele fica deste lado porque a política do
// cookie é transporte; quem ASSINA a sessão é o `accounts.Gate`.
func (h doorHost) IssueSession(w http.ResponseWriter, user sqlcgen.User) bool {
	return h.accounts.issueSession(w, user)
}
