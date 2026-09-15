package creature

import (
	"fmt"
	"strings"
)

// Block é o bloco de criatura do livro, que é como o Tormenta 20
// descreve TANTO monstro quanto NPC humano — "BANDIDO — ND 1/4 — Humanoide
// (humano) Médio" (p289) tem a mesma forma do Ogro (p293).
//
// Os campos são as linhas rotuladas do bloco impresso, nesta ordem: ND e tipo,
// Iniciativa e Percepção, Defesa e as três resistências, Pontos de Vida,
// Pontos de Mana (só conjurador — o Centauro Xamã tem 20, p290), Deslocamento,
// ataques, os seis atributos, Perícias, Equipamento e Tesouro. O que sobra do
// bloco é prosa nomeada (as habilidades especiais), e fica como texto porque é
// texto no livro.
//
// O que NÃO está aqui, de propósito: PV e PM ATUAIS. Eles são estado de
// combate e vivem na linha da iniciativa — o bloco é a descrição da criatura,
// e um vilão recorrente não guarda o dano da semana passada.
type Block struct {
	ND         float64 `json:"nd"`
	Tipo       string  `json:"tipo"`
	Size       string  `json:"size"`
	Iniciativa int     `json:"iniciativa"`
	Percepcao  int     `json:"percepcao"`
	Defesa     int     `json:"defesa"`
	Fortitude  int     `json:"fortitude"`
	Reflexos   int     `json:"reflexos"`
	Vontade    int     `json:"vontade"`
	HP         int     `json:"hp"`
	// PM é ponteiro porque a maioria das criaturas não tem a linha: o Bandido
	// não conjura, e um zero ali diria "tem mana e está sem", que é outra coisa.
	PM               *int     `json:"pm,omitempty"`
	Deslocamento     string   `json:"deslocamento"`
	Forca            int      `json:"forca"`
	Destreza         int      `json:"destreza"`
	Constituicao     int      `json:"constituicao"`
	Inteligencia     int      `json:"inteligencia"`
	Sabedoria        int      `json:"sabedoria"`
	Carisma          int      `json:"carisma"`
	Attacks          []Attack `json:"attacks"`
	Skills           []Skill  `json:"skills"`
	Equipment        string   `json:"equipment"`
	Treasure         string   `json:"treasure"`
	SpecialAbilities []string `json:"specialAbilities"`
	// SourceMonsterID diz de qual verbete este bloco foi copiado, quando foi.
	// Serve para a tela dizer "veio do Ogro (p293)" e para uma futura
	// comparação com o catálogo; vazio quando o mestre escreveu do zero.
	SourceMonsterID string `json:"sourceMonsterId,omitempty"`
}

// Attack é uma linha de ataque: "Corpo a Corpo Clava +7 (1d6+3)".
type Attack struct {
	Name        string `json:"name"`
	AttackBonus int    `json:"attackBonus"`
	Damage      string `json:"damage"`
	// Ranged separa "Corpo a Corpo" de "À Distância", que no livro são duas
	// linhas diferentes do mesmo bloco.
	Ranged bool `json:"ranged,omitempty"`
	// Special é a nota entre parênteses da linha de ataque ("mais agarrar"),
	// que o livro escreve em prosa.
	//
	// SEM `omitempty`, ao contrário do `Ranged` logo acima, e a diferença é do
	// EDITOR: o campo é ligado a uma caixa de texto por `data-bind`, e um sinal
	// AUSENTE chega ao navegador como `undefined` — a caixa nasceria com a
	// palavra "undefined" escrita dentro, e salvar a guardaria como o efeito do
	// ataque. Booleano ausente vira `false`, que é o valor certo; texto ausente
	// vira uma palavra.
	Special string `json:"special"`
}

