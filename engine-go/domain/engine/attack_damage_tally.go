package engine

import "t20engine/domain/ecs"

// A CONTA DO DANO, depois de os dados rolarem: somar as parcelas vivas, aplicar o que o corpo do alvo faz
// com elas, e tirar a RD. Ela mora fora do `attack_ecs.go` porque muda por outra razão: lá está o que
// decide QUAIS parcelas existem e quantos dados rolam; aqui, o que cada uma vale quando chega ao alvo.

// rawTally, bodyTally e absorbedTally são as três últimas contas, e são três porque são
// três sistemas: quem soma as parcelas não sabe do alvo, quem aplica o corpo do alvo não sabe de RD,
// e quem aplica a RD não sabe de parcela.
type rawTally struct {
	Raw int
	// ByType é o cru POR TIPO de dano, e ele existe para a RD contra um tipo
	// (p229) saber o que alcança. A soma dele é o `Raw` — toda parcela tem tipo,
	// nem que seja o da própria arma.
	ByType map[string]int
	// CriticalExtra é quanto do cru veio do crítico — ver `criticalExtra`.
	CriticalExtra int
	// DiceByType é quantos DADOS de cada tipo rolaram, e ele existe para o dano a mais POR DADO (os Ossos
	// Frágeis, p28) saber quanto acrescentar.
	DiceByType map[string]int
}

// bodyTally é o cru depois do que o corpo do alvo faz com ele, e antes da RD — ver `theTargetsBodyTakesIt`.
type bodyTally struct {
	Raw    int
	ByType map[string]int
	Taken  []TakenRow
}
type absorbedTally struct{ Absorbed, Final int }

// sumTheParcels soma o que sobrou vivo.
func sumTheParcels(card WeaponCard) ecs.System {
	return func(w *ecs.World) {
		raw := 0
		byType := map[string]int{}
		diceByType := map[string]int{}
		// O TIPO DA PARCELA é o dela quando ela traz um, e o da ARMA quando não:
		// os dados da espada e o bônus de Força são o corte da espada. Sem este
		// recuo, uma redução de corte não alcançaria o dano da arma de corte —
		// que é o caso da Armadura Óssea do osteon (RD 5 a corte, frio e
		// perfuração).
		extra := 0
		typeOf := func(e ecs.Entity) string {
			if own, has := ecs.Get[damageTypeOf](w, e); has {
				return own.Name
			}
			return card.DamageType
		}
		soma := func(e ecs.Entity, amount int) {
			raw += amount
			if _, inteira := ecs.Get[criticalExtra](w, e); inteira {
				extra += amount
			}
			if tipo := typeOf(e); tipo != "" {
				byType[tipo] += amount
			}
		}
		ecs.Each(w, func(e ecs.Entity, rolled diceRolled) {
			if dead(w, e) {
				return
			}
			soma(e, rolled.Total)
			diceByType[typeOf(e)] += len(rolled.Values)
			// O EXCEDENTE da parcela que o crítico multiplicou: o que passa da
			// contagem de antes é o que ele acrescentou.
			if before, multiplicada := ecs.Get[criticalMultiplied](w, e); multiplicada {
				for i := before.Base; i < len(rolled.Values); i++ {
					extra += rolled.Values[i]
				}
			}
		})
		ecs.Each(w, func(e ecs.Entity, parcel damageFlat) {
			if !dead(w, e) {
				soma(e, parcel.Amount)
			}
		})
		setResource(w, rawTally{Raw: raw, ByType: byType, CriticalExtra: extra, DiceByType: diceByType})
	}
}

// theTargetsBodyTakesIt aplica o que o CORPO do alvo faz com o dano, antes da RD (ALE-650).
//
//   - A IMUNIDADE come a parcela do tipo inteira: "Você é imune a dano desse tipo" (Fonte Elemental, p27). A
//     cura pelo dano MÁGICO desse tipo não entra: o golpe não diz se o dano é mágico, e quem decide é o mestre.
//   - O DANO A MAIS POR DADO soma ao tipo dele: "Você sofre 1 ponto de dano adicional por dado de dano de
//     impacto" (Ossos Frágeis, p28). Ele é dano sofrido como qualquer outro, e por isso a RD vem DEPOIS.
//
// A imunidade vem primeiro porque o tipo a que se é imune não tem dado que acrescente: zero mais um por dado
// continuaria sendo dano que o livro diz que não se sofre.
func theTargetsBodyTakesIt(target AttackTarget) ecs.System {
	return func(w *ecs.World) {
		tally := resourceOf[rawTally](w)
		body := bodyTally{Raw: tally.Raw, ByType: map[string]int{}}
		for tipo, amount := range tally.ByType {
			body.ByType[tipo] = amount
		}
		immune := map[string]bool{}
		for _, line := range target.ImmuneTo {
			immune[line.Type] = true
			if eaten := body.ByType[line.Type]; eaten > 0 {
				body.Raw -= eaten
				body.ByType[line.Type] = 0
				body.Taken = append(body.Taken, TakenRow{Label: "imune a " + line.Type + " (" + line.Source + ")", Amount: -eaten})
			}
		}
		for _, line := range target.TakenPerDie {
			dice := tally.DiceByType[line.Type]
			if immune[line.Type] || dice == 0 {
				continue
			}
			more := dice * line.Amount
			body.Raw += more
			body.ByType[line.Type] += more
			body.Taken = append(body.Taken, TakenRow{Label: line.Source, Amount: more})
		}
		setResource(w, body)
	}
}

// absorbWithDamageReduction aplica a RD do alvo, e são DUAS famílias.
//
//	"Se uma criatura com RD 5 sofre um ataque que causa 8 pontos de dano, perde
//	 apenas 3 PV. A redução pode ser contra um ou mais tipos de dano
//	 específicos. Assim, uma criatura com redução de fogo 10 ignora 10 pontos de
//	 dano de fogo, mas sofre dano de outros tipos normalmente." (p229)
//
// A DO TIPO VEM PRIMEIRO, e a ordem é decisão desta casa — o livro não diz o
// que fazer quando as duas alcançam o mesmo ataque. É a única ordem bem
// definida: a geral come de um total, e depois dela ninguém sabe quanto do que
// sobrou era fogo. Cada uma age UMA VEZ por ataque, que é como o livro conta
// ("ignora 10 pontos de dano de fogo", não "de cada parcela").
//
// O PISO EM ZERO é decisão desta casa: o livro só dá o caso em que sobra dano, e
// "ignora parte do dano que sofre" não descreve um ataque que devolve PV. Sem o
// piso, uma RD alta viraria cura — e a cura tem regra própria, que não é esta.
func absorbWithDamageReduction(target AttackTarget, ignored int) ecs.System {
	return func(w *ecs.World) {
		tally := resourceOf[bodyTally](w)
		general, byType := withoutIgnoredReduction(target, ignored)
		absorbed := 0
		for tipo, rd := range byType {
			absorbed += min(rd, tally.ByType[tipo])
		}
		absorbed += min(general, tally.Raw-absorbed)
		setResource(w, absorbedTally{Absorbed: absorbed, Final: tally.Raw - absorbed})
	}
}
