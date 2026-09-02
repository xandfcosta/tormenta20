package tabuleiro

// O MARCADOR — o ponto apontado no mapa (ALE-195). Ver GLOSSARIO.md: ele nasce
// ESCONDIDO, porque marcar a armadilha na frente da mesa entrega a armadilha.
//
// Este arquivo nasceu do mesmo defeito que criou o `ground.go`, e desta vez ele
// tinha chegado à tela: a lista de cores existia DUAS vezes e as duas discordavam
// — a autoridade (`AddMarker`) aceita `ouro/carmim/azul/verde` em pt-BR, e o
// piloto tinha escrito `gold/red/green/blue/violet` à mão, no view e no CSS.
// Nenhuma das cinco casa com nenhuma das quatro, então TODO marcador do piloto
// caía no dourado — inclusive o carmim que o mestre escolheu na outra tela.
//
// O defeito é da família que este repositório persegue: ele não estoura, ele
// pinta a cor errada em silêncio. E a lição contra ele já estava escrita a 150
// linhas do erro, no comentário da lista de chões.

// MarkerColor é uma das cores que o mestre pode escolher.
type MarkerColor struct {
	ID     string
	Rotulo string
}

// MarkerColors é o conjunto FECHADO, e fechado não é economia: a cor vira
// `style` na tela, então aceitar string livre deixaria o cliente escrever CSS no
// estado da mesa.
//
// A ORDEM é a da SPA, e a primeira é o padrão de quem não escolheu.
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

// NextMarkerLetter é a próxima letra livre para um marcador novo (ALE-195).
//
// Quem está apontando a armadilha no meio da cena não quer digitar, e "A", "B",
// "C" é como a mesa fala de lugares num mapa. Esgotadas as letras, cai em "??" —
// que é feio de propósito: com 26 marcadores na tela, o rótulo já não é o que
// distingue nada.
//
// A regra vivia só na SPA (`nextMarkerText`), onde o CLIENTE escolhia a letra e
// mandava pronta. Trazê-la para cá é o que faz as duas telas nomearem igual — e
// é onde ela pertence, porque "livre" é pergunta sobre o estado do tabuleiro.
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
// forma do corpo JSON da SPA. O piloto não tem esse mapa — os gestos dele levam
// a intenção no CAMINHO —, e montar um mapa só para desmontá-lo em seguida seria
// atravessar o formato de fio de uma tela para chegar ao domínio da outra.

// MarkerReveal monta o patch que mostra ou esconde.
//
// REVELAR é o verbo que importa: o marcador nasce escondido porque marcar a
// armadilha na frente da mesa entrega a armadilha (ALE-195).
func MarkerReveal(escondido bool) markerPatch {
	return markerPatch{Hidden: &escondido}
}

// NewMarkerColor monta o patch da cor. Cor fora da lista é IGNORADA pelo
// `UpdateMarker`, então o marcador fica com a que tinha — que é melhor do que
// cair no padrão, porque aqui já existe uma escolha anterior a preservar.
func NewMarkerColor(cor string) markerPatch {
	return markerPatch{Color: &cor}
}
