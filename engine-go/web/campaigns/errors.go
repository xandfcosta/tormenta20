package campaigns

import (
	"database/sql"
	"errors"
	"strings"
)

// errNoSuchCampaign é o sinal INTERNO desta cena para "esse id não é de
// campanha nenhuma".
//
// Ele existe porque o `LoadOne` responde 404 e o `LoadList` não, então os dois
// precisam distinguir "não achei" de "deu erro". Não é o sentinela do
// hospedeiro: aquele é valor do `api` e não atravessa a fronteira — para a
// recusa de ENTRAR, o que atravessa é o `JoinRefusal`.
var errNoSuchCampaign = errors.New("campanha não existe")

// trimOrNull traduz texto vazio em NULL, e é uma CÓPIA declarada de sete linhas
// que o `api` também tem.
//
// Pô-la na porta seria mais acoplamento que duplicação — é a mesma decisão do
// `oPassoDaURL` da forja, e pelo mesmo motivo: a alternativa é um método na
// interface para converter uma string.
//
// Ela mora aqui e não no `campaign` porque o pacote de REGRA não pode carregar
// `database/sql`: a regra devolve texto, quem grava traduz. Esta cena grava.
func trimOrNull(texto string) sql.NullString {
	if t := strings.TrimSpace(texto); t != "" {
		return sql.NullString{String: t, Valid: true}
	}
	return sql.NullString{}
}
