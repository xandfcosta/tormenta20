package api

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"t20engine/account"
	"t20engine/db"
	"t20engine/db/sqlcgen"
	"t20engine/web/door"
)

// A PORTA, com adaptador próprio (ALE-278, fatia 6).
//
// `doorHost` é o núcleo mais as REGRAS DE CONTA E SESSÃO — e não o `*Server`.
// Ele é o adaptador que mais ganhou com a divisão: das nove assinaturas que a
// porta pede, sete são autenticação, cadastro e redefinição de senha, e todas
// as sete vivem agora num tipo que diz o que elas são (`accountRules`) em vez
// de num tipo que diz onde elas estavam.
// As regras vêm por CAMPO e não embutidas, e o compilador é que decidiu: as
// duas partes carregam um `queries`, então embutir as duas deixa `h.queries`
// ambíguo. A ambiguidade é um sintoma honesto — são dois caminhos para a mesma
// conexão —, e nomear o campo é o que faz cada chamada dizer de qual das duas
// coisas ela está falando.
type doorHost struct {
	sceneCore
	accounts accountRules
}

func (s *Server) doorHost() doorHost {
	return doorHost{sceneCore: s.sceneCore(), accounts: s.accountRules()}
}

// O QUE O HOSPEDEIRO DEVE À CENA DA PORTA (ALE-278).
//
// A `door.Deps` é declarada lá, no consumidor — é isso que a torna uma porta e
// não um segundo nome para o objeto-deus. O que mora aqui é o cumprimento dela,
// e ele é fino de propósito: cada método é um invólucro sobre o que o `Server`
// já fazia, com o nome exportado que a interface pede.
//
// Três deles fazem mais que embrulhar, e as três razões são a mesma decisão de
// FRONTEIRA vista de três ângulos.

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

func (h doorHost) Authenticate(ctx context.Context, email, password string) (sqlcgen.User, error) {
	return h.accounts.authenticate(ctx, email, password)
}

func (h doorHost) CreateAccount(ctx context.Context, body account.RegisterBody) (sqlcgen.User, error) {
	return h.accounts.createAccount(ctx, body)
}

func (h doorHost) IssueSession(w http.ResponseWriter, user sqlcgen.User) bool {
	return h.accounts.issueSession(w, user)
}

// ResetLinkOwner junta as duas perguntas que a cena fazia em sequência — o link
// vale? de quem é a conta? — numa só.
//
// Elas eram `usableReset` seguida de `GetUserByID`, e a cena carregava o
// `sqlcgen.PasswordReset` no meio só para ter o `Userid`. A linha do banco não
// interessa à tela: o que ela mostra é o e-mail, para quem clicou saber que está
// mudando a conta certa.
func (h doorHost) ResetLinkOwner(ctx context.Context, token string) (string, bool) {
	reset, ok := h.accounts.usableReset(ctx, token)
	if !ok {
		return "", false
	}
	user, err := h.queries.GetUserByID(ctx, reset.Userid)
	if err != nil {
		return "", false
	}
	return user.Email, true
}

// ResetPassword é o caminho inteiro, e ele fica DESTE lado por causa do bcrypt.
//
// A cena gerava o hash ela mesma, com o `bcryptCost` daqui. Isso obrigaria a
// porta a carregar uma constante de custo criptográfico para a tela fazer
// trabalho que não é dela — e o custo do bcrypt é decisão de segurança do
// servidor, não de quem desenha o formulário.
func (h doorHost) ResetPassword(ctx context.Context, token, password string) bool {
	reset, ok := h.accounts.usableReset(ctx, token)
	if !ok {
		return false
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return false
	}
	return h.accounts.applyReset(ctx, reset, string(hash)) == nil
}

// SignUpRefusal CLASSIFICA o erro; quem escolhe a frase é a cena.
//
// Os sentinelas que ela distingue são valores deste pacote, e a cena os lia
// direto — alcance que a divisão existe para cortar. A repartição é: o
// hospedeiro sabe o que os erros dele significam, a cena sabe o que o jogador
// lê. O motivo devolvido é um tipo da CENA, porque o vocabulário de recusa é
// dela; se ele fosse daqui, a voz da porta passaria a morar no `api`.
func (h doorHost) SignUpRefusal(err error) (door.RefusalMotive, int) {
	switch {
	case db.IsUniqueViolation(err):
		return door.RefusalEmailTaken, http.StatusConflict
	case errors.Is(err, errInviteRejected), errors.Is(err, errInviteSpent):
		return door.RefusalBadInvite, http.StatusForbidden
	default:
		return door.RefusalInternal, http.StatusInternalServerError
	}
}
