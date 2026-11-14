package table

import (
	"slices"

	"t20engine/infra/events"
)

// streamSubscription é a assinatura de um fluxo, mantida em dia com o que a tela dele mostra.
//
// Os interesses saem da VIEW — as fichas do Grupo, os combatentes da fila —, e a view muda com a Mesa
// aberta: um herói senta pelo convite, um combatente entra na fila. Assinada uma vez só, na abertura, a
// tela deixaria de ouvir a ficha de quem chegou depois, e sem relógio nenhum para cobrir a falta.
type streamSubscription struct {
	bus       *events.Bus
	interests []events.Interest
	sub       events.Subscription
	stop      func()
}

func subscribeStream(bus *events.Bus, interests []events.Interest) *streamSubscription {
	sub, stop := bus.Subscribe(interests...)
	return &streamSubscription{bus: bus, interests: interests, sub: sub, stop: stop}
}

// woke é o canal do evento que acorda o fluxo.
func (s *streamSubscription) woke() <-chan events.Event { return s.sub.C }

func (s *streamSubscription) close() { s.stop() }

// follow troca a assinatura quando os interesses mudaram, e diz se a velha saiu com evento por ler.
//
// A NOVA entra antes de a velha sair, para nada publicado no meio cair no vão. Mas o evento que chegou à
// velha DEPOIS de o fluxo ler o estado e ANTES da troca está só na fila dela: quem chama redesenha de
// novo quando `follow` devolve verdadeiro, em vez de esperar um próximo evento que pode não vir.
//
// @example if subscription.follow(readerInterests(view)) { /* redesenha de novo */ }
func (s *streamSubscription) follow(interests []events.Interest) (unread bool) {
	if slices.Equal(interests, s.interests) {
		return false
	}
	sub, stop := s.bus.Subscribe(interests...)
	unread = len(s.sub.C) > 0
	s.stop()
	s.interests, s.sub, s.stop = interests, sub, stop
	return unread
}
