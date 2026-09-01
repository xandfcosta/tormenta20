package book

// AS PROFICIÊNCIAS QUE A CLASSE CONCEDE (ALE-278).
//
// Elas moravam em dois lugares do `api` — a tabela na ficha, o cálculo na
// criação — e vieram para cá pela dependência: a tabela é lida do
// `catalog/data/classes.json`, então a pergunta é do LIVRO. A ficha e a forja só
// recebem a resposta.

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
	for _, cat := range proficiencyOrder {
		if granted[cat] {
			out = append(out, cat)
		}
	}
	return out
}

// proficiencyOrder is PROFICIENCY_CATEGORIES — grantedProficiencies emits in it.
var proficiencyOrder = []string{
	"armas-simples", "armas-marciais", "armas-exoticas", "armas-de-fogo",
	"armaduras-leves", "armaduras-pesadas", "escudos",
}

// asProficienciasPorClasse é a tabela do livro, lida do catálogo de classes.
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
