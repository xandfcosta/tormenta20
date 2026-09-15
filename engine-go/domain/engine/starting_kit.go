package engine

// O KIT INICIAL — "Equipamento Inicial" (p140).
//
// O livro tem UM kit para todas as classes: nenhum bloco de classe (p36–83) traz
// uma seção de equipamento inicial. O que a classe muda são três linhas, e duas
// delas por PROFICIÊNCIA — a arma marcial e o escudo —, mais a exceção do
// arcanista, que o livro escreve pelo NOME da classe ("Exceção: arcanistas
// começam sem armadura") e não por proficiência.
//
// A armadura leve é do kit e NÃO depende de proficiência: o livro dá "uma
// armadura de couro, couro batido ou gibão de peles, a sua escolha" a todo
// mundo. Só a brunea e o escudo são condicionados.
//
// A ARMADURA É UMA ESCOLHA, e aqui a versão em TypeScript que este arquivo
// substitui divergia do livro: `class-starting-kits.ts` trocava as três leves
// pela brunea assim que a classe usava armaduras pesadas. O livro diz "em vez
// disso PODE começar com uma brunea" — quem usa pesadas escolhe entre QUATRO, e
// não recebe uma. Um guerreiro que quisesse gibão de peles não tinha como.

// Os itens do kit são IDs do catálogo (`catalog/data/items.json`) e não nomes:
// o item nasce na Mochila ligado à sua linha do livro, com os espaços e o preço
// que o catálogo já sabe. Nome solto viraria item inventado.
var (
	startingKitBaseItems = []string{"mochila", "saco-de-dormir", "traje-viajante"}
	startingLightArmors  = []string{"armadura-couro", "couro-batido", "gibao-peles"}
)

const (
	startingHeavyArmor = "brunea"
	startingShield     = "escudo-leve"

	// A classe que o livro excetua da armadura, por nome.
	arcanistaClassName = "Arcanista"
)

// StartingKit é o kit de p140 já estreitado pela classe: o que vem sem escolha,
// e as listas entre as quais o jogador escolhe na forja.
//
// A arma simples não é campo porque não é escolha de CLASSE — todo personagem
// começa com uma ("Todos os personagens sabem usar armas simples", p143), e
// qual delas é escolha do jogador sobre o catálogo inteiro da categoria.
type StartingKit struct {
	// BaseItems são os três que todo mundo leva: mochila, saco de dormir e
	// traje de viajante.
	BaseItems []string
	// MartialWeapon diz se a classe também começa com uma arma marcial.
	MartialWeapon bool
	// Armors são as armaduras entre as quais escolher. Vazia para o arcanista,
	// que começa sem nenhuma.
	Armors []string
	// Shield é o ID do escudo, ou "" quando a classe não usa escudos.
	Shield string
	// MoneyDice é o dado da bolsa inicial — sempre "4d6" no 1º nível.
	MoneyDice string
}

// StartingKitFor monta o kit de p140 para uma classe.
//
// Recebe as proficiências em vez de lê-las: elas são dado TRANSCRITO e moram no
// catálogo (`classes.json`), e o motor não abre catálogo para responder uma
// regra que cabe nos parâmetros.
//
//	StartingKitFor("Guerreiro", []string{"armas-marciais", "armaduras-pesadas", "escudos"})
func StartingKitFor(className string, proficiencies []string) StartingKit {
	usa := map[string]bool{}
	for _, categoria := range proficiencies {
		usa[categoria] = true
	}
	semArmadura := className == arcanistaClassName

	kit := StartingKit{
		BaseItems:     startingKitBaseItems,
		MartialWeapon: usa["armas-marciais"],
		MoneyDice:     StartingMoneyDice,
	}
	if semArmadura {
		return kit
	}
	kit.Armors = startingLightArmors
	if usa["armaduras-pesadas"] {
		kit.Armors = append(append([]string{}, startingLightArmors...), startingHeavyArmor)
	}
	if usa["escudos"] {
		kit.Shield = startingShield
	}
	return kit
}
