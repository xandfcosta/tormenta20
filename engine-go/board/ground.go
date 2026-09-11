package board

// O CHÃO DO LUGAR — a APARÊNCIA da cena (pedra, taverna, floresta, ermo, cripta,
// papel). Ver GLOSSARY.md: não é o TERRENO, que é o que o quadrado FAZ e mora
// no `terrain.go`; um é como a cena se parece, o outro é quanto custa
// atravessá-la.
//
// Este arquivo nasceu junto com o `terrain.go`, e pelo MESMO motivo que a sessão
// da main deu ao criar aquele: domínio compartilhado não mora em arquivo de uma
// tela. Isto vivia no fim do `vista.go`, que é maquinário de RENDERIZAÇÃO do
// piloto — o servidor desenhando a moldura porque quem enquadra é o navegador.
//
// A `main` ainda não tem este arquivo: hoje só o piloto consome a lista, e o
// equivalente da SPA é o `TERRAIN_LABEL` em TypeScript. Fica aqui já separado
// para a próxima colheita não repetir a extração — e o nome está avisado do
// outro lado, para as duas branches não inventarem dois arquivos para a mesma
// coisa.

// PlaceGround é uma das aparências que o lugar pode ter. Ver GLOSSARY.md: é o
// CHÃO, e não o terreno difícil — um é como a cena se parece, o outro é quanto
// custa atravessá-la.
type PlaceGround struct {
	ID     string
	Rotulo string
}

// PlaceGrounds é a lista que o mestre escolhe ao abrir uma cena.
//
// Ela vive aqui e não na tela porque JÁ EXISTIA duas vezes — em `.chao-*` no CSS
// do piloto e no `TERRAIN_LABEL` da SPA —, e uma terceira cópia escrita à mão no
// templ é como nasce a opção que a tela oferece e o CSS não sabe pintar. O
// `api/piloto_ground_test.go` amarra esta lista ao CSS: acrescentar um chão aqui sem
// pintá-lo lá derruba o guarda.
//
// A ORDEM é a da SPA, e o primeiro é o padrão de quem não escolhe.
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
// Ela existe porque o caminho de CRIAR lugar gravava o que chegasse do
// formulário, enquanto o de abrir cena já filtrava (ALE-301). Enquanto os ids
// eram os mesmos do formulário da casa a diferença não aparecia; quando eles
// saíram em inglês, um cliente velho — ou um endereço guardado — passou a poder
// gravar um chão que a folha não sabe pintar, e o mapa desenha SEM TEXTURA, sem
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
