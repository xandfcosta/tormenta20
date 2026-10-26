package sheetui

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// AS LINHAS QUE TODA FICHA DE COMBATE TEM, por grupo de custo — o ataque e a
// manobra (padrão), o deslocamento (movimento), a investida e a corrida
// (completa). Elas saíram do `actions.go` quando ele passou do teto: lá mora
// como a superfície se MONTA; aqui, o que o livro dá para fazer na vez (p233-235),
// que muda quando uma ação nova do livro entra.

// standardRows são as ações padrão que toda ficha de combate tem.
//
// O ATAQUE VEM PRIMEIRO, e o livro diz por quê: *"fazer um ataque ou lançar uma
// magia são as ações padrão mais comuns"* (p233). A 390px a primeira linha é a
// única que se lê sem rolar.
func standardRows(combat panelCombat, melee int, computed engine.ComputedSheet, boosts [][]attackBoost) []actionRow {
	rows := make([]actionRow, 0, len(combat.Weapons)+2)
	for i, weapon := range combat.Weapons {
		var offered []attackBoost
		if i < len(boosts) {
			offered = boosts[i]
		}
		rows = append(rows, actionRow{
			Weapon: i,
			Boosts: offered,
			Name:   "Atacar · " + weapon.Name,
			Detail: weapon.Damage + " · " + weapon.Crit + " · " + weapon.Skill,
			Value:  weapon.Attack,
			Can:    true,
			Moves:  []string{"attack", "damage"},
			// A ARMA VAI NO CAMINHO, pelo ÍNDICE: o `Propose` escolhe por índice
			// e já recusa o que não existe, e um nome no endereço obrigaria o
			// servidor a casar texto para achar de volta a mesma arma.
			Rolls: []actionRoll{{
				Label: "Rolar", Path: fmt.Sprintf("/acoes/atacar/%d", i), NeedsTarget: true,
			}},
		})
	}
	// A MANOBRA SÓ EXISTE COM ARMA CORPO A CORPO NA MÃO: *"não é possível fazer
	// manobras de combate com ataques à distância"* (p234). Oferecê-la ao
	// arqueiro seria desenhar um gesto que a regra recusa.
	if hasMeleeWeapon(combat.Weapons) {
		rows = append(rows, actionRow{
			Name: "Manobra",
			// O DETALHE DEIXA DE LISTAR as cinco quando elas viram botão: a
			// mesma lista escrita duas vezes na mesma linha é a segunda que
			// envelhece.
			Detail: "teste oposto de Luta · p234",
			Value:  book.WithSign(melee),
			Can:    true,
			Moves:  []string{"attack"},
			Rolls:  maneuverRolls(),
		})
	}
	// FINTAR é teste de Enganação contra os Reflexos do alvo (p234) — é a única
	// ação padrão do livro que não sai de Luta nem de Pontaria.
	rows = append(rows, actionRow{
		Name:   "Fintar",
		Detail: "Enganação contra os Reflexos do alvo · p234",
		Value:  book.WithSign(expertiseOrZero(computed, "Enganação", "charisma").Total),
		Can:    true,
		Moves:  []string{"expertise:Enganação"},
		Rolls:  []actionRoll{expertiseRoll("Enganação")},
	})
	return rows
}

// movementRow é o deslocamento, em METROS.
//
// O motor o guarda em QUADRADOS de 1,5m porque é assim que o tabuleiro conta, e
// a tela diz metros porque é assim que a mesa fala — a conversão mora aqui, de um
// lado só.
func movementRow(computed engine.ComputedSheet) actionRow {
	meters := float64(computed.Displacement.Total) * 1.5
	written := strconv.FormatFloat(meters, 'f', -1, 64)
	return actionRow{
		Name:   "Mover",
		Detail: "percorrer o seu deslocamento · p233",
		Value:  strings.Replace(written, ".", ",", 1) + "m",
		Can:    true,
	}
}

// fullRows são as duas completas que saem de número que a ficha já tem.
//
// O GOLPE DE MISERICÓRDIA fica de fora apesar de ser a terceira da p235: ele não
// tem teste nem número — é crítico automático mais uma chance de morte que o
// mestre rola —, e a decisão do dono é que linha sem número nem botão não entra.
func fullRows(combat panelCombat, computed engine.ComputedSheet, reckless bool) []actionRow {
	return []actionRow{chargeRow(combat, reckless), {
		Name:   "Corrida",
		Detail: "Atletismo · p235",
		Value:  book.WithSign(expertiseOrZero(computed, "Atletismo", "strength").Total),
		Can:    true,
		Moves:  []string{"expertise:Atletismo"},
		Rolls:  []actionRoll{expertiseRoll("Atletismo")},
	}}
}

