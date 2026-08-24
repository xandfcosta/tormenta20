package api

// AS REGRAS DE CONTA: o que é um e-mail e o que é uma senha aceitável.
//
// Estavam no `validate.go` e vieram para cá quando o `plataforma/` nasceu
// (ALE-254) — e não por gosto: o COMPILADOR apontou. `ValidateRegister` lê
// `registerBody` e `ValidateLogin` lê `loginBody`, tipos de conta, e um pacote
// que não é domínio nenhum não pode conhecê-los.
//
// Foi o ciclo funcionando como a issue previu: ele disse que a fronteira estava
// no lugar errado, e a resposta certa foi tabuleiro.Mover a regra, não afrouxar o pacote.
// Em `plataforma/` ficou só a MAQUINARIA — o mapa de erro por campo e como ele
// chega ao cliente.

import (
	"regexp"
	"unicode/utf8"

	"t20engine/plataforma"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func IsEmail(s string) bool { return emailRe.MatchString(s) }

func ValidateRegister(b registerBody) plataforma.FieldErrorMap {
	f := plataforma.FieldErrorMap{}
	if !IsEmail(b.Email) {
		f["email"] = []string{"email must be an email"}
	}
	for field, messages := range ValidatePassword(b.Password) {
		f[field] = messages
	}
	if b.Name != nil && utf8.RuneCountInString(*b.Name) > 80 {
		f["name"] = []string{"name must be shorter than or equal to 80 characters"}
	}
	return f
}

// ValidatePassword is the ONE password rule, shared by registration and by the
// Reset link (ALE-120) — two spellings of "at least 8" would drift, and the
// screen that ended up laxer would be the one that matters.
func ValidatePassword(password string) plataforma.FieldErrorMap {
	f := plataforma.FieldErrorMap{}
	length := utf8.RuneCountInString(password)
	if length < 8 {
		f["password"] = append(f["password"], "Password must be at least 8 characters")
	}
	if length > 128 {
		f["password"] = append(f["password"], "password must be shorter than or equal to 128 characters")
	}
	return f
}

func ValidateLogin(b loginBody) plataforma.FieldErrorMap {
	f := plataforma.FieldErrorMap{}
	if !IsEmail(b.Email) {
		f["email"] = []string{"email must be an email"}
	}
	if b.Password == "" {
		f["password"] = []string{"password must be longer than or equal to 1 characters"}
	}
	return f
}

// NowISO is the timestamp format stored in the TEXT DateTime columns (ISO-8601,
// UTC, millisecond precision — what Prisma serialized).
// plataforma.IsoLayout is the one spelling of a timestamp in this database. Shared so a
// stored value parses back with the layout that wrote it (ALE-120).
