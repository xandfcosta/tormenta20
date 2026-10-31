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
	// DamageReduction é a RD GERAL do alvo (p229), já resolvida: qual RD vale
	// quando há várias fontes é decisão do `characterDamageReduction`, e ela não
	// muda por ataque.
	DamageReduction int
	// DamageReductionByType é a RD CONTRA UM TIPO — *"a redução pode ser contra
	// um ou mais tipos de dano específicos"* (p229). Chaveada pelo mesmo
	// vocabulário de tipo que a parcela de dano usa ("fogo", "frio", "ácido"),
	// porque é parcela a parcela que ela decide o que alcança.
	DamageReductionByType map[string]int
	// CritExtraIgnoreChance é a chance, em PORCENTO, de este alvo ignorar o
	// dano EXTRA de um acerto crítico (p166, p338).
	//
	// Ela não muda número nenhum: o app INFORMA e a MESA rola (decisão do dono,
	// ALE-436). Ela viaja para o resultado poder dizer, ao lado do extra
	// recortado, que existe uma chance de ele não valer.
	CritExtraIgnoreChance int
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
	// NonLethal é quanto do `Damage` não conta para sangrar nem para morrer
	// (p236). Ou TUDO, quando a arma é Piedosa, ou nada — a propriedade é da
	// arma, e o livro não dá meio-termo.
	NonLethal int `json:"nonLethal,omitempty"`
	// CriticalExtra é quanto do dano veio do CRÍTICO: os dados que o
	// multiplicador acrescentou mais o bônus que só existe nele (p336).
	//
	// Ele viaja recortado porque o livro dá a alguns alvos uma CHANCE de ignorar
	// justamente essa parte (p166, p338) — e quem rola essa chance é a mesa.
	CriticalExtra int `json:"criticalExtra,omitempty"`
	// CritExtraIgnoreChance é a chance do ALVO, em porcento, de ignorar o
	// `CriticalExtra`. Ela volta no resultado para a frase da mesa ter os dois
	// números lado a lado: o que o crítico acrescentou e a chance de ele não
	// valer.
	CritExtraIgnoreChance int `json:"critExtraIgnoreChance,omitempty"`
	// Announces são as condições que o LIVRO prevê para este golpe — o
	// sangramento da Sanguinária, o atordoamento da Trovejante (p335-336).
	//
	// INFORMAÇÃO e não aplicação, que é a regra da casa e o desenho que a
	// manobra já usa (p234, ALE-423): o motor diz o que o livro reserva, e quem
	// põe a condição na criatura é o mestre, no gesto da fila.
	Announces []AnnouncedCondition `json:"announces,omitempty"`
	Absorbed  int                  `json:"absorbed"` // o que a RD comeu
	Damage    int                  `json:"damage"`   // o que o alvo perde de PV
}

// announcedBy escolhe, entre as condições que a arma reserva, as que o
// DESFECHO deste golpe alcança.
func announcedBy(card WeaponCard, out AttackOutcome) []AnnouncedCondition {
	if !out.Hit {
		return nil
	}
	saem := []AnnouncedCondition{}
	for _, anuncio := range card.Announces {
		if anuncio.OnCritical && !out.Critical {
			continue
		}
		saem = append(saem, anuncio)
	}
	if len(saem) == 0 {
		return nil
	}
	return saem
}

// AnnouncedCondition é uma condição que o livro reserva a este golpe.
//
// Ela não carrega veredito NENHUM — nem se pegou, nem quantas rodadas dura, nem
// se o teste de resistência passou. Os três são do mestre, e o
// `TestTheAnnouncedConditionCarriesNoVerdict` existe para que a próxima pessoa
// não acrescente um deles sem perceber o que está mudando.
type AnnouncedCondition struct {
	// Condition é o id da condição no catálogo (`sangrando`, `atordoado`), e
	// nunca o texto de tela: quem traduz é o `book.ConditionName`, pela mesma
	// razão que a faixa da manobra traduz (o id não vai para a tela).
	Condition string `json:"condition"`
	// OnCritical diz que o livro só a reserva ao acerto crítico.
	OnCritical bool `json:"onCritical,omitempty"`
	// Note é o que o verbete escreve ao lado da condição — duração, teste que
	// evita, limite por cena. Texto do LIVRO, para o mestre arbitrar com a
	// regra na frente.
	Note string `json:"note,omitempty"`
}

