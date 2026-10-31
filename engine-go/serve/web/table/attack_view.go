package table

import (
	"fmt"
	"strconv"
	"strings"
	"t20engine/domain/engine"

	"t20engine/domain/book"
	"t20engine/domain/live"
)

// O ATAQUE PROPOSTO, como a mesa o lê.
//
// O cartão do MESTRE é o duelo — a rolagem contra o que ela enfrenta — e muda
// por TEMPO (`stageOf`): julgar, esperar o dano, confirmar. O do JOGADOR é
// recortado (`forPlayer`): a própria rolagem, a decisão do mestre e o dano que
// ela pede, nunca a Defesa nem a sugestão do livro.
type attackProposal struct {
	Stage         string
	Attacker      string
	Target        string
	TargetEntryID string
	Weapon        string
	// Action é o nome da manobra ("Derrubar"), que a cabeça das faixas mostra
	// ao lado de quem ataca quem; vazio no golpe comum.
	Action   string
	Maneuver bool
	// Tally é a conta inteira numa linha, para os Detalhes do mestre — "24 vs 17
	// · 2d8+3 (8+5) · RD 5 → 11" — e, no cartão do jogador, a rolagem dele.
	Tally string
	// Situations são os motivos da Tabela 5-3, em prosa e FORA da conta.
	Situations string
	// Boost é o impulso de PM que o golpe leva e o que a confirmação cobra —
	// "Golpe Divino · 3 PM ao confirmar" (ALE-487).
	Boost string
	// Guard é a guarda que a investida abre em quem investiu (p235). Para o
	// NPC, que não tem ficha onde a penalidade more, a frase é o que o mestre
	// aplica.
	Guard string
	// Mine diz se quem olha foi quem rolou: só ele cancela e rola o dano do que
	// é dele. O mestre faz os dois por qualquer um.
	Mine bool
	// Rerolled diz os "rolar de novo" pagos, e Offers são os botões de quem
	// rolou (p226, ALE-508) — só antes da decisão.
	Rerolled string
	Offers   []rerollOffer

	// O MESTRE: o duelo, os três botões e a razão da sugestão, e as linhas do
	// dano rolado.
	Duel     duelView
	Verdicts []verdictButton
	Hint     string
	Rows     []duelRow
	Notes    []string
	// ToRoll é o dano que a decisão pede ("2d8 + 3"); Hand são os campos da
	// mão, um por dado.
	ToRoll    string
	Hand      []handGroup
	HandBonus string
	// Said é a decisão do mestre como o jogador a lê ("crítico ×2").
	Said string
	// DamageLine é o dano rolado na notação, e DamageTotal o cru: o jogador não
	// vê a RD do alvo.
	DamageLine  string
	DamageTotal int
}

// attackProposalOf traduz o provisório do regime no cartão do MESTRE, ou nil
// quando não há ataque pendurado.
//
// Os NOMES vêm da fila e não do provisório: a linha pode ter sido renomeada
// entre a rolagem e o desenho, e o nome que a mesa lê é o de agora.
func attackProposalOf(st *live.SessionRuntimeState, userID int64) *attackProposal {
	if st == nil || st.PendingAttack == nil {
		return nil
	}
	pa := st.PendingAttack
	out := &attackProposal{
		Stage:    stageOf(*pa),
		Attacker: entryLabel(st, pa.AttackerEntryID), Target: targetLabel(*pa, st),
		TargetEntryID: pa.TargetEntryID, Weapon: pa.Weapon,
		Tally: attackLine(*pa), Situations: writtenSituations(*pa),
		Boost:    writtenBoost(*pa),
		Guard:    writtenGuard(*pa, entryLabel(st, pa.AttackerEntryID)),
		Mine:     pa.ByUserID == userID,
		Rerolled: writtenRerolls(pa.Rerolls, pa.Roll),
		Said:     rulingSaid(*pa),
	}
	if pa.DamageToRoll != nil {
		out.ToRoll, out.Hand = toRollLine(*pa.DamageToRoll), handGroupsOf(*pa.DamageToRoll)
		out.HandBonus = strings.TrimSpace(signedTerm(pa.DamageToRoll.Bonus))
	}
	if pa.Maneuver != nil {
		var note string
		out.Maneuver, out.Action = true, maneuverSeal(*pa.Maneuver)
		out.Duel, note = maneuverDuel(*pa)
		out.Tally = maneuverLine(*pa)
		out.Rows = maneuverConsequence(*pa)
		// O EFEITO DA MARGEM DE 5 já está na consequência; a nota só fala
		// sozinha no empate, que é o "outro teste" da p234.
		if note != "" && pa.Maneuver.Margin < 5 {
			out.Hint = capitalize(note) + "."
		}
		return out
	}
	out.Duel = strikeDuel(*pa)
	switch out.Stage {
	case stageJudge:
		out.Verdicts, out.Hint = verdictsOf(*pa), suggestionHint(*pa)
	case stageConfirm:
		out.Rows, out.Notes = damageRows(*pa), damageNotes(*pa)
		out.DamageLine = damageSum(*pa) + " = " + strconv.Itoa(pa.RawDamage)
		out.DamageTotal = pa.RawDamage
	}
	return out
}

