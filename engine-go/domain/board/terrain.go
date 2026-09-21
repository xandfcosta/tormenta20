package board

import "t20engine/domain/engine"

// O TERRENO — o que um quadrado FAZ com quem está nele ou atrás dele (T20 p238,
// Tabela 5-3): as quatro espécies, e o pincel que as pinta no tabuleiro.
//
// Arquivo próprio, separado do maquinário que DESENHA o tabuleiro: o que está
// aqui é domínio, e misturá-lo ao desenho faz qualquer leitor do domínio
// carregar a renderização junto.

// TerrainKind é uma das quatro coisas que um quadrado FAZ (T20 p238,
// Tabela 5-3). Ver GLOSSARY.md: terreno é a família, não o chão do lugar.
type TerrainKind string

// O VALOR é o segmento de ROTA, e por isso está em português: o
// endereço é a única parte da fronteira que uma pessoa vê, e `/terreno/dificil`
// é o que ela entende. As outras fronteiras — tabela, coluna, campo JSON, evento
// SSE — seguem em inglês, porque nenhuma aparece para ninguém.
//
// A CLASSE CSS que cada espécie veste é outra coisa: classe é IDENTIFICADOR e
// sai em inglês, e quem traduz é o `ClassOf` logo abaixo. As duas grafias
// conviverem é o preço de o mesmo conceito atravessar duas fronteiras com regras
// opostas — e é melhor pagá-lo aqui, num mapa de quatro linhas, do que numa
// classe pela metade (`board-dificil`).
const (
	TerrenoDificil    TerrainKind = "dificil"
	TerrenoCobertura  TerrainKind = "cobertura"
	TerrenoCamuflagem TerrainKind = "camuflagem"
	TerrenoElevado    TerrainKind = "elevado"
)

// ClassOf devolve o nome INGLÊS da espécie, que é o que a classe CSS usa.
//
// Espécie desconhecida devolve vazio, e quem chama escreve uma classe a menos —
// o certo, porque o `TerrainKinds` é a lista fechada e um id fora dela não veio
// da tela.
func ClassOf(k TerrainKind) string {
	switch k {
	case TerrenoDificil:
		return "difficult"
	case TerrenoCobertura:
		return "cover"
	case TerrenoCamuflagem:
		return "concealment"
	case TerrenoElevado:
		return "elevated"
	}
	return ""
}

// TerrainBrush é uma espécie pronta para a tela oferecer.
type TerrainBrush struct {
	ID    TerrainKind
	Label string
	// Effect é a frase do LIVRO, e ela vai para a tela porque hoje é tudo o que
	// acontece: só o DIFÍCIL é consumido por regra (entra no custo do
	// movimento). O app não resolve ataque contra Defesa em lugar nenhum — o
	// dano é aplicado à mão pelos vitais —, então "+5 na Defesa" não teria a
	// quem informar se não fosse dito por escrito. Um mapa tático existe para
	// tornar isto visível; hoje o mestre narra os três de cabeça.
	//
	// Quando a resolução de ataque chegar, o NÚMERO vira modificador no `engine`
	// e esta frase passa a derivar dele. Até lá ela é a única cópia, e a
	// autoridade é a página citada.
	Effect string
}

// TerrainKinds é o que o pincel oferece, na ordem em que a tela mostra.
//
// O DIFÍCIL vem primeiro porque é o único que a regra consome e o único que
// existia antes — quem já usava o pincel encontra o de sempre no lugar de
// sempre.
var TerrainKinds = []TerrainBrush{
	{TerrenoDificil, "Difícil", "entrar custa o dobro"},
	{TerrenoCobertura, "Cobertura", "+5 na Defesa de quem está nela"},
	{TerrenoCamuflagem, "Camuflagem", "20% de chance de falha contra quem está nela"},
	{TerrenoElevado, "Elevado", "+2 no ataque de quem ataca de lá"},
}

// KnownTerrainKind devolve a espécie pedida, ou o difícil.
//
// O padrão é o difícil e não um erro porque o id vem do cliente: uma espécie que
// a tela não oferece só chega por posse do fio, e a resposta a isso é pintar o
// que o pincel sempre pintou — não discutir.
func KnownTerrainKind(requested string) TerrainKind {
	for _, e := range TerrainKinds {
		if string(e.ID) == requested {
			return e.ID
		}
	}
	return TerrenoDificil
}

