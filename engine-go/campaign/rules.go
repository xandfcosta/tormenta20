package campaign

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"t20engine/engine"
	"t20engine/plataforma"
)

// As regras de uma CAMPANHA: o que é um nome válido e o que é uma descrição
// válida (ALE-246, extraídas do `api` na ALE-278).
//
// Elas viviam soldadas ao transporte, e em DOIS lugares: o `handleCreateCampaign`
// e o `handleUpdateCampaign` repetiam o mesmo `if` e a mesma frase, palavra por
// palavra. Regra duplicada é regra que diverge — basta alguém mexer num dos dois.
//
// E ao juntá-las apareceu uma DIVERGÊNCIA que já existia: o `campaign-schema.ts`
// da SPA recusava descrição acima de 2000 caracteres e o servidor aceitava
// qualquer tamanho. A regra do texto morava só no cliente, e a virada para
// servidor-renderizado a teria apagado sem ninguém notar.
//
// > Aqui a prosa dizia "as regras de uma CRÔNICA". `crônica` é termo PROIBIDO
// > pelo GLOSSARIO desde 2026-08-22 — a palavra é `campanha` —, e o comentário
// > atravessou a decisão sem ninguém reler. É o defeito que a seção
// > "Documentação" descreve, acontecido no arquivo que define o conceito.
//
// # Por que pacote, e não `plataforma`
//
// Mesma razão escrita no `account`, e ela vale letra por letra: "o nome cabe em
// 120 caracteres" é regra de PRODUTO, e o `plataforma` é infraestrutura sem
// domínio. Um conceito do jogo lá dentro é a fronteira no lugar errado.

const (
	// MaxNameLength e MaxDescriptionLength são exportados porque a CENA os
	// escreve na tela — o contador de caracteres do formulário precisa do mesmo
	// número que a recusa usa, senão o campo diz 2000 e o servidor recusa em
	// 1500 sem que nada explique.
	MaxNameLength        = 120
	MaxDescriptionLength = 2000
)

// AS MENSAGENS SÃO AS QUE O MESTRE LÊ, então são em pt-BR e moram AQUI.
//
// Antes eram duas para cada regra: a cena escrevia a dela em português e a rota
// JSON respondia `err.Error()`, que era uma frase em inglês herdada do NestJS
// ("name must be between 1 and 120 characters"). Duas frases para uma regra é a
// forma que o `account` desfez nesta mesma épica, e pelo mesmo motivo: quando
// alguém mudar o limite, uma das duas fica para trás.
//
// Nada media a frase inglesa — nenhum teste a citava —, e é por isso que ela
// pôde sair junto com a extração em vez de virar issue própria.
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
func Name(bruto string) (string, plataforma.FieldErrorMap) {
	nome := strings.TrimSpace(bruto)
	if nome == "" || utf8.RuneCountInString(nome) > MaxNameLength {
		return "", plataforma.FieldErrorMap{"name": {msgNomeInvalido}}
	}
	return nome, nil
}

// Description apara e valida, devolvendo string VAZIA para ausente.
//
// Ela não devolve `sql.NullString`, e isso é a extração corrigindo um vazamento:
// a versão anterior devolvia o tipo do banco, o que fazia a regra de produto
// carregar `database/sql`. Quem grava converte — vazio é NULL nos dois caminhos
// (criar e editar), senão o cliente lê `""` de um e `null` do outro para
// exatamente a mesma entrada.
//
// A medida é em RUNAS e não em bytes: "Coração" tem 7 caracteres para quem
// escreve e 8 bytes para quem conta errado, e um limite que encolhe conforme os
// acentos é um limite que mente.
func Description(bruto *string) (string, plataforma.FieldErrorMap) {
	if bruto == nil {
		return "", nil
	}
	if utf8.RuneCountInString(*bruto) > MaxDescriptionLength {
		return "", plataforma.FieldErrorMap{"description": {msgDescricaoLonga}}
	}
	return strings.TrimSpace(*bruto), nil
}

// ValidateText é o PAR, para quem recebe os dois campos no mesmo formulário.
//
// Ela existe porque os dois chamadores — a cena e a rota JSON — precisam das
// duas recusas JUNTAS: parar no primeiro erro faria o mestre corrigir o nome,
// reenviar, e só então descobrir que a descrição também estava longa. Um
// formulário que devolve um erro por vez é um formulário que se preenche duas
// vezes.
func ValidateText(nomeBruto string, descricaoBruta *string) (string, string, plataforma.FieldErrorMap) {
	erros := plataforma.FieldErrorMap{}
	nome, errNome := Name(nomeBruto)
	for campo, frases := range errNome {
		erros[campo] = frases
	}
	descricao, errDesc := Description(descricaoBruta)
	for campo, frases := range errDesc {
		erros[campo] = frases
	}
	if len(erros) == 0 {
		return nome, descricao, nil
	}
	return nome, descricao, erros
}

// AS REGRAS OPCIONAIS: o que o mestre DESLIGOU na campanha (ALE-221).
//
// O nome do campo diz o que está DESLIGADO e isso é proposital: valor zero
// significa "tudo em vigor", que é o padrão do livro. Ver o GLOSSARIO, verbete
// **regra opcional**.

// NormalizeIgnoredRules ordena, tira repetidos e recusa o que o motor não
// conhece.
//
// A recusa NOMEIA o valor ofensor e a lista esperada, que é a regra da casa para
// mensagem de erro — "regra inválida" mandaria o mestre adivinhar qual das
// dezenas ele digitou errado.
//
// **A frase era em INGLÊS** ("unknown rule %q — expected one of %v") e ela chega
// na tela: a cena a manda para o navegador no sinal `erroDaRegra`. Traduzida na
// extração, pelo mesmo motivo que as duas mensagens acima — quem lê é o mestre.
func NormalizeIgnoredRules(brutas []string) ([]string, string) {
	vistas := map[string]bool{}
	fora := []string{}
	for _, regra := range brutas {
		if !engine.IsKnownRule(regra) {
			return nil, fmt.Sprintf("regra desconhecida %q — esperava uma de %v", regra, engine.KnownRules)
		}
		if vistas[regra] {
			continue
		}
		vistas[regra] = true
		fora = append(fora, regra)
	}
	sort.Strings(fora)
	return fora, ""
}
