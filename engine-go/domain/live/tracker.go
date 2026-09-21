package live

import (
	"errors"
	"fmt"
	"strings"
)

// AS REGRAS DO RASTREADOR ficam no `live` e não no `api` porque falam do estado
// da sessão ao vivo, que é o que este pacote é.

// UpcomingTurns é quem está na vez e quem vem depois, na ORDEM DA MESA.
//
// A lista é CIRCULAR: depois do último vem o primeiro, com a rodada seguinte.
// Cortar no fim deixaria a tira vazia justamente no turno em que saber "quem
// vem depois" mais importa — o último antes de virar a rodada.
//
// Fora de combate (`turn` negativo) não há vez de ninguém e não há fila.
//
// CUIDADO: ela devolve uma JANELA, não a fila. Usá-la para enumerar quem está em
// combate devolve `howMany` de nove combatentes, e uma limpeza baseada nela
// deixa quatro para trás.
func UpcomingTurns(queue []InitiativeEntry, turn, howMany int) []InitiativeEntry {
	if turn < 0 || len(queue) == 0 || howMany <= 0 {
		return nil
	}
	if howMany > len(queue) {
		howMany = len(queue)
	}
	outside := make([]InitiativeEntry, 0, howMany)
	for step := 0; step < howMany; step++ {
		outside = append(outside, queue[(turn+step)%len(queue)])
	}
	return outside
}

// NextTurnTarget é para onde o avanço vai, e como o botão o anuncia.
type NextTurnTarget struct {
	Label string
	Entry *InitiativeEntry
}

// NextTurnButton escreve o rótulo do botão mais clicado da sessão.
//
// Ele diz PARA ONDE vai, e não o que faz: com um "▶" o mestre conta a lista para
// saber quem entra. Fora de combate o verbo muda — "Próximo: Arwen" mentiria
// sobre uma rodada que ainda não começou, e quem clica ali está COMEÇANDO o
// combate.
//
// Fila vazia não tem para onde ir, e prometer um nome seria inventá-lo. O rótulo
// diz o MOTIVO de estar desligado e não o verbo que não vai acontecer: "em cena
// sem ninguém na fila" é o instante em que o mestre acabou de iniciar e vai
// montar a ordem, e ali um "Próximo turno" apagado não explica o que falta.
func NextTurnButton(queue []InitiativeEntry, turn int) NextTurnTarget {
	if len(queue) == 0 {
		return NextTurnTarget{Label: "Ninguém na fila"}
	}
	inCombat := turn >= 0
	index := 0
	verb := "Começar"
	if inCombat {
		index = (turn + 1) % len(queue)
		verb = "Próximo"
	}
	row := queue[index]
	return NextTurnTarget{Label: verb + ": " + row.Label, Entry: &row}
}

// TurnCounter é a frase que diz ONDE a sessão está: fora de cena, em cena
// montando a ordem, ou em que turno de que rodada.
//
// É função e não aninhamento de condicionais porque são QUATRO estados
// exclusivos, e o que decide entre eles é regra — a cena existe antes da fila, e
// a fila existe antes do turno.
//
// "Rodada 0" aparece de propósito no terceiro caso: a rodada só vira 1 no
// primeiro avanço.
func TurnCounter(scene *Scene, round, turn int, inQueue int) string {
	if scene == nil {
		return "Fora de cena"
	}
	// FORA DA CENA DE AÇÃO não há rodada nem vez (p252), então o contador conta
	// outra coisa: QUAL cena é esta. Antes ele recebia um booleano e dizia
	// "Fora de cena" para tudo que não fosse combate — o que era verdade
	// enquanto combate era a única cena que existia.
	if !scene.CountsRounds() {
		return fmt.Sprintf("%s · cena %d", scene.Kind.Name(), scene.Number)
	}
	if inQueue == 0 {
		return "Em cena · ninguém na fila"
	}
	if turn < 0 {
		return fmt.Sprintf("Rodada %d · %d na fila", round, inQueue)
	}
	// O QUE SOBROU DO TURNO entra aqui e não numa segunda tira: a economia de
	// ação (p233) é sobre o turno, e o turno já é o que esta frase conta. Quem
	// vai clicar precisa saber ANTES — a recusa sozinha chega depois do gesto.
	return fmt.Sprintf("Rodada %d · Turno %d/%d · %s", round, turn+1, inQueue, scene.actionsLeft())
}

// ── presença ────────────────────────────────────────────────────────────────

// TableMember é o mínimo que as regras de presença precisam saber. Ele não é o
// DTO do roster: as regras não devem depender da forma que o roster tem hoje.
type TableMember struct {
	CharacterID int64
	// OwnerID é zero quando o membro não tem personagem ligado.
	OwnerID int64
}

// ConnectedCharacters são os personagens de quem está com a aba aberta agora.
//
// Membro SEM personagem não entra — não é que ele esteja offline, é que não há
// personagem para marcar, e um zero na lista viraria "o personagem 0 está
// online" na tela.
func ConnectedCharacters(members []TableMember, present []int64) map[int64]bool {
	online := map[int64]bool{}
	for _, id := range present {
		online[id] = true
	}
	connected := map[int64]bool{}
	for _, m := range members {
		if m.OwnerID != 0 && online[m.OwnerID] {
			connected[m.CharacterID] = true
		}
	}
	return connected
}

