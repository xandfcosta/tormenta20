package engine

// Modifier é um modificador de item. O `scale` (maxPv/maxPm) é ignorado pelo
// motor de resolução e preservado para o despejo de paridade da coleta.
type Modifier struct {
	Target ModifierTarget `json:"target"`
	Amount int            `json:"amount"`
	// Dice é a parcela em DADO — o "+1d6 de fogo" de uma arma flamejante
	// (p336). Ela existe ao lado do `Amount` e não no lugar dele porque as duas
	// se comportam diferente no crítico: o bônus numérico e o dado EXTRA não
	// multiplicam, e só os dados da ARMA multiplicam (p231).
	//
	// Vale só para `target.k = damage`, e quem a colhe é o `harvestWeaponRiders`
	// — ela nunca é parcela da pilha, pela mesma razão que o `Factor` não é.
	Dice string `json:"dice,omitempty"`
	// FromChoice diz qual campo do ALVO a escolha do jogador preenche, e a
	// escolha mora em `powerChoices`. O catálogo escreve o número uma vez e o
	// coletor faz uma cópia por escolha (`expandChoiceModifiers`); sem escolha,
	// o modificador não vale em lugar nenhum.
	//
	//	name   o nome do alvo — o atributo do Aumento de Atributo (p38)
	//	scope  sufixo do escopo escrito — `weapon:` + o id da arma, na
	//	       Especialização em Arma (p65)
	//	school a escola da magia — o Especialista e o Mestre em Escola (p38), e o
	//	       tomo hermético, cuja escolha é do item (`chosenByTheItem`)
	//	spell  a magia — o Foco em Magia (p131)
	//	god    o deus do item, que não preenche o alvo: o modificador só vale
	//	       para devoto dele — o símbolo sagrado (p157) (`chosenByTheItem`)
	FromChoice string             `json:"fromChoice,omitempty"`
	BonusType  string             `json:"bonusType"`
	Condition  *ModifierCondition `json:"condition,omitempty"`
	Note       string             `json:"note,omitempty"`
	Scale      *VitalScale        `json:"scale,omitempty"`
	// scaledBy é a nota de quem TROCOU o atributo da escala — "Balística: Inteligência no lugar de
	// Destreza" —, posta pela segunda passada (`resolveSecondPassSums`). Fica à parte da `Note`
	// porque a parcela ligada por interruptor troca a nota do modificador pela do interruptor, e a
	// troca de atributo tem de aparecer mesmo assim. O TEXTO é a nota da licença, escrita no catálogo;
	// o campo em si não viaja no JSON, e por isso não é exportado.
	scaledBy string
	// Factor MULTIPLICA o total do alvo, depois da soma — ver `factor.go`. Um
	// modificador com fator ignora o `Amount`: ele não é parcela da pilha.
	Factor *Ratio `json:"factor,omitempty"`
	// AppliesToCategory são as CATEGORIAS de item em que esta linha da
	// sobreposição vale.
	//
	// O livro escreve sobreposição que diz dois números, um por família do
	// PEÇA que a carrega: *"armaduras leves e escudos, RD 2; armaduras pesadas, RD 5"*
	// (o adamante, p166). Com o campo, o catálogo escreve as duas linhas e a
	// COLETA descarta a que não casa com a peça — vazia, ela vale para qualquer
	// item, que é o caso de quase todo modificador.
	//
	// O valor é a CATEGORIA do catálogo (`armor-heavy`, `shield`), e não um
	// vocabulário novo: uma segunda grafia para "armadura pesada" seria a
	// terceira do repositório, e as três envelheceriam em velocidades
	// diferentes.
	//
	// Não é CONDIÇÃO, e a diferença importa: a condição é avaliada na resolução
	// e sobrevive até a aba Efeitos, onde ofereceria um interruptor para uma
	// pergunta que o catálogo já respondeu quando o item foi lido.
	AppliesToCategory []string `json:"appliesToCategory,omitempty"`
	// From é a PEÇA de onde este modificador veio, quando ele chegou por
	// COMPOSIÇÃO — o `melhoria-reforcada` dentro do Baluarte anão (p340).
	//
	// Ele existe por causa do ENDEREÇO: um item composto dobra base, melhoria,
	// material e encanto numa fonte só, e sem a peça o +1 da Reforçada e o +2
	// do Defensor dividem `baluarte-anao::defense::::vested`. Calar um mataria
	// os dois, que é exatamente o que o `TestEveryCatalogTermHasAUniqueAddress`
	// existe para não deixar acontecer.
	//
	// Não é a NOTA com outro nome: a nota é prosa que uma errata reescreve, e
	// por isso o endereço não a carrega. Isto é identificador.
	From string `json:"from,omitempty"`
	// StacksWith e StacksWithAll são as exceções que o livro escreve à p226 numa redução de custo de PM:
	// "cumulativo com Mestre em Escola" (cajado do poder, p337), pelo id da fonte, e "cumulativo com
	// outras reduções de custo" (Foco em Magia, p131). Quem as lê é o `pmCostDispute`.
	StacksWith    []string `json:"stacksWith,omitempty"`
	StacksWithAll bool     `json:"stacksWithAll,omitempty"`
	// RaisesItem diz que o modificador AUMENTA o bônus que o próprio item dá: "o bônus na Defesa do item aumenta"
	// (a Reforçada, p165; o Defensor e o Guardião, p337 e p339). No empilhamento ele disputa só com outro aumento
	// do MESMO item, e o da armadura soma com o do escudo, porque cada um aumenta o seu — ver `stackingKey`.
	RaisesItem bool `json:"raisesItem,omitempty"`
	// classLevel é o nível na classe que concedeu o poder, anotado pela coleta
	// (`classActiveItems`) para o teto `classLevel` de um `scale` por atributo.
	// Fora do fio: é procedência da ficha, e não dado do catálogo.
	classLevel int
}