// maneuverSeal é o NOME da manobra em curso ("Derrubar"), que a cabeça das
// duas faixas mostra. O desfecho não mora aqui: para o mestre ele é o
// `maneuverVerdict`, e o jogador não o vê (ALE-532).
func maneuverSeal(m live.ManeuverRoll) string {
	if nome, conhecida := maneuverNames[m.Kind]; conhecida {
		return nome
	}
	return "Manobra"
}

// maneuverNames são as cinco da p234 como a mesa as lê, no INFINITIVO.
//
// Eram pares — "Derrubou" / "não derrubou" —, e o passado afirmava um desfecho
// que o sistema não tem mais como saber. O infinitivo diz o que foi TENTADO, que
// é o que de fato aconteceu.
var maneuverNames = map[string]string{
	"agarrar":  "Agarrar",
	"derrubar": "Derrubar",
	"desarmar": "Desarmar",
	"empurrar": "Empurrar",
	"quebrar":  "Quebrar",
}

// maneuverLine escreve a conta do teste OPOSTO, e ela é a resposta inteira: dois
// totais e a diferença entre eles.
//
// A linha é a CONTA, e ela é do mestre: o selo ao lado diz o desfecho que os
// números sugerem (`maneuverVerdict`), e a diferença aparece porque é a
// subtração que ele faria de cabeça.
//
//	maneuverLine(...) // "19 vs 10 · diferença +9 · vitória deixa caído"
func maneuverLine(pa live.PendingAttack) string {
	m := *pa.Maneuver
	// O NOME DA MANOBRA NÃO ENTRA AQUI: o crachá ao lado já o diz, e repeti-lo
	// gastaria o começo da linha — que a 390px é o pedaço mais caro dela.
	line := fmt.Sprintf("%d vs %d · diferença %+d", pa.Total, m.Opposed, m.Margin)
	// O QUE O LIVRO PREVÊ, dito SEMPRE e não só na vitória: o sistema não sabe
	// quem venceu, e esta linha existe justamente para o mestre decidir com a
	// consequência à vista. Nem toda manobra deixa condição — o desarmar derruba
	// um item —, e por isso ela só aparece quando há.
	//
	// O NOME ESCRITO e não o id: "fica caido" é o identificador vazando para a
	// tela. Quem o traduz é o catálogo, que é onde o nome é autorado.
	if m.ConditionOnAWin != "" {
		line += " · vitória deixa " + strings.ToLower(book.ConditionName(m.ConditionOnAWin))
	}
	return line
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// attackLine escreve a conta inteira do ataque.
//
// Um ataque que ERRA para na comparação: não houve dano, e escrever "0 de dano"
// convidaria a procurar de onde o zero saiu. A RD só aparece quando ela agiu —
// "RD 0" é ruído numa linha que a mesa lê no meio do turno.
//
//	attackLine(...) // "24 vs 17 · 2d8+3 (8+5) · RD 5 → 11"
func attackLine(pa live.PendingAttack) string {
	line := fmt.Sprintf("%d vs %d", pa.Total, pa.Defense)
	// O NATURAL QUE DESMENTE A COMPARAÇÃO é nomeado (p221): "ERROU · 14 vs 13"
	// é a conta certa com cara de regra quebrada. Quando a comparação já conta a
	// história, o natural não entra — seria ruído no meio do turno.
	switch {
	case pa.Roll == 1 && pa.Total >= pa.Defense:
		line += " · 1 natural erra"
	case pa.Roll == 20 && pa.Total < pa.Defense:
		line += " · 20 natural acerta"
	}
	if !pa.Hit || len(pa.Dice) == 0 {
		return line
	}
	line += " · " + damageDice(pa)
	// SEM RD AGINDO o total É o dano, e escrever os dois repete o número: a
	// primeira versão desta linha dizia "1d4 (4) = 4 → 4 de dano", e foi olhar a
	// tela que a pegou. A seta só existe quando há de onde para onde.
	if pa.Absorbed == 0 {
		return line + fmt.Sprintf(" = %d de dano%s%s%s",
			pa.Damage, writtenNonLethal(pa), writtenCritChance(pa), writtenForeseen(pa))
	}
	return line + fmt.Sprintf(" = %d · RD %d → %d de dano%s%s%s",
		pa.RawDamage, pa.Absorbed, pa.Damage, writtenNonLethal(pa), writtenCritChance(pa),
		writtenForeseen(pa))
}

// writtenCritChance é a chance de o alvo ignorar o que o crítico acrescentou.
//
// Ela vem com o NÚMERO a tirar, porque é esse o par que serve à mesa: o app
// informa e o mestre rola (ALE-436). E ela só aparece quando há o que ignorar —
// anunciar uma chance num acerto normal manda rolar por nada, que é a mesma
// razão de a RD só aparecer quando agiu.
func writtenCritChance(pa live.PendingAttack) string {
	if pa.CritExtraIgnoreChance == 0 || pa.CriticalExtra == 0 {
		return ""
	}
	return fmt.Sprintf(" · %d%% de ignorar os %d do crítico",
		pa.CritExtraIgnoreChance, pa.CriticalExtra)
}

// writtenForeseen são as condições que o LIVRO reserva a este golpe.
//
// "o livro prevê" e não "a vítima fica": a diferença entre as duas frases é a
// regra da casa inteira — a primeira informa o mestre, a segunda decide por
// ele. É a mesma escolha de palavra que a faixa da manobra fez ao trocar
// "Derrubou" pelo infinitivo (ALE-423).
//
// O NOME vai traduzido, porque id não vai para a tela, e a NOTA do livro vem
// junto: sem a duração e o teste que evita, o mestre teria de abrir o verbete
// no meio do turno.
func writtenForeseen(pa live.PendingAttack) string {
	if len(pa.Announces) == 0 {
		return ""
	}
	ditas := make([]string, 0, len(pa.Announces))
	for _, a := range pa.Announces {
		dita := book.ConditionName(a.Condition)
		if a.Note != "" {
			dita += " (" + a.Note + ")"
		}
		ditas = append(ditas, dita)
	}
	return " · o livro prevê: " + strings.Join(ditas, ", ")
}

// targetLabel é o nome do que foi atacado, e ele tem DUAS procedências.
//
// A criatura vem da FILA e não do provisório, porque a linha pode ter sido
// renomeada entre a rolagem e o desenho — o nome que a mesa lê é o de agora. O
// OBJETO não tem linha, então o nome dele viaja no provisório; é a única coisa
// que a faixa recebe sobre ele, e sem isso ela diria "acertou" sem dizer o quê.
func targetLabel(pa live.PendingAttack, st *live.SessionRuntimeState) string {
	if pa.TargetTokenID != "" {
		return pa.TargetLabel
	}
	return entryLabel(st, pa.TargetEntryID)
}

func entryLabel(st *live.SessionRuntimeState, entryID string) string {
	if i := live.FindEntryIndex(st, entryID); i >= 0 {
		return st.Initiative[i].Label
	}
	return "quem saiu da fila"
}

// writtenBoost é o impulso do golpe com o PM que a confirmação cobra, ou "".
func writtenBoost(pa live.PendingAttack) string {
	if pa.BoostPowerID == "" {
		return ""
	}
	name := pa.BoostPowerID
	if power, known := book.ClassPowers()[pa.BoostPowerID]; known {
		name = power.Name
	}
	return fmt.Sprintf("%s · %d PM ao confirmar", name, pa.BoostPM)
}

// writtenGuard é a penalidade que a investida deixa, ou "".
func writtenGuard(pa live.PendingAttack, attacker string) string {
	penalty := engine.ChargeDefensePenalty(pa.Charge)
	if penalty == 0 {
		return ""
	}
	return fmt.Sprintf("%s: –%d na Defesa até a próxima vez", attacker, -penalty)
}

// writtenSituations são as linhas da Tabela 5-3 que valeram, em prosa.
//
// FORA DA CONTA, e isso foi o OLHO a 390px que decidiu. Dentro dela o parêntese
// ficava em monoespaçado no meio da aritmética e, com três situações, partia a
// linha em três — empurrando o dano, que é o que a mesa veio ler, para o fim da
// terceira. Em prosa ele é ~30% mais estreito, e a quebra cai no lugar natural:
// entre o número e o motivo.
//
// Elas existem pela mesma razão do "20 natural acerta" que a `attackLine`
// nomeia: um número que DESMENTE o que a mesa sabe vem com o motivo. A Defesa
// não é a da ficha quando o alvo está coberto, e a camuflagem desmente a
// própria comparação — o total bateu a Defesa e o ataque errou (p238).
func writtenSituations(pa live.PendingAttack) string {
	return strings.Join(pa.Situations, ", ")
}

// writtenNonLethal qualifica o dano quando ele não mata (p236).
//
// COLADO NO DANO e não na prosa das situações, ao contrário da Tabela 5-3: a
// situação explica a COMPARAÇÃO e esta qualifica o NÚMERO — "9 de dano" e "9 de
// dano não letal" levam a mesas diferentes, e o mestre que não souber vai pedir
// um teste de Constituição que a regra não manda fazer.
//
// Só quando é TUDO: o livro não dá meio-termo — ou a arma é Piedosa e todo o
// dano dela é não letal, ou não é.
func writtenNonLethal(pa live.PendingAttack) string {
	if pa.NonLethal > 0 && pa.NonLethal >= pa.Damage {
		return " " + nonLethalWords
	}
	return ""
}

// nonLethalWords carrega um espaço INQUEBRÁVEL entre as duas palavras, e o olho
// a 390px foi quem pediu: numa linha com RD e situação, a conta quebrava em
// "→ 11 de dano não" / "letal", e "dano não" sozinho no fim de uma linha diz o
// contrário do que a regra diz.
const nonLethalWords = "não letal"

// attackNotes são as linhas do cartão do ataque além da conta: o impulso, as
// situações, a guarda aberta e o que já foi rolado de novo — na ordem em que o
// de hoje as escreve.
func attackNotes(a *attackProposal) []string {
	var notes []string
	for _, line := range []string{a.Boost, a.Situations, a.Guard, a.Rerolled} {
		if line != "" {
			notes = append(notes, line)
		}
	}
	return notes
}

// damageDice escreve os dados do dano com o bônus e cada face rolada:
// "2d8+3 (8+5)".
func damageDice(pa live.PendingAttack) string {
	sum := 0
	rolled := make([]string, 0, len(pa.Dice))
	for _, d := range pa.Dice {
		sum += d
		rolled = append(rolled, fmt.Sprint(d))
	}
	damage := fmt.Sprintf("%dd%d", len(pa.Dice), pa.Faces)
	if bonus := pa.RawDamage - sum; bonus != 0 {
		damage += fmt.Sprintf("%+d", bonus)
	}
	return damage + " (" + strings.Join(rolled, "+") + ")"
}