// GmSeesVitals: o mestre vê PV de NPC, o jogador não.
//
// A pergunta é "há vitais nesta fila para esconder?", e não "quem é o mestre" —
// numa fila só de PCs não há o que reservar, e a tela não deve mudar de forma
// por causa de um papel que ali não muda nada.
func GmSeesVitals(queue []InitiativeEntry, isGM bool) bool {
	if !isGM {
		return false
	}
	for i := range queue {
		if queue[i].HpMax != nil {
			return true
		}
	}
	return false
}

// ── acrescentar um combatente ────────────────────────────────────────────────

// Os limites do que o mestre pode digitar ao acrescentar um combatente.
//
// Eles moram aqui e não nos atributos do campo, que são UI e não trava: quem
// postasse na mão passaria por cima dos quatro. E moram no `live` e não no app
// porque dois formulários com escadas diferentes deixariam as duas telas
// discordando sobre o que é um combatente aceitável.
//
// A faixa da iniciativa é de jogabilidade e não do livro: um d20 mais bônus cabe
// folgado nela, e o que ela barra é o dedo escorregado que digita 400 e manda o
// combatente para o topo de toda rodada até alguém achar o erro.
const (
	MaxLabelLetters = 60
	MinInitiative   = -5
	MaxInitiative   = 40
	MaxHitPoints    = 999
)

// CombatantDraft é o que o mestre digitou, antes de virar linha.
type CombatantDraft struct {
	Label      string
	Initiative int
	// HP zero é "sem vida registrada", e a linha nasce SEM barra. Um capanga
	// anônimo não precisa de HP, e uma barra 0/0 mentiria dizendo que ele já
	// está morto.
	HP   int64
	Kind string
}

// ValidateCombatantDraft devolve o que impede o combatente de entrar, ou nil.
//
// As mensagens nomeiam o VALOR ofensivo e a forma esperada, porque quem as lê
// está no meio de um combate e precisa consertar sem sair da tela.
func ValidateCombatantDraft(c CombatantDraft) error {
	label := strings.TrimSpace(c.Label)
	if label == "" {
		return errors.New("o combatente precisa de um nome")
	}
	// Conta RUNAS e não bytes: "Ogro Ancião" tem acentos, e um limite em bytes
	// recusaria um nome mais curto do que o que ele deixa passar em ASCII.
	if n := len([]rune(label)); n > MaxLabelLetters {
		return fmt.Errorf("o nome tem %d letras; o limite é %d", n, MaxLabelLetters)
	}
	if err := ValidateInitiative(c.Initiative); err != nil {
		return err
	}
	if c.HP < 0 || c.HP > MaxHitPoints {
		return fmt.Errorf("PV %d está fora da faixa de 0 a %d; 0 é 'sem vida registrada'", c.HP, MaxHitPoints)
	}
	if c.Kind != "npc" && c.Kind != "character" {
		return fmt.Errorf("tipo %q não existe; um combatente é 'npc' ou 'character'", c.Kind)
	}
	return nil
}

// ValidateInitiative é a faixa jogável, e ela vale tanto para o combatente que
// NASCE quanto para o que é CORRIGIDO depois.
//
// Está separada porque tem dois chamadores: acrescentar e editar. Como duas
// constantes copiadas em dois formulários, o que as mantinha iguais era um
// comentário.
func ValidateInitiative(v int) error {
	if v < MinInitiative || v > MaxInitiative {
		return fmt.Errorf("iniciativa %d está fora da faixa de %d a %d", v, MinInitiative, MaxInitiative)
	}
	return nil
}

// UpkeepLine é a SEGUNDA linha da faixa: o que sustentar custou nesta vez, e o
// que acabou por falta de mana (T20 p227).
//
// Vazia quando quem entrou não sustenta nada, e a cena não desenha linha
// nenhuma: uma linha que diz "0 PM" todo turno é ruído que ensina a ignorar a
// faixa, e é justamente nela que a notícia da magia caída aparece.
//
// @example UpkeepLine(&TurnUpkeep{Paid: []string{"Velocidade"}, Cost: 1})
//
//	// "Velocidade · −1 PM"
func UpkeepLine(statement *TurnUpkeep) string {
	if statement == nil {
		return ""
	}
	var parts []string
	if len(statement.Paid) > 0 {
		parts = append(parts, fmt.Sprintf("%s · −%d PM (%d → %d)",
			joinWithAnd(statement.Paid), statement.Cost, statement.MpBefore, statement.MpAfter))
	}
	// A RAZÃO da queda entra na frase: sem mana é uma escolha que acabou;
	// inconsciente é um personagem no chão, e quem lê a mesa precisa saber qual
	// dos dois aconteceu para decidir se vale curar ou reconjurar.
	if len(statement.Dropped) > 0 {
		if statement.Unconscious {
			parts = append(parts, joinWithAnd(statement.Dropped)+" "+
				endedVerb(statement.Dropped)+": inconsciente não sustenta")
		} else {
			for _, fell := range statement.Dropped {
				parts = append(parts, fell+" acabou: sem PM para sustentar")
			}
		}
	}
	return strings.Join(parts, " · ")
}

// joinWithAnd escreve uma lista como uma pessoa a lê: "A, B e C".
func joinWithAnd(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " e " + names[len(names)-1]
}

// endedVerb concorda o verbo com quantas habilidades caíram. Plural fixo sobre
// contagem variável lê "Velocidade acabaram", e isso já foi visto na tela.
func endedVerb(dropped []string) string {
	if len(dropped) == 1 {
		return "acabou"
	}
	return "acabaram"
}
