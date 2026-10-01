package engine

import "t20engine/domain/ecs"

// A RESOLUÇÃO DE UM ATAQUE (T20 p230-231).
//
// É a primeira regra do capítulo 5 a existir em código: até aqui o motor era
// uma calculadora de ficha — ele dizia o BÔNUS de ataque e a FÓRMULA de dano, e
// quem resolvia era uma pessoa rolando dado na mesa e digitando o PV novo.
//
// # O que entra pronto, e por quê
//
// O `d20` chega por parâmetro e os dados de dano por uma função. A regra não
// sorteia nada: ela decide o que se faz COM a rolagem. Isso é o que deixa cada
// caso do livro virar um teste com o número escrito à mão — inclusive o
// crítico, que de outro jeito precisaria de mil execuções para aparecer.
//
// Quem sorteia é o `RollDie`, uma camada acima.
//
// A REGRA em si mora em `attack_ecs.go`, em sistemas sobre parcelas de dano.
// Aqui ficam o vocabulário da fronteira e a montagem do mundo.

// AttackTarget é o alvo, do ponto de vista da regra — e só isto: a Defesa que
// ele opõe, a redução que ele aplica e se ele é imune a crítico.
//
// Ele NÃO é uma ficha nem uma linha da fila. O alvo de um ataque pode ser um
// herói, um NPC do bestiário ou uma porta, e as três coisas têm Defesa; pedir a
// ficha aqui amarraria a regra de combate ao formato de personagem.
type AttackTarget struct {
	// Defense é a CD do teste de ataque (p230).
	Defense int
	// DamageReduction é a RD do alvo (p229). Uma só, já resolvida: qual RD vale
	// quando há várias fontes é decisão do `characterDamageReduction`, e ela não
	// muda por ataque.
	DamageReduction int
	// CritImmune: "certas criaturas são imunes a acertos críticos" (p231).
	CritImmune bool
}

// ExtraRoll é uma parcela extra já rolada.
type ExtraRoll struct {
	Dice  []int  `json:"dice"`
	Faces int    `json:"faces"`
	Type  string `json:"type"`
	Total int    `json:"total"`
}

// AttackOutcome é tudo que a mesa precisa ver, e não só o número final.
//
// A conta viaja inteira de propósito, pela mesma razão que o `ExpertiseBreakdown`
// existe: uma mesa que vê "8 de dano" e uma que vê "1d8 deu 8, a RD 5 comeu
// cinco, sobraram 3" são mesas diferentes — a segunda entende a regra sem
// perguntar, e a primeira desconfia do servidor.
type AttackOutcome struct {
	Roll     int  `json:"roll"`     // o d20 natural
	Total    int  `json:"total"`    // d20 + bônus de ataque
	Hit      bool `json:"hit"`      //
	Critical bool `json:"critical"` //
	// Situations são os RÓTULOS das linhas da Tabela 5-3 que valeram, na ordem
	// em que chegaram. Elas viajam porque o número sozinho DESMENTE o que a
	// mesa sabe: a Defesa não é a da ficha, e um ataque que bateu a Defesa pode
	// ter errado pela camuflagem (p238). Procedência, não enfeite.
	Situations []string `json:"situations,omitempty"`
	// Defense é a Defesa que ESTE ataque enfrentou, depois da Tabela 5-3 — ela
	// não é a da ficha quando o alvo está sob cobertura (p239). Viaja porque
	// "errei por 1" e "errei por 1 porque ele está atrás da carroça" são
	// leituras diferentes do mesmo número.
	Defense int `json:"defense"`
	// Dice são as rolagens de dano na ordem, já na quantidade que o crítico
	// pediu. Vazio quando o ataque erra: quem erra não rola dano.
	Dice []int `json:"dice"`
	// Faces é o dado da arma (o 8 de "1d8"). Ele viaja porque a mesa lê a
	// NOTAÇÃO — "2d8+3" diz de onde os números vieram, e "8+5+3" faz quem olha
	// reconstruir a arma de cabeça.
	Faces int `json:"faces"`
	// Extra são as parcelas de outro tipo — o 1d6 de fogo do encanto. Elas
	// entram no `RawDamage`, mas viajam DISCRIMINADAS: a mesa lê "1d8 deu 8,
	// mais 1d6 de fogo deu 4", e um total sozinho apagaria o tipo de dano.
	Extra     []ExtraRoll `json:"extra,omitempty"`
	RawDamage int         `json:"rawDamage"` // dados + bônus, antes da RD
	Absorbed  int         `json:"absorbed"`  // o que a RD comeu
	Damage    int         `json:"damage"`    // o que o alvo perde de PV
}

// ResolveAttack rola um ataque contra um alvo e devolve a conta inteira.
//
//	fora, err := ResolveAttack(carta, AttackTarget{Defense: 15}, 19, rolar)
//	// fora.Critical == true, fora.Damage == 16 para 1d8+3 com x2
func ResolveAttack(
	card WeaponCard, target AttackTarget, d20 int, rollDie func(faces int) (int, error),
) (AttackOutcome, error) {
	return ResolveAttackUnder(card, target, nil, d20, rollDie)
}

// ResolveAttackUnder é o ataque com as SITUAÇÕES ESPECIAIS da Tabela 5-3 (p239)
// que valem agora — a cobertura do alvo, a posição elevada do atacante, o
// flanqueio.
//
// Ela é a função de verdade e o `ResolveAttack` é o caso sem situação nenhuma,
// que continua existindo porque é o caso COMUM: uma mesa sem tabuleiro ataca
// assim, e os casos do livro são escritos assim.
//
// @example ResolveAttackUnder(carta, AttackTarget{Defense: 15},
//
//	[]SpecialSituation{TargetUnderLightCover}, 19, rolar)
func ResolveAttackUnder(
	card WeaponCard, target AttackTarget, situations []SpecialSituation,
	d20 int, rollDie func(faces int) (int, error),
) (AttackOutcome, error) {
	w := attackWorld(card, target, situations, d20, rollDie)
	out := outcomeFromWorld(w)
	if fault, has := ecs.Get[attackFault](w, theResource(w)); has {
		return out, fault.Err
	}
	return out, nil
}

// attackWorld monta o mundo e roda os sistemas. Separado do `ResolveAttack`
// porque os guardas precisam do MUNDO e não da conta: o que a fatia comprou —
// a parcela suprimida que continua lá — não aparece na saída.
func attackWorld(
	card WeaponCard, target AttackTarget, situations []SpecialSituation,
	d20 int, rollDie func(faces int) (int, error),
) *ecs.World {
	w := ecs.NewWorld()
	ecs.Set(w, w.Spawn(), theAttackResource{})
	ecs.Run(w, attackSystems(card, target, situations, d20, rollDie)...)
	return w
}