// PaintTerrain marca ou apaga UMA casa como terreno difícil (T20 p238).
//
// Recebe o valor DESEJADO e não alterna, e a razão mudou junto com a tela: o
// pincel pinta ARRASTANDO, e o arraste passa pela mesma casa mais de uma vez —
// alternar faria a casa piscar entre brejo e chão limpo debaixo do dedo. Com o
// valor explícito a mensagem é idempotente, que é o que um arraste precisa.
// Quem apaga é a borracha, que manda `false`.
func PaintTerrain(b *BoardState, square engine.Square, species TerrainKind, on bool) {
	list := listForKind(b, species)
	if list == nil {
		return // espécie que não existe não pinta nada, e não derruba a mesa
	}
	for i, existing := range *list {
		if existing == square {
			if on {
				return // já é brejo: nada mudou, e a versão não sobe à toa
			}
			*list = append((*list)[:i], (*list)[i+1:]...)
			b.Version++
			return
		}
	}
	if !on {
		return
	}
	*list = append(*list, square)
	b.Version++
}

// listForKind é o ÚNICO lugar que sabe qual lista guarda qual espécie.
//
// Devolve ponteiro para o campo porque o pincel escreve nele. É o que segura a
// repetição das quatro listas irmãs num ponto só: acrescentar uma quinta espécie
// é uma linha aqui e uma no `TerrainKinds`, e o resto do código não muda.
//
// nil para espécie desconhecida, e o pincel trata: o id vem do cliente, e uma
// espécie inventada não pode derrubar a mesa nem pintar a lista errada.
func listForKind(b *BoardState, species TerrainKind) *[]engine.Square {
	switch species {
	case TerrenoDificil:
		return &b.Difficult
	case TerrenoCobertura:
		return &b.Cover
	case TerrenoCamuflagem:
		return &b.Concealment
	case TerrenoElevado:
		return &b.Elevated
	}
	return nil
}

// ClearSquare tira TODO terreno de um quadrado, seja qual for a espécie
// (decisão do dono).
//
// O pincel na mão NÃO entra na conta, e a alternativa — a borracha como MODO
// que inverte o pincel selecionado — é o defeito que ela consertou: com
// `Cobertura` na mão, clicar num quadrado de `Difícil` apagava a cobertura que
// não estava ali, e a tela não dizia nada. O que se perde é "tirar só a
// cobertura desta casa": repintar o que sobrou é um clique, e descobrir por que
// um gesto não fez nada é uma noite.
//
// Devolve se ALGUMA COISA saiu: quem chama usa para não subir a versão (e não
// acordar a mesa) por um clique em chão limpo.
func ClearSquare(b *BoardState, square engine.Square) bool {
	cleared := false
	for _, brush := range TerrainKinds {
		list := listForKind(b, brush.ID)
		if list == nil {
			continue
		}
		for i, existing := range *list {
			if existing == square {
				*list = append((*list)[:i], (*list)[i+1:]...)
				cleared = true
				break
			}
		}
	}
	if cleared {
		b.Version++
	}
	return cleared
}

// moveTerrainOf traduz a lista esparsa para o que o motor cobra. A conversão
// mora aqui e não no motor porque o motor não conhece tabuleiro: ele responde
// sobre um caminho e um chão, e quem tem chão é o estado.
func moveTerrainOf(b *BoardState) engine.MoveTerrain {
	if len(b.Difficult) == 0 {
		return engine.MoveTerrain{}
	}
	difficult := make(map[engine.Square]bool, len(b.Difficult))
	for _, square := range b.Difficult {
		difficult[square] = true
	}
	return engine.MoveTerrain{Difficult: difficult}
}

// SquaresOf são as casas pintadas de uma espécie, para quem só LÊ.
//
// Existe para o mapeamento espécie→lista continuar com um dono só: sem ela,
// quem desenha refaz o `switch` do `listForKind` do lado de fora, e é a cópia
// de fora que fica para trás quando a quinta espécie chegar. Devolve a fatia e
// não o ponteiro justamente por ser leitura — o pincel é quem escreve.
func SquaresOf(b *BoardState, species TerrainKind) []engine.Square {
	if b == nil {
		return nil
	}
	if list := listForKind(b, species); list != nil {
		return *list
	}
	return nil
}
