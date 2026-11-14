package table

import (
	"sync"
	"time"

	"t20engine/infra/events"
)

// O QUE A TELA DA MESA LEMBRA, por sessão (ALE-608): o enquadramento que o
// mestre mandou, o retrato de quando ele congelou, e quantas telas estão
// abertas.
//
// É MEMÓRIA DO SERVIDOR, e não sinal do navegador, pela razão da lente: o
// display é OUTRO navegador, e o que o mestre faz na tela dele só chega lá
// passando por aqui. E é efêmera — morre com o processo, como a lente e as
// abas escolhidas —, porque é estado de uma noite de jogo, não da campanha.
//
// Uma por SESSÃO e não por pessoa: a mesa tem um mestre, e é o tabuleiro dele
// que o display segue.

// displayFraming é o recorte do mestre em QUADRADOS do plano: o meio dele e o
// tamanho. Em quadrados, e não em pixels, porque o display tem outro tamanho:
// quem encaixa o recorte na tela é ela (`displayFits`).
type displayFraming struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// displayView é o que a tela da mesa desenha: o tabuleiro como a mesa o vê e
// o recorte. Congelada, ela é o retrato inteiro, os dois juntos.
type displayView struct {
	Board   BoardView
	Framing displayFraming
}

// displayState é o que o tabuleiro do MESTRE sabe da tela da mesa: se há uma
// aberta, e se está congelada desde quando.
type displayState struct {
	Watching bool
	Frozen   bool
	FrozenAt string
}

type frozenDisplay struct {
	view displayView
	at   time.Time
}

type tableDisplays struct {
	mu       sync.Mutex
	framing  map[int64]displayFraming
	frozen   map[int64]frozenDisplay
	watching map[int64]int
	// bus leva cada mudança a quem a desenha: a tela da mesa segue o recorte do
	// mestre a cada arrasto, e o tabuleiro dele acende o "ao vivo" quando uma
	// tela abre. Publicar acontece FORA da trava, senão quem acorda para ler
	// espera o escritor.
	bus *events.Bus
}

func newTableDisplays(bus *events.Bus) *tableDisplays {
	return &tableDisplays{
		framing:  map[int64]displayFraming{},
		frozen:   map[int64]frozenDisplay{},
		watching: map[int64]int{},
		bus:      bus,
	}
}

// Frame guarda o recorte que o mestre está olhando.
func (d *tableDisplays) Frame(sessionID int64, f displayFraming) {
	d.mu.Lock()
	d.framing[sessionID] = f
	d.mu.Unlock()
	d.bus.Publish(events.TableDisplayFramed{SessionID: sessionID})
}

func (d *tableDisplays) Framing(sessionID int64) displayFraming {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.framing[sessionID]
}

// Freeze guarda o retrato, e não faz nada com a tela JÁ congelada: o segundo
// clique não tira outro retrato, porque veria o que o mestre acabou de mexer
// escondido. O `freezeDisplay` pergunta antes só para não montar o tabuleiro à
// toa; a garantia é esta, debaixo da trava.
func (d *tableDisplays) Freeze(sessionID int64, v displayView, at time.Time) {
	d.mu.Lock()
	_, already := d.frozen[sessionID]
	if !already {
		d.frozen[sessionID] = frozenDisplay{view: v, at: at}
	}
	d.mu.Unlock()
	if !already {
		d.bus.Publish(events.TableDisplayFrozen{SessionID: sessionID})
	}
}

// Thaw solta a tela, que volta a seguir o tabuleiro de agora.
func (d *tableDisplays) Thaw(sessionID int64) {
	d.mu.Lock()
	delete(d.frozen, sessionID)
	d.mu.Unlock()
	d.bus.Publish(events.TableDisplayThawed{SessionID: sessionID})
}

// Frozen devolve o retrato, se a tela está congelada.
func (d *tableDisplays) Frozen(sessionID int64) (displayView, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	f, ok := d.frozen[sessionID]
	return f.view, ok
}

// Join e Leave contam as telas abertas. Quem chama é o fluxo da tela, que é o
// único que sabe quando ela fecha — o `r.Context()` cancelado.
func (d *tableDisplays) Join(sessionID int64) {
	d.mu.Lock()
	d.watching[sessionID]++
	d.mu.Unlock()
	d.bus.Publish(events.TableDisplayOpened{SessionID: sessionID})
}

func (d *tableDisplays) Leave(sessionID int64) {
	d.mu.Lock()
	// APAGA a entrada no zero, como a lente: uma sessão que guardasse um zero
	// por tela que já fechou nunca devolveria a memória.
	if d.watching[sessionID]--; d.watching[sessionID] <= 0 {
		delete(d.watching, sessionID)
	}
	d.mu.Unlock()
	d.bus.Publish(events.TableDisplayClosed{SessionID: sessionID})
}

// StateOf é o que o interruptor do mestre desenha.
func (d *tableDisplays) StateOf(sessionID int64) displayState {
	d.mu.Lock()
	defer d.mu.Unlock()
	st := displayState{Watching: d.watching[sessionID] > 0}
	if f, ok := d.frozen[sessionID]; ok {
		st.Frozen, st.FrozenAt = true, f.at.Format("15:04")
	}
	return st
}

// HowMany conta o que a tela desta sessão lembra — o recorte, o retrato e as
// telas abertas —, para o `Watching` ter denominador.
func (d *tableDisplays) HowMany(sessionID int64) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	if _, ok := d.framing[sessionID]; ok {
		n++
	}
	if _, ok := d.frozen[sessionID]; ok {
		n++
	}
	return n + d.watching[sessionID]
}

// Erase esquece a sessão inteira, quando ela é apagada.
func (d *tableDisplays) Erase(sessionID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.framing, sessionID)
	delete(d.frozen, sessionID)
	delete(d.watching, sessionID)
}
