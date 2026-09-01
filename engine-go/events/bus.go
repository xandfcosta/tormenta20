// Package events é o vocabulário do que acontece numa mesa, e o barramento que
// o entrega dentro do processo (ALE-279).
//
// Ele nasceu de quatro mecanismos com a mesma forma e nenhum nome em comum: o
// `SessionStore.Assinar`, o `BoardStore.Assinar`, o `CharacterWatch.Assinar` e o
// `SSEHub.Emit`. Os três primeiros eram `chan struct{}` — diziam QUE algo mudou
// e nunca O QUÊ —, e o `select` do stream da Mesa tinha um `case` para cada um,
// só para juntar de volta o que estava separado por acidente de onde o estado
// mora.
//
// # Isto não é event sourcing
//
// O banco continua sendo o estado. O evento diz o que aconteceu para quem
// precisa reagir; ele não é a fonte da verdade, e não se reconstrói ficha a
// partir dele. A razão está na ALE-278: as regras do livro são conta e não
// fluxo, e o `engine/` não é tocado por nada disto.
//
// # Por que tudo aqui é em inglês, inclusive os verbos
//
// A regra da casa manda identificador novo em inglês, e aqui ela vale inteira.
// O `CharacterWatch` teve os métodos em português com uma justificativa
// registrada — o stream chamava os três `Assinar` lado a lado, e três verbos
// para um ato só seria pior que o desvio. Este pacote SUBSTITUI os três, então
// aquela justificativa deixou de existir junto com eles.
package events

import (
	"sync"
	"sync/atomic"
)

// Event is something that happened at a table.
//
// O `Target` é o que permite entregar sem que o barramento conheça nenhum tipo
// concreto: quem publica diz a quem aquilo interessa, e o barramento compara.
type Event interface {
	Target() Target
}

// Target is who an event concerns. Zero num campo significa "não é sobre isso"
// — um turno que passa não é sobre personagem nenhum, e uma ficha salva fora de
// mesa não é sobre sessão nenhuma.
type Target struct {
	SessionID   int64
	CharacterID int64
}

// Interest is what a listener wants to receive. Ver `OfSession` e `OfCharacter`.
type Interest struct {
	session   int64
	character int64
}

// OfSession pede tudo que acontece numa mesa: a fila, a cena, o tabuleiro.
func OfSession(sessionID int64) Interest { return Interest{session: sessionID} }

// OfCharacter pede tudo que acontece com uma ficha, em qualquer mesa.
//
// É por PERSONAGEM e não por sessão porque a pergunta é sobre a ficha: o mesmo
// personagem pode estar em duas mesas, e a ficha dele também muda fora de mesa
// nenhuma — o próprio dono mexendo no PV.
func OfCharacter(characterID int64) Interest { return Interest{character: characterID} }

func (i Interest) matches(t Target) bool {
	if i.session != 0 && i.session == t.SessionID {
		return true
	}
	return i.character != 0 && i.character == t.CharacterID
}

// queueSize é quantos eventos um ouvinte acumula antes de o barramento começar
// a descartar.
//
// Dezesseis, e não um, que era o que os canais originais tinham. A diferença é
// de NATUREZA: `chan struct{}` colapsa — dois "mudou" pendentes não dizem mais
// que um —, e evento tipado não colapsa, porque "o turno passou" e "o combatente
// saiu" são notícias diferentes. Um buffer de um perderia a segunda de todo par
// que chega junto, e par junto é o caso comum: o mestre fere e passa o turno no
// mesmo gesto.
const queueSize = 16

// Bus delivers events to interested listeners, inside the process.
//
// O zero dele já funciona: não existe estado desligado para alguém tolerar, que
// é a lição do gancho `characterChanged` da ALE-253 — campo que outro arquivo
// precisa preencher é recurso que nasce metade desligado, e o Go segue verde.
type Bus struct {
	mu        sync.Mutex
	listeners []*listener
}

type listener struct {
	interests []Interest
	queue     chan Event
	dropped   atomic.Int64
}

// Subscription is what a listener holds: the channel and the count of what was
// dropped.
type Subscription struct {
	C <-chan Event
	l *listener
}

// Dropped is how many events were discarded because the queue was full.
//
// Existe pela mesma razão que o medidor de contraste devolve o DENOMINADOR:
// "não perdi nada" e "não medi nada" se parecem no terminal. Um ouvinte que
// descarta em silêncio e um ouvinte em dia são iguais vistos de fora, e é
// justamente na fila cheia que a tela para de acompanhar a mesa.
func (s Subscription) Dropped() int64 { return s.l.dropped.Load() }

// Subscribe registers a listener and returns the subscription plus its cancel.
//
// Sem interesse nenhum o ouvinte não recebe nada, e isso é deliberado: escutar
// TUDO é o pedido que ninguém consegue justificar, e deixá-lo fácil é como nasce
// o ouvinte que acorda a cada tecla digitada em qualquer mesa da casa.
func (b *Bus) Subscribe(interests ...Interest) (Subscription, func()) {
	l := &listener{interests: interests, queue: make(chan Event, queueSize)}
	b.mu.Lock()
	b.listeners = append(b.listeners, l)
	b.mu.Unlock()

	return Subscription{C: l.queue, l: l}, func() { b.unsubscribe(l) }
}

// Publish delivers the event to whoever cares about it.
//
// A entrega é NÃO BLOQUEANTE, como nos três canais que este barramento
// substituiu: um leitor lento nunca pode segurar uma escrita. O que muda é o que
// acontece com a fila cheia — ali o evento é DESCARTADO e contado, e o contrato
// com quem escuta continua o de sempre: o evento é a notícia, a verdade está no
// store. Quem perdeu um evento relê e continua correto; quem quiser confiar na
// sequência olha `Dropped`.
//
// # Por que ela poderia ser chamada de dentro da trava de um store
//
// O barramento é FOLHA na ordem de travas: pega só o `b.mu`, não chama de volta
// nenhum store, e não executa código de quem escuta — o `select` com `default`
// nunca cede o processador para o ouvinte. O abraço mortal que o
// `tabuleiro/aviso.go` documenta é entre DOIS STORES com travas próprias
// chamando um ao outro, e não é este caso.
//
// Mesmo assim os stores publicam DEPOIS de soltar a trava, e a razão é outra:
// agora quem acorda sabe o que houve e pode ler o estado na hora. Publicar ainda
// sob a trava faria esse leitor esperar pelo escritor no exato instante em que
// ele foi acordado para ler.
func (b *Bus) Publish(ev Event) {
	target := ev.Target()
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, l := range b.listeners {
		if !wants(l.interests, target) {
			continue
		}
		select {
		case l.queue <- ev:
		default:
			l.dropped.Add(1)
		}
	}
}

func wants(interests []Interest, target Target) bool {
	for _, i := range interests {
		if i.matches(target) {
			return true
		}
	}
	return false
}

// unsubscribe tira o ouvinte da lista. Sem isto, cada aba fechada deixa um canal
// para sempre e o `Publish` percorre uma lista que só cresce.
func (b *Bus) unsubscribe(target *listener) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := b.listeners[:0]
	for _, l := range b.listeners {
		if l != target {
			remaining = append(remaining, l)
		}
	}
	b.listeners = remaining
}

// Listeners is how many are subscribed.
//
// Existe para o teste da BAIXA poder afirmar que o `unsubscribe` limpou o
// registro — a mesma razão do `SessionStore.Ouvintes`. Sem ele o teste diria
// "não sobrou ouvinte" porque não conseguiu olhar, que é o pior tipo de verde.
func (b *Bus) Listeners() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.listeners)
}
