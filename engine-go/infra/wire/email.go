package wire

import "strings"

// NormalizeEmail é a grafia ÚNICA de uma conta: criar conta, entrar e a lista de
// administradores passam os três por ela, então `Mestre@T20.local` e
// `mestre@t20.local` são UMA conta e não duas.
//
// É isso que deixa a checagem de administrador ignorar caixa sem abrir porta:
// uma variante de caixa não pode virar um segundo administrador.
//
// Ela mora AQUI e não no `domain/account`, que seria o lugar pelo assunto,
// porque o `infra/config` também precisa dela — e infraestrutura não importa
// domínio. Duas normalizações é exatamente o que a frase acima proíbe, então a
// única implementação tem de ficar abaixo das duas.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
