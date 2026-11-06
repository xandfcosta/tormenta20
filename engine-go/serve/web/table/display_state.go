package table

import (
	"sync"
	"time"
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
	// changed é um canal por sessão que FECHA a cada mudança e é trocado por
	// um novo: quem espera acorda junto, sem ninguém precisar saber quantos
	// esperam. É o que leva o enquadramento à tela na hora — o batimento de 1s
	// do fluxo seguiria o mestre com um segundo de atraso a cada arrasto.
	changed map[int64]chan struct{}
}

func newTableDisplays() *tableDisplays {
	return &tableDisplays{
		framing:  map[int64]displayFraming{},
		frozen:   map[int64]frozenDisplay{},
		watching: map[int64]int{},
		changed:  map[int64]chan struct{}{},
	}
}

// Changed devolve o canal que fecha na próxima mudança desta sessão.
func (d *tableDisplays) Changed(sessionID int64) <-chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.changedLocked(sessionID)
}

func (d *tableDisplays) changedLocked(sessionID int64) chan struct{} {
	ch, ok := d.changed[sessionID]
	if !ok {
		ch = make(chan struct{})
		d.changed[sessionID] = ch
	}
	return ch
}

func (d *tableDisplays) notifyLocked(sessionID int64) {
	if ch, ok := d.changed[sessionID]; ok {
		close(ch)
		delete(d.changed, sessionID)
	}
}

// Frame guarda o recorte que o mestre está olhando.
func (d *tableDisplays) Frame(sessionID int64, f displayFraming) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.framing[sessionID] = f
	d.notifyLocked(sessionID)
}

func (d *tableDisplays) Framing(sessionID int64) displayFraming {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.framing[sessionID]
}

// Freeze guarda o retrato, e devolve falso quando a tela JÁ estava congelada:
// o segundo clique não tira outro retrato, porque veria o que o mestre acabou
// de mexer escondido.
func (d *tableDisplays) Freeze(sessionID int64, v displayView, at time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.frozen[sessionID]; ok {
		return false
	}
	d.frozen[sessionID] = frozenDisplay{view: v, at: at}
	d.notifyLocked(sessionID)
	return true
}

// Thaw solta a tela, que volta a seguir o tabuleiro de agora.
func (d *tableDisplays) Thaw(sessionID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.frozen, sessionID)
	d.notifyLocked(sessionID)
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
	defer d.mu.Unlock()
	d.watching[sessionID]++
}

func (d *tableDisplays) Leave(sessionID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// APAGA a entrada no zero, como a lente: uma sessão que guardasse um zero
	// por tela que já fechou nunca devolveria a memória.
	if d.watching[sessionID]--; d.watching[sessionID] <= 0 {
		delete(d.watching, sessionID)
	}
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

// Erase esquece a sessão inteira, quando ela é apagada.
func (d *tableDisplays) Erase(sessionID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.framing, sessionID)
	delete(d.frozen, sessionID)
	delete(d.watching, sessionID)
	d.notifyLocked(sessionID)
}
