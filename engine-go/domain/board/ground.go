package board

// O CHÃO DO LUGAR — a APARÊNCIA da cena (pedra, taverna, floresta, ermo, cripta,
// papel). Ver GLOSSARY.md: não é o TERRENO, que é o que o quadrado FAZ e mora
// no `terrain.go`; um é como a cena se parece, o outro é quanto custa
// atravessá-la.
//
// Ele mora aqui e não no arquivo de uma tela pelo motivo do `terrain.go`:
// domínio compartilhado não mora em maquinário de renderização.

// PlaceGround é uma das aparências que o lugar pode ter. Ver GLOSSARY.md: é o
// CHÃO, e não o terreno difícil — um é como a cena se parece, o outro é quanto
// custa atravessá-la.
type PlaceGround struct {
	ID     string
	Rotulo string
}

// PlaceGrounds é a lista que o mestre escolhe ao abrir uma cena.
//
// Ela vive aqui e não na tela porque o CSS já a tem em `.chao-*`, e uma segunda
// cópia escrita à mão no templ é como nasce a opção que a tela oferece e o CSS
// não sabe pintar. O `api/ground_test.go` amarra esta lista ao CSS: acrescentar
// um chão aqui sem pintá-lo lá derruba o guarda.
//
// O PRIMEIRO é o padrão de quem não escolhe.
var PlaceGrounds = []PlaceGround{
	{"stone", "Pedra"},
	{"tavern", "Taverna"},
	{"forest", "Floresta"},
	{"wilds", "Ermo"},
	{"crypt", "Cripta"},
	{"paper", "Papel"},
}

// DefaultGround é o que o servidor usa quando ninguém escolheu.
func DefaultGround() string { return PlaceGrounds[0].ID }

// KnownGround devolve o chão pedido se ele existe, ou o PADRÃO.
//
// TODO caminho que grava chão passa por ela, e não só o de abrir cena: gravar
// o que chegar do formulário deixa um cliente velho — ou um endereço guardado —
// pôr um chão que a folha não sabe pintar, e o mapa desenha SEM TEXTURA, sem
// erro em lugar nenhum.
//
// O padrão em vez da recusa é a mesma escolha que a cena já fazia: chão é
// APARÊNCIA, e derrubar a criação de um lugar por causa dela seria caro demais
// para o que se perde.
func KnownGround(asked string) string {
	for _, g := range PlaceGrounds {
		if g.ID == asked {
			return asked
		}
	}
	return DefaultGround()
}
