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
// Fora de combate (`turno` negativo) não há vez de ninguém e não há fila.
//
// CUIDADO: ela devolve uma JANELA, não a fila. Usá-la para enumerar quem está em
// combate devolve `quantos` de nove combatentes, e uma limpeza baseada nela
// deixa quatro para trás.
func UpcomingTurns(fila []InitiativeEntry, turno, quantos int) []InitiativeEntry {
	if turno < 0 || len(fila) == 0 || quantos <= 0 {
		return nil
	}
	if quantos > len(fila) {
		quantos = len(fila)
	}
	fora := make([]InitiativeEntry, 0, quantos)
	for passo := 0; passo < quantos; passo++ {
		fora = append(fora, fila[(turno+passo)%len(fila)])
	}
	return fora
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
func NextTurnButton(fila []InitiativeEntry, turno int) NextTurnTarget {
	if len(fila) == 0 {
		return NextTurnTarget{Label: "Ninguém na fila"}
	}
	emCombate := turno >= 0
	indice := 0
	verbo := "Começar"
	if emCombate {
		indice = (turno + 1) % len(fila)
		verbo = "Próximo"
	}
	linha := fila[indice]
	return NextTurnTarget{Label: verbo + ": " + linha.Label, Entry: &linha}
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
func TurnCounter(cena *Scene, rodada, turno int, naFila int) string {
	if cena == nil {
		return "Fora de cena"
	}
	// FORA DA CENA DE AÇÃO não há rodada nem vez (p252), então o contador conta
	// outra coisa: QUAL cena é esta. Antes ele recebia um booleano e dizia
	// "Fora de cena" para tudo que não fosse combate — o que era verdade
	// enquanto combate era a única cena que existia.
	if !cena.CountsRounds() {
		return fmt.Sprintf("%s · cena %d", cena.Kind.Name(), cena.Number)
	}
	if naFila == 0 {
		return "Em cena · ninguém na fila"
	}
	if turno < 0 {
		return fmt.Sprintf("Rodada %d · %d na fila", rodada, naFila)
	}
	// O QUE SOBROU DO TURNO entra aqui e não numa segunda tira: a economia de
	// ação (p233) é sobre o turno, e o turno já é o que esta frase conta. Quem
	// vai clicar precisa saber ANTES — a recusa sozinha chega depois do gesto.
	return fmt.Sprintf("Rodada %d · Turno %d/%d · %s", rodada, turno+1, naFila, cena.actionsLeft())
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
func ConnectedCharacters(membros []TableMember, presentes []int64) map[int64]bool {
	online := map[int64]bool{}
	for _, id := range presentes {
		online[id] = true
	}
	conectados := map[int64]bool{}
	for _, m := range membros {
		if m.OwnerID != 0 && online[m.OwnerID] {
			conectados[m.CharacterID] = true
		}
	}
	return conectados
}

// GmSeesVitals: o mestre vê PV de NPC, o jogador não.
//
// A pergunta é "há vitais nesta fila para esconder?", e não "quem é o mestre" —
// numa fila só de PCs não há o que reservar, e a tela não deve mudar de forma
// por causa de um papel que ali não muda nada.
func GmSeesVitals(fila []InitiativeEntry, ehMestre bool) bool {
	if !ehMestre {
		return false
	}
	for i := range fila {
		if fila[i].HpMax != nil {
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
	rotulo := strings.TrimSpace(c.Label)
	if rotulo == "" {
		return errors.New("o combatente precisa de um nome")
	}
	// Conta RUNAS e não bytes: "Ogro Ancião" tem acentos, e um limite em bytes
	// recusaria um nome mais curto do que o que ele deixa passar em ASCII.
	if n := len([]rune(rotulo)); n > MaxLabelLetters {
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
func UpkeepLine(extrato *TurnUpkeep) string {
	if extrato == nil {
		return ""
	}
	var partes []string
	if len(extrato.Paid) > 0 {
		partes = append(partes, fmt.Sprintf("%s · −%d PM (%d → %d)",
			joinWithAnd(extrato.Paid), extrato.Cost, extrato.MpBefore, extrato.MpAfter))
	}
	for _, caiu := range extrato.Dropped {
		partes = append(partes, caiu+" acabou: sem PM para sustentar")
	}
	return strings.Join(partes, " · ")
}

// joinWithAnd escreve uma lista como uma pessoa a lê: "A, B e C".
func joinWithAnd(nomes []string) string {
	if len(nomes) < 2 {
		return strings.Join(nomes, "")
	}
	return strings.Join(nomes[:len(nomes)-1], ", ") + " e " + nomes[len(nomes)-1]
}
