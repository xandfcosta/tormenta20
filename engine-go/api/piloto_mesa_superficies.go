package api

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
// # Por que DUAS e não três
//
// A SPA tem "Minha ficha", "Mesa" e "Tabuleiro". A FICHA é a última tela da
// migração e não existe em Datastar — decisão do dono: a aba nasce junto com ela.
// Uma terceira aba hoje prometeria na tela o que o app não tem, e o jogador
// clicaria nela procurando a própria ficha.

const (
	superficieDaMesa        = "mesa"
	superficieDoTabuleiro   = "tabuleiro"
	superficieQueAbrePadrao = superficieDaMesa
)

type superficieDoJogador struct {
	ID     string
	Rotulo string
	Icone  string
}

// asSuperficiesDoJogador, na ordem em que aparecem.
//
// A MESA primeiro porque é ela que abre (decisão do dono): quem entra na sessão
// quer saber de quem é a vez e quem está em cena, e o tabuleiro pode nem estar
// aberto. Os ÍCONES são os mesmos que a `PlayerSurfaceSwitch` da SPA importa do
// lucide, para quem aprendeu um lado reconhecer o outro.
var asSuperficiesDoJogador = []superficieDoJogador{
	{superficieDaMesa, "Mesa", "Users2"},
	{superficieDoTabuleiro, "Tabuleiro", "LayoutGrid"},
}

// naSuperficie é a condição que mostra uma superfície — e o mesmo teste marca o
// botão dela. Escrita aqui e não no `.templ` porque o id do botão tem de casar
// com o do painel, e dois literais divergem no dia em que alguém renomear um.
func naSuperficie(qual string) string {
	return fmt.Sprintf("$superficie === %q", qual)
}

// escolheASuperficie liga a pedida. Não desliga ao reclicar, ao contrário do
// trilho de ferramentas do mapa: uma superfície desligada não deixaria nada na
// tela.
func escolheASuperficie(qual string) string {
	return fmt.Sprintf("$superficie = %q", qual)
}

// oVestidoDaSuperficie liga UMA das duas aparências, e nunca deixa as duas.
//
// Os dois lados no `data-class` pela mesma armadilha de CASCATA que a aba do
// editor de bloco documenta: a marca de escolhida mora em `@layer components` e
// as cores do Tailwind são utilidades, que vivem numa camada POSTERIOR — camada
// vence especificidade, e o dourado perderia para o cinza sem nada acusar.
func oVestidoDaSuperficie(qual string) string {
	return fmt.Sprintf(
		"{'superficie-escolhida': %s, 'border-grimorio-iron': !(%s), 'text-muted-foreground': !(%s)}",
		naSuperficie(qual), naSuperficie(qual), naSuperficie(qual))
}
