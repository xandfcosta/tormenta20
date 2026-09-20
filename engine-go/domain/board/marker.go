package board

import (
	"fmt"
	"strings"
)

// O MARCADOR — o ponto apontado no mapa. Ver GLOSSARY.md: ele nasce ESCONDIDO,
// porque marcar a armadilha na frente da mesa entrega a armadilha.
//
// A LISTA DE CORES mora aqui e em nenhum outro lugar, como a dos chões: uma
// segunda lista escrita à mão na tela não estoura quando discorda desta — ela
// pinta a cor errada em silêncio, e todo marcador cai no padrão.

// BoardMarker é um LUGAR marcado no mapa que não é uma peça: a armadilha, a
// porta que range, o ponto de encontro.
//
// A alternativa seria uma PEÇA `object` dizendo "a armadilha é aqui", e peça
// ocupa quadrado, entra na conta de quem está na área e aparece na lista de
// quem o gabarito pega. O marcador não ocupa nada: ele aponta.
//
// Texto de DUAS letras ("1A", "B3") e cor de um conjunto fechado: o bastante
// para apontar sem exigir asset.
type BoardMarker struct {
	ID    string `json:"id"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Text  string `json:"text"`
	Color string `json:"color"`
	// Hidden: nasce escondido e o mestre revela, pela MESMA redação por papel da
	// peça — uma segunda política sobre o que a mesa vê seria a forma mais fácil
	// de vazar a armadilha.
	Hidden bool `json:"hidden,omitempty"`
}

// boardMaxMarkers — teto de marcadores, pelo mesmo motivo do teto de peças: o
// estado inteiro viaja em todo broadcast.
const boardMaxMarkers = 100

// AddMarker põe um lugar marcado no mapa.
func AddMarker(b *BoardState, m BoardMarker, newID func() string) error {
	if len(b.Markers) >= boardMaxMarkers {
		return fmt.Errorf("o tabuleiro já tem %d marcadores (teto %d)", len(b.Markers), boardMaxMarkers)
	}
	if abs(m.X) > boardCoordLimit || abs(m.Y) > boardCoordLimit {
		return fmt.Errorf("marcador em (%d,%d) está além do limite de sanidade de %d quadrados", m.X, m.Y, boardCoordLimit)
	}
	if !KnownMarkerColor(m.Color) {
		m.Color = DefaultMarkerColor()
	}
	m.Text = trimMarkerText(m.Text)
	m.ID = newID()
	b.Markers = append(b.Markers, m)
	b.Version++
	return nil
}

// trimMarkerText corta o rótulo em DUAS letras — em runas e não em bytes, senão
// "Ê2" viraria meio caractere e a tela desenharia lixo.
func trimMarkerText(text string) string {
	runas := []rune(strings.TrimSpace(text))
	if len(runas) > 2 {
		runas = runas[:2]
	}
	return string(runas)
}

// UpdateMarker altera texto, cor ou o ocultamento — a posição não muda porque
// marcador que anda é peça, e peça já existe.
func UpdateMarker(b *BoardState, markerID string, patch MarkerPatch) error {
	for i := range b.Markers {
		if b.Markers[i].ID != markerID {
			continue
		}
		if patch.Text != nil {
			b.Markers[i].Text = trimMarkerText(*patch.Text)
		}
		if patch.Color != nil && KnownMarkerColor(*patch.Color) {
			b.Markers[i].Color = *patch.Color
		}
		if patch.Hidden != nil {
			b.Markers[i].Hidden = *patch.Hidden
		}
		b.Version++
		return nil
	}
	return fmt.Errorf("marcador %q não está no tabuleiro", markerID)
}

// MarkerPatch é a alteração parcial: ausente é "não mexa", não "zere".
type MarkerPatch struct {
	Text   *string `json:"text"`
	Color  *string `json:"color"`
	Hidden *bool   `json:"hidden"`
}

// RemoveMarker tira o lugar marcado. Some em silêncio se já não está lá.
func RemoveMarker(b *BoardState, markerID string) {
	for i, m := range b.Markers {
		if m.ID != markerID {
			continue
		}
		b.Markers = append(b.Markers[:i], b.Markers[i+1:]...)
		b.Version++
		return
	}
}

// MarkerColor é uma das cores que o mestre pode escolher.
type MarkerColor struct {
	ID     string
	Rotulo string
}

// MarkerColors é o conjunto FECHADO, e fechado não é economia: a cor vira
// `style` na tela, então aceitar string livre deixaria o cliente escrever CSS no
// estado da mesa.
//
// A primeira é o padrão de quem não escolheu.
var MarkerColors = []MarkerColor{
	{"ouro", "Ouro"},
	{"carmim", "Carmim"},
	{"azul", "Azul"},
	{"verde", "Verde"},
}

// DefaultMarkerColor é onde cai quem manda cor que não existe.
func DefaultMarkerColor() string { return MarkerColors[0].ID }

// KnownMarkerColor diz se a cor pedida está na lista.
func KnownMarkerColor(id string) bool {
	for _, c := range MarkerColors {
		if c.ID == id {
			return true
		}
	}
	return false
}

// NextMarkerLetter é a próxima letra livre para um marcador novo.
//
// Quem está apontando a armadilha no meio da cena não quer digitar, e "A", "B",
// "C" é como a mesa fala de lugares num mapa. Esgotadas as letras, cai em "??" —
// que é feio de propósito: com 26 marcadores na tela, o rótulo já não é o que
// distingue nada.
//
// A letra é escolhida pelo SERVIDOR e não pelo cliente: "livre" é pergunta sobre
// o estado do tabuleiro, e duas telas escolhendo por conta nomeariam diferente.
//
//	NextMarkerLetter(b.Markers) // => "C", com A e B já no mapa
func NextMarkerLetter(marcadores []BoardMarker) string {
	usadas := make(map[string]bool, len(marcadores))
	for _, m := range marcadores {
		usadas[m.Text] = true
	}
	for letra := 'A'; letra <= 'Z'; letra++ {
		if !usadas[string(letra)] {
			return string(letra)
		}
	}
	return "??"
}

// ── os patches TIPADOS, para quem não fala JSON ──────────────────────────────
//
// O `ParseMarkerPatch` monta o patch a partir de um `map[string]any`, que é a
// forma de um corpo JSON. Os gestos do app levam a intenção no CAMINHO, e montar
// um mapa só para desmontá-lo em seguida seria atravessar um formato de fio que
// ninguém está falando.

// MarkerReveal monta o patch que mostra ou esconde.
//
// REVELAR é o verbo que importa: o marcador nasce escondido porque marcar a
// armadilha na frente da mesa entrega a armadilha.
func MarkerReveal(escondido bool) MarkerPatch {
	return MarkerPatch{Hidden: &escondido}
}

// NewMarkerColor monta o patch da cor. Cor fora da lista é IGNORADA pelo
// `UpdateMarker`, então o marcador fica com a que tinha — que é melhor do que
// cair no padrão, porque aqui já existe uma escolha anterior a preservar.
func NewMarkerColor(cor string) MarkerPatch {
	return MarkerPatch{Color: &cor}
}
