package api

import (
	"database/sql"
	"errors"
	"strings"
)

// As regras de uma CRÔNICA: o que é um nome válido e o que é uma descrição
// válida (ALE-246).
//
// Elas viviam soldadas ao transporte, e desta vez em DOIS lugares: o
// `handleCreateCampaign` e o `handleUpdateCampaign` repetiam o mesmo `if` e a
// mesma frase de erro, palavra por palavra. Regra duplicada é regra que diverge
// — basta alguém mexer num dos dois. É a oitava vez que a migração encontra
// este padrão.
//
// E ao juntá-las apareceu uma DIVERGÊNCIA que já existia: o `campaign-schema.ts`
// da SPA recusa descrição acima de 2000 caracteres e o servidor aceitava
// qualquer tamanho. Ou seja, a regra do texto morava só no cliente, e a virada
// desta fatia a teria apagado sem ninguém notar — a tela nova é do servidor, e
// o servidor não a conhecia. Agora ele conhece.
//
// Isso APERTA a rota JSON, e é deliberado: nenhum cliente legítimo mandava mais
// que 2000, porque o formulário da SPA já recusava. O que muda é que passar por
// fora do formulário deixou de funcionar.

const (
	nomeDeCampanhaMax      = 120
	descricaoDeCampanhaMax = 2000
)

var (
	errNomeDeCampanha      = errors.New("name must be between 1 and 120 characters")
	errDescricaoDeCampanha = errors.New("description must be at most 2000 characters")
)

// nomeDeCampanha apara e valida. O apara vem ANTES da medida, senão um nome de
// puros espaços passa no `!= ""` e a crônica nasce sem título no livro.
func nomeDeCampanha(bruto string) (string, error) {
	nome := strings.TrimSpace(bruto)
	if nome == "" || len([]rune(nome)) > nomeDeCampanhaMax {
		return "", errNomeDeCampanha
	}
	return nome, nil
}

// descricaoDeCampanha apara e valida, devolvendo NULL para texto vazio.
//
// Vazio é NULL e não string vazia nos dois caminhos (criar e editar), senão o
// cliente lê `""` de um e `null` do outro para exatamente a mesma entrada.
//
// A medida é em RUNAS e não em bytes: "Coração" tem 7 caracteres para quem
// escreve e 8 bytes para quem conta errado, e um limite que encolhe conforme os
// acentos é um limite que mente.
func descricaoDeCampanha(bruto *string) (sql.NullString, error) {
	if bruto == nil {
		return sql.NullString{}, nil
	}
	if len([]rune(*bruto)) > descricaoDeCampanhaMax {
		return sql.NullString{}, errDescricaoDeCampanha
	}
	return trimOrNull(bruto), nil
}
