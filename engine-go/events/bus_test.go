package events

import (
	"sync"
	"testing"
)

// O RECORTE é a regra deste pacote, e é o que se prende primeiro.
//
// Arranjar o resultado por ordem de chamada diria o que acontece COM quem
// recebe, e nada sobre QUEM recebe — que é justamente a decisão aqui. Um
// barramento que entrega a todo mundo passaria por qualquer teste que só
// afirmasse "o evento chegou".
func TestSoQuemTemInteresseRecebe(t *testing.T) {
	var b Bus
	mesa7, para7 := b.Subscribe(OfSession(7))
	defer para7()
	mesa9, para9 := b.Subscribe(OfSession(9))
	defer para9()

	b.Publish(TurnAdvanced{SessionID: 7})

	select {
	case ev := <-mesa7.C:
		if _, ok := ev.(TurnAdvanced); !ok {
			t.Fatalf("a mesa 7 recebeu %T", ev)
		}
	default:
		t.Fatal("a mesa 7 não recebeu o turno da própria sessão")
	}
	select {
	case ev := <-mesa9.C:
		t.Fatalf("a mesa 9 recebeu %T de outra sessão", ev)
	default:
	}
}

// O interesse por PERSONAGEM atravessa a sessão, e é isso que faz a ficha dentro
// da mesa se atualizar quando o dono a edita de OUTRO lugar (ALE-275).
func TestOInteressePorPersonagemNaoDependeDaSessao(t *testing.T) {
	var b Bus
	ficha, parar := b.Subscribe(OfCharacter(14))
	defer parar()

	b.Publish(CharacterChanged{CharacterID: 14})

	select {
	case ev := <-ficha.C:
		if e, ok := ev.(CharacterChanged); !ok || e.CharacterID != 14 {
			t.Fatalf("chegou %#v", ev)
		}
	default:
		t.Fatal("quem escuta o personagem 14 não soube que ele mudou")
	}
}

// DOIS INTERESSES numa assinatura só — é como o stream da Mesa escuta a sessão e
// a própria ficha sem dois canais e dois `case`.
func TestUmaAssinaturaComDoisInteresses(t *testing.T) {
	var b Bus
	sub, parar := b.Subscribe(OfSession(7), OfCharacter(14))
	defer parar()

	b.Publish(TurnAdvanced{SessionID: 7})
	b.Publish(CharacterChanged{CharacterID: 14})
	b.Publish(TurnAdvanced{SessionID: 8}) // de outra mesa: não é dela

	if n := len(sub.C); n != 2 {
		t.Fatalf("%d eventos na fila, esperado 2", n)
	}
}

// O VITAL de quem tem ficha alcança OS DOIS interesses com um evento só.
//
// É o caso que a ALE-275 resolveu com um canal à parte: o mestre fere pela fila,
// a mesa inteira precisa saber, e a ficha daquele jogador também.
func TestOVitalComFichaAlcancaAMesaEAFicha(t *testing.T) {
	var b Bus
	mesa, pararMesa := b.Subscribe(OfSession(7))
	defer pararMesa()
	ficha, pararFicha := b.Subscribe(OfCharacter(14))
	defer pararFicha()

	b.Publish(VitalsChanged{SessionID: 7, EntryID: "a", CharacterID: 14})

	if len(mesa.C) != 1 {
		t.Error("a mesa não soube do dano")
	}
	if len(ficha.C) != 1 {
		t.Error("a ficha do jogador ferido não soube do dano")
	}
}

// NPC não tem ficha, e o `CharacterID` zero não pode casar com ninguém.
//
// Sem esta regra, todo dano em monstro acordaria toda ficha do processo — o
// alvo zero casaria com um interesse zero, e é o tipo de defeito que só aparece
// como tráfego.
func TestOVitalDeNpcNaoAcordaFichaNenhuma(t *testing.T) {
	var b Bus
	ficha, parar := b.Subscribe(OfCharacter(0))
	defer parar()

	b.Publish(VitalsChanged{SessionID: 7, EntryID: "ogro"})

	if n := len(ficha.C); n != 0 {
		t.Fatalf("%d eventos chegaram a um interesse de personagem zero", n)
	}
}

// A FILA CHEIA descarta e CONTA. O contrato é o dos canais que este barramento
// substituiu — o evento é a notícia, a verdade está no store —, mas o descarte
// deixa de ser invisível.
func TestAFilaCheiaDescartaEConta(t *testing.T) {
	var b Bus
	sub, parar := b.Subscribe(OfSession(7))
	defer parar()

	for i := 0; i < queueSize+3; i++ {
		b.Publish(BoardChanged{SessionID: 7})
	}

	if n := len(sub.C); n != queueSize {
		t.Errorf("%d eventos na fila, e ela tem %d lugares", n, queueSize)
	}
	if p := sub.Dropped(); p != 3 {
		t.Errorf("%d descartados, esperado 3 — descarte que não se conta é descarte invisível", p)
	}
}

// O leitor lento NÃO segura quem escreve. É a garantia que os três canais
// originais documentam, e a que não se pode perder na troca.
func TestOLeitorLentoNaoSeguraQuemEscreve(t *testing.T) {
	var b Bus
	_, parar := b.Subscribe(OfSession(7))
	defer parar()

	pronto := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			b.Publish(BoardChanged{SessionID: 7})
		}
		close(pronto)
	}()

	<-pronto // sem ninguém drenando a fila: se `Publish` bloqueasse, isto travava
}

// A BAIXA limpa o registro. Sem ela, cada aba fechada deixa um canal para sempre
// e o `Publish` percorre uma lista que só cresce.
func TestABaixaTiraOOuvinte(t *testing.T) {
	var b Bus
	_, parar := b.Subscribe(OfSession(7))
	if n := b.Listeners(); n != 1 {
		t.Fatalf("%d ouvintes depois de assinar, esperado 1", n)
	}
	parar()
	if n := b.Listeners(); n != 0 {
		t.Errorf("%d ouvintes depois da baixa, esperado 0", n)
	}
}

// Publicar de várias goroutines não corrompe a lista. O `-race` é quem mede
// isto; sem ele o caso passa mesmo com a trava errada.
func TestPublicarEAssinarAoMesmoTempo(t *testing.T) {
	var b Bus
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); b.Publish(TurnAdvanced{SessionID: 7}) }()
		go func() { defer wg.Done(); _, parar := b.Subscribe(OfSession(7)); parar() }()
	}
	wg.Wait()

	if n := b.Listeners(); n != 0 {
		t.Errorf("%d ouvintes sobraram depois de todas as baixas", n)
	}
}