// O ATAQUE EM DOIS TEMPOS: `JudgeAttack` é o teste, e `RollAttackDamage` é o
// dano, rolado DEPOIS da decisão do mestre. O crítico muda quantos dados se
// rolam (p231), e quem diz se houve crítico é o mestre — rolado junto do d20, o
// dano sairia multiplicado pela sugestão do livro e não pela palavra dele.

// Ruling é a decisão do mestre sobre um golpe que pede dano: acerto ou crítico.
// O erro não tem dano a rolar, e por isso não é uma decisão que chega aqui.
type Ruling string

const (
	RulingHit      Ruling = "hit"
	RulingCritical Ruling = "critical"
)

// AttackDie é o d20 do ataque em suas duas leituras: o que SOMA no total e o
// que as pontas da p221 e a margem de ameaça leem. São o mesmo número, menos
// quando um poder manda contar o dado como 20 natural.
type AttackDie struct{ Rolled, Natural int }

// D20 é o dado rolado, lido como ele é.
func D20(rolled int) AttackDie { return AttackDie{Rolled: rolled, Natural: rolled} }

// D20CountingAsNaturalTwenty é o d20 que "será considerado um 20 natural"
// (Sorte de Nimb, p48, ALE-508): o ROLADO soma no total, e o 20 decide o acerto
// e o crítico (p221, p231).
func D20CountingAsNaturalTwenty(rolled int) AttackDie {
	return AttackDie{Rolled: rolled, Natural: 20}
}

// JudgeAttack rola o TESTE de ataque e devolve a sugestão do livro — acerto,
// crítico — sem dano nenhum. As situações são as da Tabela 5-3 (p239) que valem
// agora; o `rollDie` só é pedido pelo d10 da camuflagem (p238).
//
//	out, err := JudgeAttack(carta, AttackTarget{Defense: 13}, nil, D20(19), rolar)
//	// out.Hit && out.Critical na espada longa (margem 19); out.Damage == 0
func JudgeAttack(
	card WeaponCard, target AttackTarget, situations []SpecialSituation,
	die AttackDie, rollDie func(faces int) (int, error),
) (AttackOutcome, error) {
	w := runAttackWorld(append(judgeSystems(card, target, situations, die),
		undoTheHitOnConcealment(rollDie)))
	return outcomeFromWorld(w), faultOf(w)
}

// RollAttackDamage rola o dano de um golpe que o mestre JÁ decidiu: o crítico
// multiplica os dados da arma porque ele disse crítico, e não porque o d20
// caiu na margem. O `rollDie` é o servidor, ou os dados que a mesa rolou na mão
// (`DiceRolledByHand`).
//
//	out, err := RollAttackDamage(carta, alvo, nil, D20(12), RulingCritical, rolar)
//	// 2d8 + 3 numa espada longa, mesmo com o 12 fora da margem
func RollAttackDamage(
	card WeaponCard, target AttackTarget, situations []SpecialSituation,
	die AttackDie, ruling Ruling, rollDie func(faces int) (int, error),
) (AttackOutcome, error) {
	w := runAttackWorld(append(judgeSystems(card, target, situations, die),
		damageSystems(card, target, ruling, rollDie)...))
	out := outcomeFromWorld(w)
	// A CHANCE DO ALVO ecoa no resultado, e só quando há extra para ela
	// alcançar: anunciar "25% de ignorar o extra" num acerto normal mandaria a
	// mesa rolar por nada.
	if out.CriticalExtra > 0 {
		out.CritExtraIgnoreChance = target.CritExtraIgnoreChance
	}
	out.Announces = announcedBy(card, out)
	// O NÃO LETAL NÃO É SISTEMA: a propriedade é da ARMA e vale para o dano
	// inteiro, depois da RD — é esse o número que não pode fazer sangrar nem
	// matar (p236). O TIPO é um OU-EXCLUSIVO entre a natureza da arma e a
	// ESCOLHA de quem ataca: a espada comum mata, a Piedosa não (p336), e trocar
	// inverte os dois. Os −5 da escolha são a linha
	// `AttackerSwitchesTheDamageType` da Tabela 5-3, que desloca o teste.
	if card.NonLethal != hasSituation(situations, AttackerSwitchesTheDamageType) {
		out.NonLethal = out.Damage
	}
	return out, faultOf(w)
}

