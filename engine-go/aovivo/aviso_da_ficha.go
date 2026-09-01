package aovivo

import "sync"

// O AVISO DE "esta ficha mudou" (ALE-275).
//
// É o TERCEIRO canal que o stream da Mesa escuta, ao lado do da sessão
// (`SessionStore.Assinar`) e do do tabuleiro (`BoardStore.Assinar`), e ele nasce
// pela mesma razão que o segundo nasceu: sem um canal próprio, a mudança só
// chegava no batimento.
//
// # Por que ele não é um campo do SessionStore
//
// A pergunta é sobre um PERSONAGEM e não sobre uma sessão. O mesmo personagem
// pode estar em duas mesas, e a ficha dele muda por caminhos que não passam pela
// fila nenhuma — o próprio dono mexendo no PV pela ficha, por exemplo. Pendurar
// isso na sessão obrigaria quem avisa a saber em quais mesas o personagem está,
// que é justamente a pergunta que o `characterChanged` já responde do outro
// lado, e a fazer isso dentro da trava do store da sessão.
//
// # Os verbos são os dos irmãos, de propósito
//
// O guia manda escrever identificador em inglês, e o TIPO segue a regra
// (`CharacterWatch`, como `SessionStore` e `BoardStore`). Os métodos não: o
// stream chama os três lado a lado, e `Assinar`, `Assinar` e `Subscribe` na
// mesma dúzia de linhas seria pior do que o desvio — três verbos para um ato só.
type CharacterWatch struct {
	mu       sync.Mutex
	ouvintes map[int64][]chan struct{}
}

// Assinar registra um ouvinte deste personagem e devolve o canal mais a baixa.
//
// Buffer 1 e entrega NÃO BLOQUEANTE, como os irmãos: um leitor lento nunca pode
// segurar uma escrita, e um segundo aviso pendente não diz nada que o primeiro
// já não diga — "mudou" não se acumula.
func (w *CharacterWatch) Assinar(characterID int64) (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	w.mu.Lock()
	if w.ouvintes == nil {
		w.ouvintes = map[int64][]chan struct{}{}
	}
	w.ouvintes[characterID] = append(w.ouvintes[characterID], ch)
	w.mu.Unlock()

	return ch, func() { w.desassinar(characterID, ch) }
}

// Avisar cutuca quem escuta este personagem.
func (w *CharacterWatch) Avisar(characterID int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, ch := range w.ouvintes[characterID] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// desassinar tira o ouvinte da lista. Sem isto, cada aba fechada deixa um canal
// para sempre, e o `Avisar` passa a percorrer uma lista que só cresce.
func (w *CharacterWatch) desassinar(characterID int64, alvo chan struct{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	restantes := w.ouvintes[characterID][:0]
	for _, ch := range w.ouvintes[characterID] {
		if ch != alvo {
			restantes = append(restantes, ch)
		}
	}
	if len(restantes) == 0 {
		delete(w.ouvintes, characterID)
		return
	}
	w.ouvintes[characterID] = restantes
}

// Ouvintes é quantos streams escutam este personagem.
//
// Existe para o teste da BAIXA poder afirmar que o `desassinar` limpou o
// registro — a mesma razão do `SessionStore.Ouvintes`. Sem ele o teste diria
// "não sobrou ouvinte" porque não conseguiu olhar, que é o pior tipo de verde.
func (w *CharacterWatch) Ouvintes(characterID int64) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.ouvintes[characterID])
}
