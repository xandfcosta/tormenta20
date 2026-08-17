package api

import (
	"fmt"
	"strings"
)

// CreatureBlock é o bloco de criatura do livro, que é como o Tormenta 20
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
type CreatureBlock struct {
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
	PM               *int             `json:"pm,omitempty"`
	Deslocamento     string           `json:"deslocamento"`
	Forca            int              `json:"forca"`
	Destreza         int              `json:"destreza"`
	Constituicao     int              `json:"constituicao"`
	Inteligencia     int              `json:"inteligencia"`
	Sabedoria        int              `json:"sabedoria"`
	Carisma          int              `json:"carisma"`
	Attacks          []CreatureAttack `json:"attacks"`
	Skills           []CreatureSkill  `json:"skills"`
	Equipment        string           `json:"equipment"`
	Treasure         string           `json:"treasure"`
	SpecialAbilities []string         `json:"specialAbilities"`
	// SourceMonsterID diz de qual verbete este bloco foi copiado, quando foi.
	// Serve para a tela dizer "veio do Ogro (p293)" e para uma futura
	// comparação com o catálogo; vazio quando o mestre escreveu do zero.
	SourceMonsterID string `json:"sourceMonsterId,omitempty"`
}

// CreatureAttack é uma linha de ataque: "Corpo a Corpo Clava +7 (1d6+3)".
type CreatureAttack struct {
	Name        string `json:"name"`
	AttackBonus int    `json:"attackBonus"`
	Damage      string `json:"damage"`
	// Ranged separa "Corpo a Corpo" de "À Distância", que no livro são duas
	// linhas diferentes do mesmo bloco.
	Ranged bool `json:"ranged,omitempty"`
	// Special é a nota entre parênteses da linha de ataque ("mais agarrar"),
	// que o livro escreve em prosa.
	Special string `json:"special,omitempty"`
}

// CreatureSkill é uma perícia da linha "Perícias Furtividade +5" (p289).
type CreatureSkill struct {
	Name  string `json:"name"`
	Bonus int    `json:"bonus"`
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

// validateCreature recusa um bloco que a tela não saberia mostrar. As mensagens
// dizem o valor recebido e a forma esperada, como manda o guia da raiz: quem lê
// o erro está com o formulário aberto e precisa saber o que corrigir.
func validateCreature(name string, b *CreatureBlock) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("creature name is required")
	}
	if len(name) > 60 {
		return fmt.Errorf("creature name has %d chars, max is 60", len(name))
	}
	if b.ND < 0 {
		return fmt.Errorf("nd is %v, must be >= 0", b.ND)
	}
	if !creatureTipos[b.Tipo] {
		return fmt.Errorf("tipo %q is not one of the book's creature types", b.Tipo)
	}
	if !creatureSizes[b.Size] {
		return fmt.Errorf("size %q is not one of the book's sizes", b.Size)
	}
	if b.HP < 1 {
		return fmt.Errorf("hp is %d, must be >= 1", b.HP)
	}
	if b.PM != nil && *b.PM < 0 {
		return fmt.Errorf("pm is %d, must be >= 0 when present", *b.PM)
	}
	if len(b.Attacks) > creatureMaxAttacks {
		return fmt.Errorf("creature has %d attacks, max is %d", len(b.Attacks), creatureMaxAttacks)
	}
	for i, a := range b.Attacks {
		if strings.TrimSpace(a.Name) == "" {
			return fmt.Errorf("attack %d has no name", i+1)
		}
	}
	for i, s := range b.Skills {
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("skill %d has no name", i+1)
		}
	}
	return nil
}

// normalizeCreature preenche as listas vazias para o JSON sair com `[]` e não
// `null` — o cliente itera sobre elas sem checar, e `null` viraria erro de
// runtime na primeira criatura sem ataque.
func normalizeCreature(b *CreatureBlock) {
	if b.Attacks == nil {
		b.Attacks = []CreatureAttack{}
	}
	if b.Skills == nil {
		b.Skills = []CreatureSkill{}
	}
	if b.SpecialAbilities == nil {
		b.SpecialAbilities = []string{}
	}
}