// chargeRow é a investida (p235), com a Imprudente (p42) para quem tem o poder.
//
// A ARMA É A PRIMEIRA CORPO A CORPO empunhada: a investida "faz um ataque
// corpo a corpo", e o número da linha é o daquela arma com o +2. Sem arma
// corpo a corpo a linha fica e diz por quê — saber que a investida existe é
// informação. A linha reta, o dobro do deslocamento e o terreno são do mestre.
func chargeRow(combat panelCombat, reckless bool) actionRow {
	row := actionRow{
		Name:   "Investida",
		Detail: "em linha reta · +2 no ataque, −2 na Defesa até a sua vez · p235",
		Moves:  []string{"attack"},
	}
	weapon := -1
	for i, w := range combat.Weapons {
		if w.Skill == "Luta" {
			weapon = i
			break
		}
	}
	if weapon < 0 {
		row.Why = "a investida termina num ataque corpo a corpo, e não há arma de Luta na mão"
		return row
	}
	row.Can = true
	row.Value = book.WithSign(combat.Weapons[weapon].AttackTotal + engine.ChargeAttackBonus)
	row.Rolls = []actionRoll{{
		Label: "Rolar", Path: fmt.Sprintf("/acoes/investida/%d", weapon), NeedsTarget: true,
	}}
	if reckless {
		row.Detail += " · Imprudente: −5 na Defesa e +1d12 no dano (p42)"
		row.Rolls = append(row.Rolls, actionRoll{
			Label: "Imprudente", Path: fmt.Sprintf("/acoes/investida-imprudente/%d", weapon), NeedsTarget: true,
		})
	}
	return row
}

// ownsPower diz se a ficha tem o poder, pelo acervo que a aba Poderes já
// montou — ativo ou passivo.
func ownsPower(powers powersPanel, id string) bool {
	for _, rows := range [][]powerRow{powers.Actions, powers.Passives} {
		for _, row := range rows {
			if row.ID == id {
				return true
			}
		}
	}
	return false
}

// maneuverRolls são as cinco manobras da p234, cada uma um botão.
//
// A LISTA VEM DO MOTOR e não é escrita aqui: ela já é fechada em
// `engine.ManeuversOfTheBook`, e uma segunda cópia erraria um nome no dia em
// que alguém a editasse de um lado só — o casamento é por TEXTO, e um nome
// errado vira uma manobra que o servidor recusa sem ninguém entender por quê.
//
// O RÓTULO é o nome da manobra e não "Rolar": cinco botões iguais obrigam a
// contar a posição para saber qual é qual.
func maneuverRolls() []actionRoll {
	kinds := engine.ManeuversOfTheBook()
	rolls := make([]actionRoll, 0, len(kinds))
	for _, kind := range kinds {
		rolls = append(rolls, actionRoll{
			Label:       capitalized(kind),
			Path:        "/acoes/manobra/" + kind,
			NeedsTarget: true,
		})
	}
	return rolls
}

// capitalized sobe a primeira letra. Cópia da irmã do pacote da Mesa, e a cópia
// é deliberada: três linhas sobre `string[:1]` custam menos que um lugar novo
// para morar, e as cinco manobras são ASCII.
func capitalized(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// expertiseRoll é o gesto de rolar uma PERÍCIA, e ele reusa o comando que a aba
// Perícias já tem.
//
// SEM ALVO, e isso não é esquecimento: a finta é um teste de Enganação e a
// corrida um de Atletismo, e um teste não aponta para ninguém. A CD — e, na
// finta, os Reflexos do alvo — é de quem mestra, e por isso a faixa do teste não
// tem veredicto.
func expertiseRoll(name string) actionRoll {
	return actionRoll{Label: "Rolar", Path: "/pericias/rolar/" + url.PathEscape(name)}
}

// hasMeleeWeapon pergunta pela PERÍCIA do cartão, que é "Luta" ou "Pontaria".
//
// Pela perícia e não por uma lista de nomes de arma: quem decide se a arma é
// corpo a corpo é o catálogo, e o cartão já carrega a resposta dele.
func hasMeleeWeapon(weapons []weaponTile) bool {
	for _, weapon := range weapons {
		if weapon.Skill == "Luta" {
			return true
		}
	}
	return false
}
