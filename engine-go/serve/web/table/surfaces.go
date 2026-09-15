package table

import "fmt"

// AS SUPERFÍCIES DO JOGADOR.
//
// Cada superfície ocupa a TELA INTEIRA, e um seletor ancorado no topo troca
// entre elas. É a mesma decisão da forma do mestre, do outro lado da mesa: o que
// está na tela está inteiro.
//
// A FICHA só aparece para QUEM TEM personagem na sessão. O mestre não tem
// "minha ficha" — ele vê a mesa inteira —, e um jogador sem personagem na
// campanha clicaria numa aba vazia.

const (
	superficieDaFicha     = "ficha"
	superficieDaMesa      = "mesa"
	superficieDoTabuleiro = "tabuleiro"
	DefaultOpeningSurface = superficieDaMesa
)

type playerSurface struct {
	ID     string
	Rotulo string
	Icone  string
}

// PlayerSurfaces, na ordem em que aparecem.
//
// A MESA primeiro porque é ela que abre: quem entra na sessão quer saber de quem
// é a vez e quem está em cena, e o tabuleiro pode nem estar aberto.
var PlayerSurfaces = []playerSurface{
	{superficieDaMesa, "Mesa", "Users2"},
	{superficieDoTabuleiro, "Tabuleiro", "LayoutGrid"},
}

// surfaces são as superfícies que ESTE leitor recebe.
//
// A "Ficha" entra na frente das outras — primeiro o que é do jogador, depois o
// que é da mesa —, mas ela não é a que ABRE: quem entra na sessão quer saber de
// quem é a vez, e o `DefaultOpeningSurface` continua na Mesa.
func surfaces(v View) []playerSurface {
	if v.MinhaFicha == nil {
		return PlayerSurfaces
	}
	// "Ficha" e não "Minha ficha": com três superfícies o telefone dá ~124px
	// por botão, e a 390px o rótulo longo sai truncado. A palavra é a do
	// glossário.
	comAFicha := []playerSurface{{superficieDaFicha, "Ficha", "ScrollText"}}
	return append(comAFicha, PlayerSurfaces...)
}

// surface é a condição que mostra uma superfície — e o mesmo teste marca o
// botão dela. Escrita aqui e não no `.templ` porque o id do botão tem de casar
// com o do painel, e dois literais divergem no dia em que alguém renomear um.
func surface(qual string) string {
	return fmt.Sprintf("$surface === %q", qual)
}

// pickSurface liga a pedida. Não desliga ao reclicar, ao contrário do
// trilho de ferramentas do mapa: uma superfície desligada não deixaria nada na
// tela.
func pickSurface(qual string) string {
	return fmt.Sprintf("$surface = %q", qual)
}

// surfaceStyling liga UMA das duas aparências, e nunca deixa as duas.
//
// Os dois lados no `data-class` pela mesma armadilha de CASCATA que a aba do
// editor de bloco documenta: a marca de escolhida mora em `@layer components` e
// as cores do Tailwind são utilidades, que vivem numa camada POSTERIOR — camada
// vence especificidade, e o dourado perderia para o cinza sem nada acusar.
func surfaceStyling(qual string) string {
	return fmt.Sprintf(
		"{'surface-chosen': %s, 'border-grimorio-iron': !(%s), 'text-muted-foreground': !(%s)}",
		surface(qual), surface(qual), surface(qual))
}
