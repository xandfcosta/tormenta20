package board

// O MARCADOR — o ponto apontado no mapa. Ver GLOSSARY.md: ele nasce ESCONDIDO,
// porque marcar a armadilha na frente da mesa entrega a armadilha.
//
// A LISTA DE CORES mora aqui e em nenhum outro lugar, como a dos chões: uma
// segunda lista escrita à mão na tela não estoura quando discorda desta — ela
// pinta a cor errada em silêncio, e todo marcador cai no padrão.

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
