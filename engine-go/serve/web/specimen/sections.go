package specimen

// kitSection é uma divisão da folha viva, na ordem em que ela aparece. Os dois
// índices e os títulos das seções leem daqui.
type kitSection struct {
	ID    string
	Title string
	Lede  string
}

var specimenSections = []kitSection{
	{"acoes", "Ações", "O que a pessoa aperta."},
	{"campos", "Campos", "O que a pessoa escreve."},
	{"navegacao", "Navegação", "Como se anda entre telas e dentro delas."},
	{"listas", "Listas", "Várias coisas do mesmo tipo."},
	{"numeros", "Crachás e números", "Dados curtos, para bater o olho."},
	{"cartoes", "Cartões", "Uma coisa só, com o que ela tem de dizer."},
	{"retorno", "Retorno", "O que o app responde."},
}

func sectionByID(id string) kitSection {
	for _, section := range specimenSections {
		if section.ID == id {
			return section
		}
	}
	return kitSection{ID: id, Title: id}
}

func sectionTitle(id string) string { return sectionByID(id).Title }

func sectionLede(id string) string { return sectionByID(id).Lede }
