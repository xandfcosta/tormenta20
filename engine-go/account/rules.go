// Package account é o que uma CONTA aceita: o que é um e-mail, o que é uma
// senha, e a forma dos dois pedidos que criam sessão.
//
// Ele saiu do `api` na ALE-278, junto com a extração da cena da porta, e o
// motivo não foi arrumação: eram DUAS cópias da mesma regra, e elas já tinham
// divergido.
//
// # As duas cópias, medidas
//
// O `api` tinha `validateRegister`/`validateLogin`/`validatePassword` com as
// mensagens em pt-BR, e `ValidateRegister`/`ValidateLogin`/`ValidatePassword`
// com as mesmas regras em inglês. Não era código morto esperando limpeza: a
// `ValidatePassword` inglesa era chamada pela rota JSON que redefine a senha, e
// a portuguesa pela tela da porta. **A mesma regra recusando com dois textos**,
// e um deles na língua que a regra de idioma proíbe para o que um humano lê.
//
// As outras duas grafias inglesas eram mesmo dívida: a `ValidateLogin` não tinha
// chamador nenhum, e a `ValidateRegister` tinha exatamente um — um teste, que
// afirmava as frases em inglês. Mudar o mínimo da senha na cópia VIVA deixava
// aquele teste verde, porque ele prendia a outra.
//
// # Por que um pacote, e não a porta
//
// A regra é lida pela cena da porta E pela API JSON, e depende só do
// `plataforma`. O destino de uma função é a DEPENDÊNCIA dela — pô-la em
// `web/door` faria a API JSON importar um pacote de CENA para validar, que é o
// contrário da direção que a divisão existe para criar.
//
// É a mesma forma do `search`, e pelo mesmo motivo declarado lá: quando uma
// função pura fica hospedada num pacote grande, quem não pode importar aquele
// pacote escreve uma cópia — e a cópia sai errada de um jeito que compila. Aqui
// ela já tinha saído.
//
// Ela NÃO vai para `plataforma` de propósito: aquele pacote é infraestrutura sem
// domínio, e "a senha precisa ter ao menos 8 caracteres" é regra de PRODUTO.
// Quem a mudar está mudando o que o jogador pode fazer, não como o servidor
// escreve JSON.
package account

import (
	"regexp"
	"unicode/utf8"

	"t20engine/plataforma"
)

// LoginBody e RegisterBody são a forma dos dois pedidos, e os nomes de campo
// JSON são contrato de FIO — não se renomeiam junto com o pacote.
type LoginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterBody struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Name     *string `json:"name"`
	// InviteToken is the single-use link the admin handed the player. Required
	// for everyone but the ADMIN_EMAILS addresses (ALE-120).
	InviteToken string `json:"inviteToken"`
}

// AS MENSAGENS SÃO AS QUE O JOGADOR LÊ, então são em pt-BR (ALE-229).
//
// Eram em inglês — herança das frases do class-validator do NestJS —, e a SPA
// escondia isso validando com Zod antes de chamar. Servidor-renderizado elas
// chegam na cara de quem digitou, e o buraco já existia: uma senha de 200
// caracteres passa pelo Zod (que só checa o mínimo) e volta "password must be
// shorter than or equal to 128 characters".
//
// A rota JSON de redefinir senha respondia exatamente essas frases em inglês até
// a ALE-278, porque chamava a outra cópia. Agora há uma só.
const (
	msgEmailInvalido = "E-mail inválido"
	msgSenhaCurta    = "A senha precisa ter ao menos 8 caracteres"
	msgSenhaLonga    = "A senha pode ter no máximo 128 caracteres"
	msgSenhaVazia    = "Informe sua senha"
	msgNomeLongo     = "O nome pode ter no máximo 80 caracteres"
)

// Um teste pragmático de FORMA de e-mail — o `IsEmail` do class-validator é mais
// estrito, e a diferença não vale a superfície: quem digita errado descobre no
// convite que não chega.
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func IsEmail(s string) bool { return emailRe.MatchString(s) }

// ValidateRegister recusa o que não vira conta: e-mail sem forma de e-mail,
// senha fora da faixa, nome longo demais.
//
//	if fields := account.ValidateRegister(body); len(fields) > 0 { … }
func ValidateRegister(b RegisterBody) plataforma.FieldErrorMap {
	f := plataforma.FieldErrorMap{}
	if !IsEmail(b.Email) {
		f["email"] = []string{msgEmailInvalido}
	}
	for field, messages := range ValidatePassword(b.Password) {
		f[field] = messages
	}
	if b.Name != nil && utf8.RuneCountInString(*b.Name) > 80 {
		f["name"] = []string{msgNomeLongo}
	}
	return f
}

// ValidatePassword é A regra de senha, uma só, dividida pelo registro e pelo
// link de redefinição (ALE-120) — duas grafias de "ao menos 8" divergiriam, e a
// tela que ficasse mais frouxa seria a que importa.
func ValidatePassword(password string) plataforma.FieldErrorMap {
	f := plataforma.FieldErrorMap{}
	length := utf8.RuneCountInString(password)
	if length < 8 {
		f["password"] = append(f["password"], msgSenhaCurta)
	}
	if length > 128 {
		f["password"] = append(f["password"], msgSenhaLonga)
	}
	return f
}

// ValidateLogin não confere a FAIXA da senha, só que ela existe: quem já tem uma
// senha de 200 caracteres gravada precisa conseguir entrar com ela, e recusar no
// login o que o registro aceitou tranca a conta em vez de proteger.
func ValidateLogin(b LoginBody) plataforma.FieldErrorMap {
	f := plataforma.FieldErrorMap{}
	if !IsEmail(b.Email) {
		f["email"] = []string{msgEmailInvalido}
	}
	if b.Password == "" {
		f["password"] = []string{msgSenhaVazia}
	}
	return f
}
