package api

import (
	"t20engine/plataforma"
	"unicode/utf8"
)

// AS VALIDAÇÕES DA PORTA — o que sobrou aqui depois de o `api/` virar quatro
// pacotes (ALE-254).
//
// O resto deste arquivo virou `plataforma/validate.go`: `DecodeJSON`, `NowISO`,
// `NormalizeEmail`, `WriteValidationError` e companhia são infraestrutura, e
// agora se chamam pelo pacote. O que fica aqui são as quatro funções da ALE-229,
// que NÃO existem na main — elas nasceram com a porta em Datastar e o merge da
// nona puxada quase as perdeu, porque mover arquivo entre pacotes faz o git ver
// apagar+criar, e o lado que "vence" é o que não tinha a mudança.
//
// Ficam em `api/` e não em `plataforma/` de propósito: `plataforma` é
// infraestrutura sem domínio, e "a senha precisa ter ao menos 8 caracteres" é
// regra de PRODUTO — quem a mudar está mudando o que o jogador pode fazer, não
// como o servidor escreve JSON.

// O `emailRe` e o `IsEmail` vivem no `account_rules.go` desde a nona puxada —
// a main os extraiu para lá. Duas cópias do mesmo regex é exatamente o defeito
// que uma extração existe para evitar, então aqui só se usa o de lá.

// As mensagens da PORTA são as que o jogador lê, então elas são em pt-BR
// (ALE-229).
//
// Eram em inglês — herança das frases do class-validator do NestJS —, e a SPA
// escondia isso validando com Zod antes de chamar. Servidor-renderizado elas
// chegam na cara de quem digitou, e o buraco já existia: uma senha de 200
// caracteres passa pelo Zod (que só checa o mínimo) e volta "password must be
// shorter than or equal to 128 characters".
//
// O texto é o MESMO do Zod da SPA de propósito: enquanto as duas portas
// existirem, duas redações da mesma recusa seriam duas telas discordando.
const (
	msgEmailInvalido = "E-mail inválido"
	msgSenhaCurta    = "A senha precisa ter ao menos 8 caracteres"
	msgSenhaLonga    = "A senha pode ter no máximo 128 caracteres"
	msgSenhaVazia    = "Informe sua senha"
	msgNomeLongo     = "O nome pode ter no máximo 80 caracteres"
)

func validateRegister(b registerBody) plataforma.FieldErrorMap {
	f := plataforma.FieldErrorMap{}
	if !IsEmail(b.Email) {
		f["email"] = []string{msgEmailInvalido}
	}
	for field, messages := range validatePassword(b.Password) {
		f[field] = messages
	}
	if b.Name != nil && utf8.RuneCountInString(*b.Name) > 80 {
		f["name"] = []string{msgNomeLongo}
	}
	return f
}

// validatePassword is the ONE password rule, shared by registration and by the
// reset link (ALE-120) — two spellings of "at least 8" would drift, and the
// screen that ended up laxer would be the one that matters.
func validatePassword(password string) plataforma.FieldErrorMap {
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

func validateLogin(b loginBody) plataforma.FieldErrorMap {
	f := plataforma.FieldErrorMap{}
	if !IsEmail(b.Email) {
		f["email"] = []string{msgEmailInvalido}
	}
	if b.Password == "" {
		f["password"] = []string{msgSenhaVazia}
	}
	return f
}

// plataforma.NowISO is the timestamp format stored in the TEXT DateTime columns (ISO-8601,
// UTC, millisecond precision — what Prisma serialized).
// plataforma.IsoLayout is the one spelling of a timestamp in this database. Shared so a
// stored value parses back with the layout that wrote it (ALE-120).
