package sheetui

// Os guardas das PROFICIÊNCIAS na ficha em Datastar (ALE-272, fatia 2).
//
// O que eles prendem é a REGRA — quem concede o quê, e o que o restaurar
// descarta — e não o desenho das sete linhas. A lista em si é dado transcrito e
// tem validação de schema no catálogo (`TestClassProficienciesTable`); repetir
// aqui um `expect` por categoria seria a tabela escrita duas vezes.

// panelTitle é a ficha das duas metades: ele CONCEDE armas marciais e armaduras
// pesadas, e NÃO concede exóticas — é a classe que separa "tem" de "não tem" sem
// precisar de duas fixtures.
// VARREDURA: toda aba da ficha desenha um painel de verdade.
//
// O par `Tabs` + o `switch` do `sheetPanel` é exatamente a forma
// que a convenção da casa manda mecanizar: são duas listas que precisam andar
// juntas, e o modo de errar é silencioso — um nome na lista sem caso no switch
// abre a seção VAZIA. O jogador vê uma tela em branco, que é pior do que
// qualquer aviso.
//
// Até a fatia 10 este guarda filtrava pelo placar `oPainelJaPortado`, porque
// havia abas ainda não portadas. Sem o placar ele cobra TODAS — que é o que ele
// sempre quis dizer.
//
// Ele falha nomeando A ABA que ficou vazia, que é a diferença entre "conserte
// isto" e "procure". Cada fatia desta issue acrescenta um nome ao mapa e este
// guarda cobra o painel no mesmo commit.
// O TÍTULO DO PAINEL NEM SEMPRE É O RÓTULO DA ABA, e a fatia 6 provou isso: a
// aba se chama "Magias" e o painel se chama "Grimório" — a aba nomeia o assunto,
// o painel nomeia a coisa, e é a escolha da tela antiga. O guarda passou a ter um
// mapa, e ele cobra que TODA aba portada tenha entrada: uma fatia nova sem a
// linha aqui falha nomeando a aba, em vez de o guarda se calar.
var panelTitle = map[string]string{
	"proficiencies": "Proficiências",
	"combat":        "Combate",
	"expertises":    "Perícias",
	"conditionals":  "Efeitos",
	"spells":        "Grimório",
	"bag":           "Mochila",
	"abilities":     "Poderes",
}
