package table

import "fmt"

// AS SUPERFÍCIES DO JOGADOR (ALE-129, portadas na ALE-269).
//
// A cena do jogador era uma COLUNA de 48rem com tudo empilhado — grupo, mapa,
// fila —, que é a forma ANTERIOR à ALE-129: lá o dono já inverteu essa hierarquia
// uma vez, e a medição que fechou a issue foi o tabuleiro cabendo em 8×4
// quadrados dentro do trilho de 22rem.
//
// Agora cada superfície ocupa a TELA INTEIRA e um seletor ancorado no topo troca
// entre elas. É a mesma decisão da forma do mestre, do outro lado da mesa: o que
// está na tela está inteiro.
//
// # A TERCEIRA nasceu quando a ficha nasceu
//
// Por duas fatias havia só "Mesa" e "Tabuleiro", e o comentário que estava aqui
// dizia por quê: a ficha era a última tela da migração e não existia em
// Datastar — decisão do dono, "a aba nasce junto com ela". Ela nasceu na fatia
// 8 e ganhou link na 10a; a aba entra agora, antes de a SPA ser apagada, para a
// migração não tirar da mesa o que ela tinha.
//
// Ela só aparece para QUEM TEM personagem na sessão. O mestre não tem "minha
// ficha" — ele vê a mesa inteira —, e um jogador sem personagem na campanha
// clicaria numa aba vazia.

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
// A MESA primeiro porque é ela que abre (decisão do dono): quem entra na sessão
// quer saber de quem é a vez e quem está em cena, e o tabuleiro pode nem estar
// aberto. Os ÍCONES são os mesmos que a `PlayerSurfaceSwitch` da SPA importa do
// lucide, para quem aprendeu um lado reconhecer o outro.
var PlayerSurfaces = []playerSurface{
	{superficieDaMesa, "Mesa", "Users2"},
	{superficieDoTabuleiro, "Tabuleiro", "LayoutGrid"},
}

// surfaces são as superfícies que ESTE leitor recebe.
//
// A "Minha ficha" entra na frente das outras — é a ordem da SPA, e ela põe
// primeiro o que é do jogador antes do que é da mesa —, mas ela não é a que
// ABRE: quem entra na sessão quer saber de quem é a vez (decisão do dono, e o
// `DefaultOpeningSurface` continua na Mesa).
func surfaces(v View) []playerSurface {
	if v.MinhaFicha == nil {
		return PlayerSurfaces
	}
	// "Ficha" e não "Minha ficha", que é o rótulo da SPA: com três superfícies o
	// telefone dá ~124px por botão, e medido a 390px na bancada saía "MINHA
	// FIC…". O comentário do seletor já contava que a SPA passou por isto — o
	// `flex-1` dela nasceu de três abas truncando —, e encurtar o rótulo é o
	// conserto que não depende da largura. A palavra é a do glossário.
	comAFicha := []playerSurface{{superficieDaFicha, "Ficha", "ScrollText"}}
	return append(comAFicha, PlayerSurfaces...)
}

// surface é a condição que mostra uma superfície — e o mesmo teste marca o
// botão dela. Escrita aqui e não no `.templ` porque o id do botão tem de casar
// com o do painel, e dois literais divergem no dia em que alguém renomear um.
func surface(qual string) string {
	return fmt.Sprintf("$superficie === %q", qual)
}

// pickSurface liga a pedida. Não desliga ao reclicar, ao contrário do
// trilho de ferramentas do mapa: uma superfície desligada não deixaria nada na
// tela.
func pickSurface(qual string) string {
	return fmt.Sprintf("$superficie = %q", qual)
}

// surfaceStyling liga UMA das duas aparências, e nunca deixa as duas.
//
// Os dois lados no `data-class` pela mesma armadilha de CASCATA que a aba do
// editor de bloco documenta: a marca de escolhida mora em `@layer components` e
// as cores do Tailwind são utilidades, que vivem numa camada POSTERIOR — camada
// vence especificidade, e o dourado perderia para o cinza sem nada acusar.
func surfaceStyling(qual string) string {
	return fmt.Sprintf(
		"{'superficie-escolhida': %s, 'border-grimorio-iron': !(%s), 'text-muted-foreground': !(%s)}",
		surface(qual), surface(qual), surface(qual))
}
