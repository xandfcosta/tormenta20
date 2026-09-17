package sheetui

// Os guardas das PROFICIÊNCIAS na ficha.
//
// O que eles prendem é a REGRA — quem concede o quê, e o que o restaurar
// descarta — e não o desenho das sete linhas. A lista em si é dado transcrito e
// tem validação de schema no catálogo (`TestClassProficienciesTable`); repetir
// aqui um `expect` por categoria seria a tabela escrita duas vezes.

// panelTitle é o TÍTULO que cada aba desenha, e ele NEM SEMPRE é o rótulo da
// aba: a aba se chama "Magias" e o painel se chama "Grimório" — a aba nomeia o
// assunto, o painel nomeia a coisa.
//
// Ele é a lista contra a qual a varredura cobra que TODA aba desenhe um painel
// de verdade. `Tabs` e o `switch` do `sheetPanel` são duas listas que precisam
// andar juntas, e o modo de errar é silencioso: um nome na lista sem caso no
// switch abre a seção VAZIA, e o jogador vê uma tela em branco — pior do que
// qualquer aviso. Aba nova sem linha aqui falha NOMEANDO a aba, em vez de o
// guarda se calar.
var panelTitle = map[string]string{
	"proficiencies": "Proficiências",
	"combat":        "Combate",
	"expertises":    "Perícias",
	"conditionals":  "Efeitos",
	"spells":        "Grimório",
	"bag":           "Mochila",
	"abilities":     "Poderes",
}
