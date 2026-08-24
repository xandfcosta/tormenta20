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

// TurnosAVista é quem está na vez e quem vem depois, na ORDEM DA MESA (ALE-179).
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
func TurnosAVista(fila []InitiativeEntry, turno, quantos int) []InitiativeEntry {
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

// AlvoDoProximoTurno é para onde o avanço vai, e como o botão o anuncia.
type AlvoDoProximoTurno struct {
	Rotulo string
	Linha  *InitiativeEntry
}

// ProximoTurno escreve o rótulo do botão mais clicado da sessão (ALE-184).
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
func ProximoTurno(fila []InitiativeEntry, turno int) AlvoDoProximoTurno {
	if len(fila) == 0 {
		return AlvoDoProximoTurno{Rotulo: "Ninguém na fila"}
	}
	emCombate := turno >= 0
	indice := 0
	verbo := "Começar"
	if emCombate {
		indice = (turno + 1) % len(fila)
		verbo = "Próximo"
	}
	linha := fila[indice]
	return AlvoDoProximoTurno{Rotulo: verbo + ": " + linha.Label, Linha: &linha}
}

// ContadorDoTurno é a frase que diz ONDE a sessão está: fora de cena, em cena
// montando a ordem, ou em que turno de que rodada (ALE-210).
//
// É função e não aninhamento de condicionais porque são QUATRO estados
// exclusivos, e o que decide entre eles é regra — a cena existe antes da fila, e
// a fila existe antes do turno.
//
// "Rodada 0" aparece de propósito no terceiro caso: é o que a faixa já dizia
// antes daquela issue, e a rodada só vira 1 no primeiro avanço.
func ContadorDoTurno(cenaAtiva bool, rodada, turno int, naFila int) string {
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

// MembroDaMesa é o mínimo que as regras de presença precisam saber. Ele não é o
// DTO do roster: as regras não devem depender da forma que o roster tem hoje.
type MembroDaMesa struct {
	CharacterID int64
	// DonoID é zero quando o membro não tem personagem ligado.
	DonoID int64
}

// MeusPersonagens são os personagens de quem está olhando.
//
// A ponte é o DONO do personagem e não o id dele: a ficha de um membro é o
// SNAPSHOT da campanha (ALE-33), então o dono registrado é o único fio de volta
// até a pessoa.
func MeusPersonagens(membros []MembroDaMesa, usuarioID int64) map[int64]bool {
	meus := map[int64]bool{}
	if usuarioID == 0 {
		return meus
	}
	for _, m := range membros {
		if m.DonoID == usuarioID {
			meus[m.CharacterID] = true
		}
	}
	return meus
}

// PersonagensConectados são os personagens de quem está com a aba aberta agora.
//
// Membro SEM personagem não entra — não é que ele esteja offline, é que não há
// personagem para marcar, e um zero na lista viraria "o personagem 0 está
// online" na tela.
func PersonagensConectados(membros []MembroDaMesa, presentes []int64) map[int64]bool {
	online := map[int64]bool{}
	for _, id := range presentes {
		online[id] = true
	}
	conectados := map[int64]bool{}
	for _, m := range membros {
		if m.DonoID != 0 && online[m.DonoID] {
			conectados[m.CharacterID] = true
		}
	}
	return conectados
}

// OMestreVeOsVitais: o mestre vê PV de NPC, o jogador não.
//
// A pergunta é "há vitais nesta fila para esconder?", e não "quem é o mestre" —
// numa fila só de PCs não há o que reservar, e a tela não deve mudar de forma
// por causa de um papel que ali não muda nada.
func OMestreVeOsVitais(fila []InitiativeEntry, ehMestre bool) bool {
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
	MaxLetrasDoRotulo = 60
	MinIniciativa     = -5
	MaxIniciativa     = 40
	MaxPontosDeVida   = 999
)

// CombatenteNovo é o que o mestre digitou, antes de virar linha.
type CombatenteNovo struct {
	Rotulo     string
	Iniciativa int
	// PV zero é "sem vida registrada", e a linha nasce SEM barra. Um capanga
	// anônimo não precisa de PV, e uma barra 0/0 mentiria dizendo que ele já
	// está morto.
	PV   int64
	Tipo string
}

// ValidaCombatenteNovo devolve o que impede o combatente de entrar, ou nil.
//
// As mensagens nomeiam o VALOR ofensivo e a forma esperada, porque quem as lê
// está no meio de um combate e precisa consertar sem sair da tela.
func ValidaCombatenteNovo(c CombatenteNovo) error {
	rotulo := strings.TrimSpace(c.Rotulo)
	if rotulo == "" {
		return errors.New("o combatente precisa de um nome")
	}
	// Conta RUNAS e não bytes: "Ogro Ancião" tem acentos, e um limite em bytes
	// recusaria um nome mais curto do que o que ele deixa passar em ASCII.
	if n := len([]rune(rotulo)); n > MaxLetrasDoRotulo {
		return fmt.Errorf("o nome tem %d letras; o limite é %d", n, MaxLetrasDoRotulo)
	}
	if c.Iniciativa < MinIniciativa || c.Iniciativa > MaxIniciativa {
		return fmt.Errorf("iniciativa %d está fora da faixa de %d a %d", c.Iniciativa, MinIniciativa, MaxIniciativa)
	}
	if c.PV < 0 || c.PV > MaxPontosDeVida {
		return fmt.Errorf("PV %d está fora da faixa de 0 a %d; 0 é 'sem vida registrada'", c.PV, MaxPontosDeVida)
	}
	if c.Tipo != "npc" && c.Tipo != "character" {
		return fmt.Errorf("tipo %q não existe; um combatente é 'npc' ou 'character'", c.Tipo)
	}
	return nil
}
