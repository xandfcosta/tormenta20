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
// precisam distinguir "não achei" de "deu erro". Para a recusa de ENTRAR o que
// atravessa são os sentinelas do CASO DE USO (`campaign.ErrNoSuchCampaign` e
// irmãos), que esta cena lê direto — o `app/` está abaixo dela (ALE-348).
var errNoSuchCampaign = errors.New("campanha não existe")

// trimOrNull traduz texto vazio em NULL, e é uma CÓPIA declarada de sete linhas
// que o `api` também tem.
//
// Pô-la na porta seria mais acoplamento que duplicação — é a mesma decisão do
// `stepFromURL` da forja, e pelo mesmo motivo: a alternativa é um método na
// interface para converter uma string.
//
// Ela mora aqui e não no `campaign` porque o pacote de REGRA não pode carregar
// `database/sql`: a regra devolve texto, quem grava traduz. Esta cena grava.
func trimOrNull(text string) sql.NullString {
	if t := strings.TrimSpace(text); t != "" {
		return sql.NullString{String: t, Valid: true}
	}
	return sql.NullString{}
}
