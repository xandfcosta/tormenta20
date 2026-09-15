package book

// AS PROFICIÊNCIAS QUE A CLASSE CONCEDE.
//
// Elas moram aqui e não na ficha nem na forja porque a tabela é lida do
// `catalog/data/classes.json`: a pergunta é do LIVRO, e as telas só recebem a
// resposta.

// grantedProficiencies ports characterProficiencies(...).filter(granted): the
// default proficiency categories for a class set, in catalog order.
func GrantedProficiencies(classNames []string) []string {
	granted := map[string]bool{"armas-simples": true}
	porClasse := ProficienciesByClass()
	for _, cls := range classNames {
		for _, cat := range porClasse[cls] {
			granted[cat] = true
			if cat == "armaduras-pesadas" {
				granted["armaduras-leves"] = true
			}
		}
	}
	out := []string{}
	for _, cat := range ProficiencyCategories {
		if granted[cat.Key] {
			out = append(out, cat.Key)
		}
	}
	return out
}

// AS SETE CATEGORIAS, escritas UMA vez.
//
// Elas já estiveram em TRÊS lugares — esta lista de chaves, o conjunto que
// recusava categoria inventada, e a lista com rótulo e grupo que a aba
// Proficiências desenha —, com os mesmos sete nomes e na mesma ordem. Não há
// guarda contra isso: três transcrições do mesmo dado compilam e ficam verdes
// até uma divergir.
//
// A ORDEM É A DO LIVRO dentro de cada grupo (p142 para as armas, p148 para as
// armaduras), e não alfabética: "simples, marciais, exóticas, de fogo" é uma
// escala de dificuldade, e ordenar por nome a embaralharia.
type ProficiencyCategory struct {
	Key   string
	Label string
	// Group é a divisão do próprio livro — as armas na p142, as armaduras e o
	// escudo na p148 —, e é por ela que o painel da ficha separa as duas seções.
	Group string
}

const (
	WeaponGroup = "Armas"
	ArmorGroup  = "Armaduras & Escudos"
)

var ProficiencyCategories = []ProficiencyCategory{
	{Key: "armas-simples", Label: "Armas simples", Group: WeaponGroup},
	{Key: "armas-marciais", Label: "Armas marciais", Group: WeaponGroup},
	{Key: "armas-exoticas", Label: "Armas exóticas", Group: WeaponGroup},
	{Key: "armas-de-fogo", Label: "Armas de fogo", Group: WeaponGroup},
	{Key: "armaduras-leves", Label: "Armaduras leves", Group: ArmorGroup},
	{Key: "armaduras-pesadas", Label: "Armaduras pesadas", Group: ArmorGroup},
	{Key: "escudos", Label: "Escudos", Group: ArmorGroup},
}

// IsProficiencyCategory diz se a chave é uma das sete.
func IsProficiencyCategory(chave string) bool {
	for _, c := range ProficiencyCategories {
		if c.Key == chave {
			return true
		}
	}
	return false
}

// ProficiencyKeys são as sete chaves na ordem do livro.
func ProficiencyKeys() []string {
	chaves := make([]string, 0, len(ProficiencyCategories))
	for _, c := range ProficiencyCategories {
		chaves = append(chaves, c.Key)
	}
	return chaves
}

// ProficienciesByClass é a tabela do livro, lida do catálogo de classes.
//
// Ela sai de `catalog/data/classes.json` e não de um `map` escrito em Go pelo
// mesmo motivo das perícias de classe: é DADO TRANSCRITO — a linha
// "Proficiências." de cada classe, p36–83 — e dado transcrito mora no catálogo,
// onde a validação de schema o alcança.
func ProficienciesByClass() map[string][]string {
	_, classes, _ := CharacterCatalogs()
	tabela := make(map[string][]string, len(classes))
	for _, c := range classes {
		tabela[c.Name] = c.Proficiencias
	}
	return tabela
}
