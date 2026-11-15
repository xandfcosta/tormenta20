package engine

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
