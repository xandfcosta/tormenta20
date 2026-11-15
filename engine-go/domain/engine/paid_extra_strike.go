package engine

import (
	"fmt"
	"strconv"
)

// O ATAQUE PAGO QUE NÃO GASTA A VEZ (ALE-651).
//
// O Ataque Extra (guerreiro, p66) e o Golpe Relâmpago (lutador, p76) vendem, por PM, um ataque a mais depois
// de agredir. É a mesma forma do ataque extra da arma natural (p25) e do adicional do estilo (p125): a resposta
// do mestre ao golpe de agredir abre o extra na vez, ele não gasta ação, e cada fonte conta os próprios usos.
// O que muda é QUEM o dá — um poder, e não a arma —, e por isso ele viaja na carta como o `Natural` e o `Style`.
//
// O preço e o alcance moram no VERBETE (`ClassPower.ExtraStrike`), como o `Boost`: o PM é o do livro, e "só
// desarmado" é a diferença entre os dois.

// ClassPowerExtraStrike é o ataque adicional que o poder vende ao agredir — ver o arquivo.
type ClassPowerExtraStrike struct {
	PM      int  `json:"pm"`
	Unarmed bool `json:"unarmed,omitempty"`
}

// PaidExtraStrike é o ataque pago que esta carta pode fazer depois de agredir: o poder que o dá, o PM, e se ele
// só vale desarmado — e, então, só depois de agredir desarmado.
type PaidExtraStrike struct {
	PowerID string
	Name    string
	PM      int
	Unarmed bool
}

// paidExtraStrikes são os ataques pagos que os poderes da ficha dão, na ordem das classes.
func (r *Ruleset) paidExtraStrikes(ch Character) []PaidExtraStrike {
	chosen := parseChoiceSet(ch.ClassPowers)
	choices := parseClassChoices(ch.ClassChoices)
	var out []PaidExtraStrike
	for _, entry := range ch.Classes {
		for _, power := range r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName]) {
			if grant := power.ExtraStrike; grant != nil {
				out = append(out, PaidExtraStrike{PowerID: power.ID, Name: power.Name, PM: grant.PM, Unarmed: grant.Unarmed})
			}
		}
	}
	return out
}

// offerPaidExtraStrikes pendura em cada carta os ataques pagos que a alcançam: o desarmado só na carta
// desarmada, o resto em todas.
func offerPaidExtraStrikes(cards []WeaponCard, offers []PaidExtraStrike) {
	for i := range cards {
		for _, offer := range offers {
			if offer.Unarmed && !cards[i].Unarmed {
				continue
			}
			cards[i].PaidExtras = append(cards[i].PaidExtras, offer)
		}
	}
}

// O GOLPE IMPRUDENTE (lutador, p77): "Quando usa Golpe Relâmpago, você pode atacar de forma impulsiva. Se fizer
// isso, seus ataques desarmados recebem um dado de dano extra do mesmo tipo […], mas você sofre –5 na Defesa
// até o início de seu próximo turno." A escolha é feita ao pagar o Golpe Relâmpago (decisão do dono), e por
// isso o dado vale no golpe que vem dele.
const (
	ImpulsiveStrikePowerID  = "class.lutador.golpe-imprudente"
	ImpulsiveStrikeEffectID = "rule.golpe-imprudente"
)

// ImpulsiveStrikeCard é a carta com o dado a mais do Golpe Imprudente: um dado da própria arma, do mesmo tipo.
// Ele é "dado de dano extra", e dado extra não multiplica no crítico (p231) — por isso vai no `ExtraDamage`,
// como o 1d12 da Investida Imprudente. A cópia da lista não mexe na carta de origem.
//
//	card, err := ImpulsiveStrikeCard(desarmado) // card.ExtraDamage → [{1d8 impacto}] no desarmado de 1d8
func ImpulsiveStrikeCard(card WeaponCard) (WeaponCard, error) {
	_, faces, err := parseDiceNotation(card.Damage)
	if err != nil {
		return card, fmt.Errorf("o dado a mais do Golpe Imprudente é o da arma, e o dano de %s é %q: %w", card.Name, card.Damage, err)
	}
	die := "1d" + strconv.Itoa(faces)
	card.ExtraDamage = append(append([]ExtraDamage{}, card.ExtraDamage...), ExtraDamage{Dice: die, Type: card.DamageType})
	return card, nil
}

// ImpulsiveStrikeGuardModifiers é o –5 na Defesa que o Golpe Imprudente deixa até o próximo turno.
func ImpulsiveStrikeGuardModifiers() []Modifier {
	return []Modifier{{Target: ModifierTarget{K: "defense"}, Amount: -5, BonusType: "untyped", Note: "Golpe Imprudente"}}
}