// DiceGroup é um grupo de dados da mesma parcela: os 3d8 do machado, o 1d6 de
// fogo do encanto. Type vazio é o tipo da própria arma.
type DiceGroup struct {
	Count int    `json:"count"`
	Faces int    `json:"faces"`
	Type  string `json:"type,omitempty"`
}

// DamageFormula é o que se rola de dano depois da decisão, na ORDEM em que os
// dados são lidos: a arma primeiro, os extras depois. É a ordem em que a mão
// digita e em que o `DiceRolledByHand` os entrega.
type DamageFormula struct {
	Dice  []DiceGroup `json:"dice"`
	Bonus int         `json:"bonus"`
}

// DamageToRoll diz o que rolar de dano depois da decisão do mestre: o ×N da
// ARMA no crítico, e os extras fora dele (p231).
//
//	f, _ := DamageToRoll(machadoFlamejante, RulingCritical)
//	// f.Dice == [{3 8 ""} {1 6 "fogo"}]
func DamageToRoll(card WeaponCard, ruling Ruling) (DamageFormula, error) {
	w := runAttackWorld([]ecs.System{
		applyTheRuling(card, ruling), spawnTheDamageParcels(card),
		multiplyTheWeaponDice, dropWhatIsNotCritical,
	})
	formula := DamageFormula{}
	ecs.Each(w, func(e ecs.Entity, parcel damageDice) {
		if dead(w, e) {
			return
		}
		group := DiceGroup{Count: parcel.Count, Faces: parcel.Faces}
		if kind, has := ecs.Get[damageTypeOf](w, e); has {
			group.Type = kind.Name
		}
		formula.Dice = append(formula.Dice, group)
	})
	ecs.Each(w, func(e ecs.Entity, parcel damageFlat) {
		if !dead(w, e) {
			formula.Bonus += parcel.Amount
		}
	})
	return formula, faultOf(w)
}

// CriticalOf é a margem de ameaça e o multiplicador da arma, com o padrão do
// livro: "Quando nenhuma margem aparece, será 20. Quando nenhum multiplicador
// aparece, será x2" (p230). É o que o botão do mestre escreve ("Crítico ×3").
func CriticalOf(card WeaponCard) (margin, multiplier int) {
	margin, multiplier = card.CritRange, card.CritMult
	if margin <= 0 {
		margin = 20
	}
	if multiplier <= 0 {
		multiplier = 2
	}
	return margin, multiplier
}

func runAttackWorld(systems []ecs.System) *ecs.World {
	w := ecs.NewWorld()
	ecs.Set(w, w.Spawn(), theAttackResource{})
	ecs.Run(w, systems...)
	return w
}

func faultOf(w *ecs.World) error {
	if fault, has := ecs.Get[attackFault](w, theResource(w)); has {
		return fault.Err
	}
	return nil
}

// hasSituation diz se esta linha da Tabela 5-3 vale neste ataque.
//
// Um laço e não um `map`: a lista tem no máximo uma mão de linhas, e montar um
// conjunto para perguntar uma vez é cerimônia mais cara que a varredura.
func hasSituation(situations []SpecialSituation, want SpecialSituation) bool {
	for _, s := range situations {
		if s == want {
			return true
		}
	}
	return false
}
