package tabuleiro

// O CHÃO DO LUGAR — a APARÊNCIA da cena (pedra, taverna, floresta, ermo, cripta,
// papel). Ver GLOSSARIO.md: não é o TERRENO, que é o que o quadrado FAZ e mora
// no `terreno.go`; um é como a cena se parece, o outro é quanto custa
// atravessá-la.
//
// Este arquivo nasceu junto com o `terreno.go`, e pelo MESMO motivo que a sessão
// da main deu ao criar aquele: domínio compartilhado não mora em arquivo de uma
// tela. Isto vivia no fim do `vista.go`, que é maquinário de RENDERIZAÇÃO do
// piloto — o servidor desenhando a moldura porque quem enquadra é o navegador.
//
// A `main` ainda não tem este arquivo: hoje só o piloto consome a lista, e o
// equivalente da SPA é o `TERRAIN_LABEL` em TypeScript. Fica aqui já separado
// para a próxima colheita não repetir a extração — e o nome está avisado do
// outro lado, para as duas branches não inventarem dois arquivos para a mesma
// coisa.

// PlaceGround é uma das aparências que o lugar pode ter. Ver GLOSSARIO.md: é o
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
// `chao_do_lugar_test.go` amarra esta lista ao CSS: acrescentar um chão aqui sem
// pintá-lo lá derruba o guarda.
//
// A ORDEM é a da SPA, e o primeiro é o padrão de quem não escolhe.
var PlaceGrounds = []PlaceGround{
	{"pedra", "Pedra"},
	{"taverna", "Taverna"},
	{"floresta", "Floresta"},
	{"ermo", "Ermo"},
	{"cripta", "Cripta"},
	{"papel", "Papel"},
}

// DefaultGround é o que o servidor usa quando ninguém escolheu.
func DefaultGround() string { return PlaceGrounds[0].ID }
