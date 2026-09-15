// Package account é o que uma CONTA aceita: o que é um e-mail, o que é uma
// senha, e a forma dos dois pedidos que criam sessão.
//
// É um pacote e não parte da porta porque a regra é lida pela cena da porta E
// pela API JSON, e depende só do `platform`. O destino de uma função é a
// DEPENDÊNCIA dela — pô-la em `web/door` faria a API JSON importar um pacote de
// CENA para validar, que é o contrário da direção que a divisão existe para
// criar. É a mesma forma do `search`, e pelo mesmo motivo declarado lá: função
// pura hospedada num pacote grande faz quem não pode importar aquele pacote
// escrever uma cópia, e a cópia sai errada de um jeito que compila.
//
// E NÃO vai para `platform`: aquele pacote é infraestrutura sem domínio, e "a
// senha precisa ter ao menos 8 caracteres" é regra de PRODUTO. Quem a mudar está
// mudando o que o jogador pode fazer, não como o servidor escreve JSON.
package account

import (
	"regexp"
	"unicode/utf8"

	"t20engine/infra/platform"
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
	// InviteToken é o link de uso único que o admin entregou ao jogador.
	// Obrigatório para todo mundo menos os endereços do ADMIN_EMAILS.
	InviteToken string `json:"inviteToken"`
}

// AS MENSAGENS SÃO AS QUE O JOGADOR LÊ, então são em pt-BR — e são UMAS SÓ, para
// as duas portas. A cena as mostra na cara de quem digitou, e a rota JSON
// responde as mesmas.
const (
	msgEmailInvalido = "E-mail inválido"
	msgSenhaCurta    = "A senha precisa ter ao menos 8 caracteres"
	msgSenhaLonga    = "A senha pode ter no máximo 128 caracteres"
	msgSenhaVazia    = "Informe sua senha"
	msgNomeLongo     = "O nome pode ter no máximo 80 caracteres"
)

// Um teste pragmático de FORMA de e-mail. Um validador estrito não vale a
// superfície: quem digita errado descobre no convite que não chega.
var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func IsEmail(s string) bool { return emailRe.MatchString(s) }

// ValidateRegister recusa o que não vira conta: e-mail sem forma de e-mail,
// senha fora da faixa, nome longo demais.
//
//	if fields := account.ValidateRegister(body); len(fields) > 0 { … }
func ValidateRegister(b RegisterBody) platform.FieldErrorMap {
	f := platform.FieldErrorMap{}
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
// link de redefinição — duas grafias de "ao menos 8" divergem, e a tela que
// ficar mais frouxa é a que importa.
func ValidatePassword(password string) platform.FieldErrorMap {
	f := platform.FieldErrorMap{}
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
func ValidateLogin(b LoginBody) platform.FieldErrorMap {
	f := platform.FieldErrorMap{}
	if !IsEmail(b.Email) {
		f["email"] = []string{msgEmailInvalido}
	}
	if b.Password == "" {
		f["password"] = []string{msgSenhaVazia}
	}
	return f
}
