package campaign

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"t20engine/domain/engine"
	"t20engine/infra/wire"
)

// As regras de uma CAMPANHA: o que é um nome válido e o que é uma descrição
// válida.
//
// Elas são pacote e não ficam soldadas ao transporte porque as DUAS portas — a
// cena e a rota JSON — precisam delas. Regra duplicada é regra que diverge.
//
// E não vão para `platform`, pela mesma razão escrita no `account`: "o nome cabe
// em 120 caracteres" é regra de PRODUTO, e o `platform` é infraestrutura sem
// domínio. Um conceito do jogo lá dentro é a fronteira no lugar errado.

const (
	// MaxNameLength e MaxDescriptionLength são exportados porque a CENA os
	// escreve na tela — o contador de caracteres do formulário precisa do mesmo
	// número que a recusa usa, senão o campo diz 2000 e o servidor recusa em
	// 1500 sem que nada explique.
	MaxNameLength        = 120
	MaxDescriptionLength = 2000
)

// AS MENSAGENS SÃO AS QUE O MESTRE LÊ, então são em pt-BR e moram AQUI — uma
// por regra. Duas frases para a mesma regra (uma na cena, outra na rota JSON)
// deixam uma delas para trás no dia em que alguém mudar o limite.
const (
	msgNomeInvalido   = "O nome é obrigatório e cabe em 120 caracteres"
	msgDescricaoLonga = "A descrição cabe em 2000 caracteres"
)

// Name apara e valida.
//
// O apara vem ANTES da medida, senão um nome de puros espaços passa no `!= ""`
// e a campanha nasce sem título no livro.
//
//	nome, erros := campaign.Name(bruto)
//	if len(erros) > 0 { … }
func Name(raw string) (string, wire.FieldErrorMap) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > MaxNameLength {
		return "", wire.FieldErrorMap{"name": {msgNomeInvalido}}
	}
	return name, nil
}

// Description apara e valida, devolvendo string VAZIA para ausente.
//
// Ela NÃO devolve `sql.NullString`: isso faria a regra de produto carregar
// `database/sql`. Quem grava converte — vazio é NULL nos dois caminhos (criar e
// editar), senão o cliente lê `""` de um e `null` do outro para exatamente a
// mesma entrada.
//
// A medida é em RUNAS e não em bytes: "Coração" tem 7 caracteres para quem
// escreve e 8 bytes para quem conta errado, e um limite que encolhe conforme os
// acentos é um limite que mente.
func Description(raw *string) (string, wire.FieldErrorMap) {
	if raw == nil {
		return "", nil
	}
	if utf8.RuneCountInString(*raw) > MaxDescriptionLength {
		return "", wire.FieldErrorMap{"description": {msgDescricaoLonga}}
	}
	return strings.TrimSpace(*raw), nil
}

// ValidateText é o PAR, para quem recebe os dois campos no mesmo formulário.
//
// Ela existe porque os dois chamadores — a cena e a rota JSON — precisam das
// duas recusas JUNTAS: parar no primeiro erro faria o mestre corrigir o nome,
// reenviar, e só então descobrir que a descrição também estava longa. Um
// formulário que devolve um erro por vez é um formulário que se preenche duas
// vezes.
func ValidateText(rawName string, rawDescription *string) (string, string, wire.FieldErrorMap) {
	errs := wire.FieldErrorMap{}
	name, nameErr := Name(rawName)
	for field, sentences := range nameErr {
		errs[field] = sentences
	}
	description, descErr := Description(rawDescription)
	for field, sentences := range descErr {
		errs[field] = sentences
	}
	if len(errs) == 0 {
		return name, description, nil
	}
	return name, description, errs
}

// AS REGRAS OPCIONAIS: o que o mestre DESLIGOU na campanha.
//
// O nome do campo diz o que está DESLIGADO e isso é proposital: valor zero
// significa "tudo em vigor", que é o padrão do livro. Ver o GLOSSARY, verbete
// **regra opcional**.

// NormalizeIgnoredRules ordena, tira repetidos e recusa o que o motor não
// conhece.
//
// A recusa NOMEIA o valor ofensor e a lista esperada, que é a regra da casa para
// mensagem de erro — "regra inválida" mandaria o mestre adivinhar qual das
// dezenas ele digitou errado.
//
// A frase CHEGA NA TELA: a cena a manda para o navegador no sinal `erroDaRegra`,
// então ela é em português pelo mesmo motivo que as duas mensagens acima.
func NormalizeIgnoredRules(raw []string) ([]string, string) {
	seen := map[string]bool{}
	outside := []string{}
	for _, rule := range raw {
		if !engine.IsKnownRule(rule) {
			return nil, fmt.Sprintf("regra desconhecida %q — esperava uma de %v", rule, engine.KnownRules)
		}
		if seen[rule] {
			continue
		}
		seen[rule] = true
		outside = append(outside, rule)
	}
	sort.Strings(outside)
	return outside, ""
}
