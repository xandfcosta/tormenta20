package engine

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
	// Dice são as rolagens de dano na ordem, já na quantidade que o crítico
	// pediu. Vazio quando o ataque erra: quem erra não rola dano.
	Dice []int `json:"dice"`
	// Faces é o dado da arma (o 8 de "1d8"). Ele viaja porque a mesa lê a
	// NOTAÇÃO — "2d8+3" diz de onde os números vieram, e "8+5+3" faz quem olha
	// reconstruir a arma de cabeça.
	Faces     int `json:"faces"`
	RawDamage int `json:"rawDamage"` // dados + bônus, antes da RD
	Absorbed  int `json:"absorbed"`  // o que a RD comeu
	Damage    int `json:"damage"`    // o que o alvo perde de PV
}

// ResolveAttack rola um ataque contra um alvo e devolve a conta inteira.
//
//	fora, err := ResolveAttack(carta, AttackTarget{Defense: 15}, 19, rolar)
//	// fora.Critical == true, fora.Damage == 16 para 1d8+3 com x2
func ResolveAttack(
	card WeaponCard, alvo AttackTarget, d20 int, rolar func(faces int) (int, error),
) (AttackOutcome, error) {
	fora := AttackOutcome{Roll: d20, Total: d20 + card.Attack}

	// "Se o resultado é igual ou maior que a Defesa do alvo, você acerta"
	// (p230). O IGUAL decide todo ataque que empata, e é a metade que um `>`
	// perderia em silêncio.
	//
	// NÃO HÁ 20 AUTOMÁTICO NEM 1 AUTOMÁTICO, e a ausência é deliberada: a p220
	// define o teste como "1d20 + modificador, passa se for igual ou maior que a
	// CD", sem exceção para as pontas — e a p230 não acrescenta nenhuma. É
	// armadilha de hábito de outro sistema, e escrevê-la daria acerto onde o
	// livro dá erro.
	fora.Hit = fora.Total >= alvo.Defense
	if !fora.Hit {
		return fora, nil
	}

	// "Você faz um acerto crítico quando ACERTA um ataque rolando um valor igual
	// ou maior que a margem de ameaça" (p231). São as duas condições: bater a
	// margem num ataque que erra não é crítico coisa nenhuma.
	//
	// "Quando nenhuma margem aparece, será 20. Quando nenhum multiplicador
	// aparece, será x2" (p230) — por isso o zero do catálogo cai no padrão do
	// livro em vez de virar uma arma que nunca critica.
	margem, multiplicador := card.CritRange, card.CritMult
	if margem <= 0 {
		margem = 20
	}
	if multiplicador <= 0 {
		multiplicador = 2
	}
	// "Um alvo imune a acertos críticos ainda sofre o dano de um ataque normal"
	// (p231): a imunidade tira o crítico, nunca o ataque.
	fora.Critical = d20 >= margem && !alvo.CritImmune

	quantidade, faces, err := parseDiceNotation(card.Damage)
	if err != nil {
		return fora, err
	}
	// "Multiplica os DADOS de dano do ataque (incluindo quaisquer aumentos por
	// passos) pelo multiplicador da arma. Bônus numéricos de dano, assim como
	// dados extras, não são multiplicados" (p231). O exemplo trabalhado é da
	// p142: um dano de 1d8+3 torna-se 2d8+3 — mais DADOS, e o +3 uma vez só.
	if fora.Critical {
		quantidade *= multiplicador
	}
	fora.Faces = faces
	for i := 0; i < quantidade; i++ {
		valor, err := rolar(faces)
		if err != nil {
			return fora, err
		}
		fora.Dice = append(fora.Dice, valor)
		fora.RawDamage += valor
	}
	fora.RawDamage += card.DamageBonus

	// "Se uma criatura com RD 5 sofre um ataque que causa 8 pontos de dano,
	// perde apenas 3 PV" (p229).
	//
	// O PISO EM ZERO é decisão desta casa: o livro só dá o caso em que sobra
	// dano, e "ignora parte do dano que sofre" não descreve um ataque que
	// devolve PV. Sem o piso, uma RD alta viraria cura — e a cura tem regra
	// própria, que não é esta.
	fora.Absorbed = alvo.DamageReduction
	if fora.Absorbed > fora.RawDamage {
		fora.Absorbed = fora.RawDamage
	}
	fora.Damage = fora.RawDamage - fora.Absorbed
	return fora, nil
}
