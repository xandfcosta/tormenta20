package sheet

// AS VINTE E NOVE PERÍCIAS DO LIVRO, e o atributo-chave de cada uma — a Tabela
// 2-1 da p115, conferida coluna a coluna.
//
// Toda ficha nasce com TODAS, treinadas ou não. A lista é fechada porque o livro
// a fecha, e uma perícia que só existisse na ficha quando treinada obrigaria a
// tela a inventar as outras vinte e oito para desenhar a aba.
//
// Ela mora no domínio e não em quem grava: é transcrição do livro, e o
// `Iniciativa` daqui é o mesmo nome que o `engine.InitiativeTotal` procura. Duas
// cópias divergiriam num acento, e o sintoma seria o bônus virando zero em
// silêncio.
//
// O OFÍCIO é a exceção viva: o livro o trata como família, e a ficha ganha
// linhas novas dele pelo `saveNewCraft`. Por isso ele entra aqui uma vez, como
// as outras, e o resto é escolha de quem joga.

// Expertise é uma perícia do livro: o nome e o atributo que a chaveia.
type Expertise struct {
	Name      string
	Attribute string
}

// builtinExpertises é a tabela, na ordem do livro.
var builtinExpertises = []Expertise{
	{"Acrobacia", "dexterity"}, {"Adestramento", "charisma"}, {"Atletismo", "strength"},
	{"Atuação", "charisma"}, {"Cavalgar", "dexterity"}, {"Conhecimento", "intelligence"},
	{"Cura", "wisdom"}, {"Diplomacia", "charisma"}, {"Enganação", "charisma"},
	{"Fortitude", "constitution"}, {"Furtividade", "dexterity"}, {"Guerra", "intelligence"},
	{"Iniciativa", "dexterity"}, {"Intimidação", "charisma"}, {"Intuição", "wisdom"},
	{"Investigação", "intelligence"}, {"Jogatina", "charisma"}, {"Ladinagem", "dexterity"},
	{"Luta", "strength"}, {"Misticismo", "intelligence"}, {"Nobreza", "intelligence"},
	{"Ofício", "intelligence"}, {"Percepção", "wisdom"}, {"Pilotagem", "dexterity"},
	{"Pontaria", "dexterity"}, {"Reflexos", "dexterity"}, {"Religião", "wisdom"},
	{"Sobrevivência", "wisdom"}, {"Vontade", "wisdom"},
}

// BuiltinExpertises devolve a lista, e devolve uma CÓPIA: a fatia é global, e
// quem a recebesse por referência poderia reordenar a tabela do livro para todo
// mundo sem sair do próprio arquivo.
func BuiltinExpertises() []Expertise {
	fora := make([]Expertise, len(builtinExpertises))
	copy(fora, builtinExpertises)
	return fora
}
