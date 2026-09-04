package aovivo

import (
	"errors"
	"fmt"
	"strings"
)

// AS REGRAS DO RASTREADOR (ALE-265).
//
// Elas vinham de `frontend/src/features/session-tracker/tracker-rules.ts`, e a
// razão de estarem lá era a mesma das dez regras que esta migração já
// desentranhou: foi onde o componente precisou delas. A diferença é que aquelas
// estavam soldadas ao TRANSPORTE (o gateway do socket) e estas estão soldadas à
// TELA — mas o efeito é o mesmo, e o segundo consumidor não as alcança.
//
// Ficam no `aovivo` e não no `api` porque falam do estado da sessão ao vivo, que
// é o que este pacote é. A cópia da SPA fica enquanto a tela dela existir; as
// duas convivem durante a migração e some com a virada do rastreador.
//
// O que NÃO veio junto foi o `palcoBaixo`. Ver o comentário no fim.

// UpcomingTurns é quem está na vez e quem vem depois, na ORDEM DA MESA (ALE-179).
//
// A lista é CIRCULAR: depois do último vem o primeiro, com a rodada seguinte.
// Cortar no fim deixaria a tira vazia justamente no turno em que saber "quem
// vem depois" mais importa — o último antes de virar a rodada.
//
// Fora de combate (`turno` negativo) não há vez de ninguém e não há fila.
//
// CUIDADO, e o comentário da SPA registra isto: ela devolve uma JANELA, não a
// fila. Usá-la para enumerar quem está em combate devolve `quantos` de nove
// combatentes, e uma limpeza baseada nela deixa quatro para trás (ALE-211).
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

// NextTurnButton escreve o rótulo do botão mais clicado da sessão (ALE-184).
//
// Ele diz PARA ONDE vai, e não o que faz: o mestre lia "▶" e contava a lista
// para saber quem entrava. Fora de combate o verbo muda — "Próximo: Arwen"
// mentiria sobre uma rodada que ainda não começou, e quem clica ali está
// COMEÇANDO o combate.
//
// Fila vazia não tem para onde ir, e prometer um nome seria inventá-lo. O
// rótulo diz o MOTIVO de estar desligado e não o verbo que não vai acontecer:
// desde a ALE-210 esta vaga só existe DENTRO da cena, e "em cena sem ninguém na
// fila" é o instante em que o mestre acabou de iniciar e vai montar a ordem —
// ali um "Próximo turno" apagado não explica o que falta fazer.
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
// montando a ordem, ou em que turno de que rodada (ALE-210).
//
// É função e não aninhamento de condicionais porque são QUATRO estados
// exclusivos, e o que decide entre eles é regra — a cena existe antes da fila, e
// a fila existe antes do turno.
//
// "Rodada 0" aparece de propósito no terceiro caso: é o que a faixa já dizia
// antes daquela issue, e a rodada só vira 1 no primeiro avanço.
func TurnCounter(cenaAtiva bool, rodada, turno int, naFila int) string {
	if !cenaAtiva {
		return "Fora de cena"
	}
	if naFila == 0 {
		return "Em cena · ninguém na fila"
	}
	if turno < 0 {
		return fmt.Sprintf("Rodada %d · %d na fila", rodada, naFila)
	}
	return fmt.Sprintf("Rodada %d · Turno %d/%d", rodada, turno+1, naFila)
}

// ── presença ────────────────────────────────────────────────────────────────

// TableMember é o mínimo que as regras de presença precisam saber. Ele não é o
// DTO do roster: as regras não devem depender da forma que o roster tem hoje.
type TableMember struct {
	CharacterID int64
	// OwnerID é zero quando o membro não tem personagem ligado.
	OwnerID int64
}

// Aqui morava o `MyCharacters`, que montava "quais personagens são de quem está
// olhando" a partir de uma lista de membros já carregada (ALE-289).
//
// Ele ficou sem chamador de produção: quem responde essa pergunta hoje é o
// `tableRoster` da cena da Mesa, que a resolve contra o BANCO enquanto monta o
// elenco — outra entrada, mesmo conceito.
//
// A regra da ALE-33 não saiu com ele. Ela continua presa, e num lugar melhor:
// `TestTheReachOnlyShowsWhenThereIsABudget` a exercita pelo caminho inteiro,
// e foi ele que acusou o defeito que a docstring do `reachAndTarget` registra —
// o jogador na vez dele sem ver alcance nenhum, porque a POSSE é por peça e o
// caminho de leitura não a resolvia.

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

// O `palcoBaixo` da SPA NÃO foi portado, e isso é decisão e não esquecimento.
//
// Ele responde "o palco comporta a faixa de turno em duas fileiras?" a partir da
// ALTURA MEDIDA do palco — 416px, medidos a 844×390 com uma ficha aberta, onde
// o cromo comia 65% da tela. É pergunta de LEIAUTE, e no servidor ela não tem
// resposta: o Go não mede caixa.
//
// A tradução certa é consulta de CONTÊINER no CSS, como o painel do bestiário
// (`.mesa-painel`, ALE-172) — e ela é melhor que o original, porque a SPA
// precisava de JS só por causa da lista virtualizada. O comentário de lá avisa
// que a consulta tem de ser por ALTURA e não por mídia, porque o teclado virtual
// mexe na altura da JANELA (ALE-176) enquanto a altura do PALCO é outra coisa.

// ── acrescentar um combatente ────────────────────────────────────────────────

// Os limites do que o mestre pode digitar ao acrescentar um combatente.
//
// Eles vinham do formulário da SPA (`AddCombatantForm`), escritos como atributos
// dos campos — que é UI e não trava: quem postasse na mão passava por cima dos
// quatro. Vêm para cá pelo mesmo motivo das outras seis regras desta fatia, e a
// escolha de virem para o `aovivo` em vez de ficarem no piloto é a que evita o
// defeito clássico: dois formulários com escadas diferentes deixariam as duas
// telas discordando sobre o que é um combatente aceitável.
//
// Os números são os que a SPA já praticava. A faixa da iniciativa é de
// jogabilidade e não do livro: um d20 mais bônus cabe folgado nela, e o que ela
// barra é o dedo escorregado que digita 400 e manda o combatente para o topo de
// toda rodada até alguém achar o erro.
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
// Está separada porque tem dois chamadores: acrescentar e editar. Na SPA ela era
// duas constantes copiadas em dois componentes (`AddCombatantForm` e
// `InitiativeEditDialog`), com um comentário em cada dizendo "a mesma do
// formulário de adicionar" — duas cópias que só um comentário mantinha juntas.
func ValidateInitiative(v int) error {
	if v < MinInitiative || v > MaxInitiative {
		return fmt.Errorf("iniciativa %d está fora da faixa de %d a %d", v, MinInitiative, MaxInitiative)
	}
	return nil
}
