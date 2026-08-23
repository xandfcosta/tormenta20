package engine

// As regras que o livro deixa NA MÃO DO MESTRE (ALE-221).
//
// O T20 não manda aplicar tudo. Sobre os limites de carga ele diz, com todas as
// letras: "O mestre pode ignorar essa regra, desde que os jogadores não abusem"
// (p141). Um app que aplica sempre escolhe um estilo de jogo pelo mestre — e
// escolhe o mais punitivo. O interruptor entra no motor como ENTRADA, porque a
// autoridade das regras é daqui (ALE-104): se a tela decidisse sozinha se pinta
// a penalidade, voltariam as duas implementações da mesma regra.

// IgnoredRules são as regras que a campanha DESLIGOU.
//
// O campo nomeia o que está desligado, e não o que está ligado, de propósito: o
// valor zero desta struct significa "tudo em vigor", que é o padrão do livro e o
// padrão de uma campanha nova. Um `Character{}` literal — num teste, no
// oráculo, num fixture — calcula com todas as regras aplicadas sem ninguém
// lembrar de preencher nada, e uma regra nova nasce valendo. Um mapa das
// LIGADAS teria o padrão errado por construção, e o defeito apareceria como
// número faltando numa ficha.
//
// @example engine.Character{IgnoredRules: engine.IgnoredRules{Carga: true}} // mesa sem carga
type IgnoredRules struct {
	// Carga desliga os limites de carga da p141 — a mochila continua contando
	// espaços, mas ninguém fica sobrecarregado e nenhuma penalidade sai daqui.
	Carga bool `json:"carga"`
}

// RuleCarga é o identificador de fio da regra de carga: o que a campanha grava
// em `campaign_ignored_rules.rule` e o que a tela manda no PATCH. Escrito uma
// vez para o banco, o handler e o motor não divergirem por erro de digitação.
const RuleCarga = "carga"

// KnownRules são os identificadores que ESTA versão do motor reconhece. Serve à
// validação da escrita: ler um desconhecido é seguro (a regra fica em vigor),
// mas GRAVAR um desconhecido encheria a tabela de lixo que ninguém consegue
// desfazer pela tela.
var KnownRules = []string{RuleCarga}

// IsKnownRule diz se o identificador é uma regra desta versão do motor.
//
// @example engine.IsKnownRule("carga") // true
func IsKnownRule(rule string) bool {
	for _, known := range KnownRules {
		if known == rule {
			return true
		}
	}
	return false
}

// IgnoredRulesFrom monta a struct a partir dos identificadores gravados.
// Identificador desconhecido é IGNORADO em silêncio, e é a escolha certa aqui:
// a lista mora no banco e sobrevive a um rollback do binário, então uma regra
// que o Go desta versão não conhece não pode derrubar o cálculo da ficha — ela
// simplesmente continua em vigor, que é o lado seguro.
func IgnoredRulesFrom(rules []string) IgnoredRules {
	var out IgnoredRules
	for _, rule := range rules {
		if rule == RuleCarga {
			out.Carga = true
		}
	}
	return out
}