// Skill é uma perícia da linha "Perícias Furtividade +5" (p289).
type Skill struct {
	Name  string `json:"name"`
	Bonus int    `json:"bonus"`
	// Nota é o bônus CONDICIONAL que o livro escreve entre parênteses depois do
	// número — "Furtividade +4 (+14 em pântanos)" na Hidra (p306). Irmã do
	// `Special` do ataque, e pela mesma razão: é prosa que o livro grudou numa
	// linha estruturada.
	//
	// Cuidado com o outro parêntese, que quer o oposto: `Ofício (armeiro) +2`
	// vem ANTES do número e faz parte do NOME da perícia (ALE-151).
	//
	// SEM `omitempty` pela mesma razão do `Special` do ataque: ela é uma caixa de
	// texto do editor, e ausente ela chegaria escrita "undefined".
	Nota string `json:"nota"`
}

// creatureTipos são os tipos de criatura do livro. Fechado porque o livro os
// fecha: o tipo decide efeitos que miram categoria (morto-vivo, construto).
var creatureTipos = map[string]bool{
	"humanoide":  true,
	"animal":     true,
	"monstro":    true,
	"morto-vivo": true,
	"construto":  true,
	"espirito":   true,
	"planar":     true,
}

// creatureSizes são os tamanhos do livro, e o tabuleiro lê este campo para
// saber quantos quadrados a peça ocupa (ALE-124).
var creatureSizes = map[string]bool{
	"minusculo": true,
	"pequeno":   true,
	"medio":     true,
	"grande":    true,
	"enorme":    true,
	"colossal":  true,
}

const creatureMaxAttacks = 12

// Validate recusa um bloco que a tela não saberia mostrar. As mensagens
// dizem o valor recebido e a forma esperada, como manda o guia da raiz: quem lê
// o erro está com o formulário aberto e precisa saber o que corrigir.
//
// EM PORTUGUÊS desde a ALE-269, e não é cosmética: elas eram inglesas enquanto o
// formulário sempre foi português, e o editor de bloco é a primeira tela em que
// a recusa é COMUM — salvar sem nome e salvar com PV zero são o caminho normal
// de quem está inventando um NPC. "hp is 0, must be >= 1" ao lado de uma caixa
// escrita "Pontos de Vida" faz o mestre procurar um campo que não existe.
func Validate(name string, b *Block) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("o NPC precisa de um nome")
	}
	if len(name) > 60 {
		return fmt.Errorf("o nome tem %d caracteres, e o máximo é 60", len(name))
	}
	if b.ND < 0 {
		return fmt.Errorf("o ND é %v, e precisa ser 0 ou mais", b.ND)
	}
	if !creatureTipos[b.Tipo] {
		return fmt.Errorf("o tipo %q não é um dos tipos de criatura do livro", b.Tipo)
	}
	if !creatureSizes[b.Size] {
		return fmt.Errorf("o tamanho %q não é um dos tamanhos do livro", b.Size)
	}
	if b.HP < 1 {
		return fmt.Errorf("os PV são %d, e precisam ser 1 ou mais", b.HP)
	}
	if b.PM != nil && *b.PM < 0 {
		return fmt.Errorf("os PM são %d, e precisam ser 0 ou mais em quem conjura", *b.PM)
	}
	if len(b.Attacks) > creatureMaxAttacks {
		return fmt.Errorf("o NPC tem %d ataques, e o máximo é %d", len(b.Attacks), creatureMaxAttacks)
	}
	for i, a := range b.Attacks {
		if strings.TrimSpace(a.Name) == "" {
			return fmt.Errorf("o ataque %d está sem nome", i+1)
		}
	}
	for i, s := range b.Skills {
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("a perícia %d está sem nome", i+1)
		}
	}
	return nil
}

// Normalize preenche as listas vazias para o JSON sair com `[]` e não
// `null` — o cliente itera sobre elas sem checar, e `null` viraria erro de
// runtime na primeira criatura sem ataque.
func Normalize(b *Block) {
	if b.Attacks == nil {
		b.Attacks = []Attack{}
	}
	if b.Skills == nil {
		b.Skills = []Skill{}
	}
	if b.SpecialAbilities == nil {
		b.SpecialAbilities = []string{}
	}
}
