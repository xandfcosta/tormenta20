package tabuleiro

// AS ESPÉCIES DE TERRENO — o que um quadrado FAZ com quem está nele ou atrás
// dele (T20 p238, Tabela 5-3).
//
// Este arquivo NASCEU na colheita do `0f08b14`, e a razão vale escrita. Na base
// da migração as espécies moram no fim do `vista.go`, que é maquinário de
// RENDERIZAÇÃO do piloto: lá o servidor desenha o extenso do tabuleiro, porque
// quem enquadra é o navegador por `transform`. A SPA desenha a própria janela e
// não precisa de nada daquilo — trazer o arquivo inteiro poria código morto na
// `main` só para carregar um `type` e quatro constantes.
//
// Então o domínio veio, o desenho do piloto ficou. As duas branches passam a ter
// os mesmos símbolos em arquivos diferentes, e isso é dívida CONHECIDA: quando
// elas se encontrarem, este arquivo e o fim do `vista.go` colidem. A sessão da
// migração foi avisada para fazer a mesma separação do lado dela, que é o que
// desfaz a dívida em vez de adiá-la.

// EspecieDeTerreno é uma das quatro coisas que um quadrado FAZ (T20 p238,
// Tabela 5-3). Ver GLOSSARIO.md: terreno é a família, não o chão do lugar.
type EspecieDeTerreno string

const (
	TerrenoDificil    EspecieDeTerreno = "dificil"
	TerrenoCobertura  EspecieDeTerreno = "cobertura"
	TerrenoCamuflagem EspecieDeTerreno = "camuflagem"
	TerrenoElevado    EspecieDeTerreno = "elevado"
)

// PincelDeTerreno é uma espécie pronta para a tela oferecer.
type PincelDeTerreno struct {
	ID     EspecieDeTerreno
	Rotulo string
	// Efeito é a frase do LIVRO, e ela vai para a tela porque hoje é tudo o que
	// acontece: só o DIFÍCIL é consumido por regra (entra no custo do
	// movimento). O app não resolve ataque contra Defesa em lugar nenhum — o
	// dano é aplicado à mão pelos vitais —, então "+5 na Defesa" não teria a
	// quem informar se não fosse dito por escrito. Um mapa tático existe para
	// tornar isto visível; hoje o mestre narra os três de cabeça.
	//
	// Quando a resolução de ataque chegar, o NÚMERO vira modificador no `engine`
	// e esta frase passa a derivar dele. Até lá ela é a única cópia, e a
	// autoridade é a página citada.
	Efeito string
}

// EspeciesDeTerreno é o que o pincel oferece, na ordem em que a tela mostra.
//
// O DIFÍCIL vem primeiro porque é o único que a regra consome e o único que
// existia antes — quem já usava o pincel encontra o de sempre no lugar de
// sempre.
var EspeciesDeTerreno = []PincelDeTerreno{
	{TerrenoDificil, "Difícil", "entrar custa o dobro"},
	{TerrenoCobertura, "Cobertura", "+5 na Defesa de quem está nela"},
	{TerrenoCamuflagem, "Camuflagem", "20% de chance de falha contra quem está nela"},
	{TerrenoElevado, "Elevado", "+2 no ataque de quem ataca de lá"},
}

// EspecieConhecida devolve a espécie pedida, ou o difícil.
//
// O padrão é o difícil e não um erro porque o id vem do cliente: uma espécie que
// a tela não oferece só chega por posse do fio, e a resposta a isso é pintar o
// que o pincel sempre pintou — não discutir.
func EspecieConhecida(pedido string) EspecieDeTerreno {
	for _, e := range EspeciesDeTerreno {
		if string(e.ID) == pedido {
			return e.ID
		}
	}
	return TerrenoDificil
}
