package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"t20engine/cmd/internal/pyfmt"
	"unicode/utf8"

	"t20engine/domain/catalog/transcript"
)

// Os quatro catálogos que o `--gravar` RECRIA a partir do livro. O Python os
// reescrevia do zero, sem `uid` (e as classes sem proficiências nem conjuração,
// que outro passo acrescentou depois); aqui o verbete recriado herda, pelo `id`,
// o que o livro não diz — o `uid` e, nas classes, as proficiências e a regra de
// conjuração —, porque perder a identidade de um verbete a cada regravação de
// página quebraria toda referência a ele.

func classID(name string) string { return strings.ReplaceAll(fold(name), " ", "-") }

// rebuildClasses acerta o catálogo das classes: id, nome e página das que o
// índice cita e o texto confirma. O `id` segue a convenção dos poderes de
// classe ("class.arcanista.…"): o nome dobrado, sem acento e sem espaço.
func (b *book) rebuildClasses() {
	kept := map[string]transcript.Class{}
	for _, c := range transcript.Classes {
		kept[c.ID] = c
	}
	names := transcript.CreationOptions.Classes
	var classes []transcript.Class
	for _, name := range names {
		var pages []int
		for _, p := range b.byIndex(name, "classe", "") {
			if b.confirms(name, p) {
				pages = append(pages, p)
			}
		}
		if len(pages) == 0 {
			fmt.Printf("   ✗ classe %s sem página — fica de fora\n", name)
			continue
		}
		c := kept[classID(name)]
		c.ID, c.Name, c.BookPage = classID(name), name, pages[0]
		classes = append(classes, c)
	}
	transcript.Classes = classes
	fmt.Printf("catálogo de classes: %d de %d\n", len(classes), len(names))
}

// effectsPage é a impressa onde o livro define os tipos de efeito, conferida
// pelo título: é o ÚNICO número de página que o script escreve à mão, e a
// conferência é o que impede que ele envelheça em silêncio.
const effectsPage = 228

// effectNames são os nomes que o livro imprime, na ordem dele. A lista existe
// para o extrator saber onde CORTAR — sem ela, uma frase que comece com
// maiúscula viraria um tipo inventado.
var effectNames = []string{
	"Arcano", "Atordoamento", "Cansaço", "Climático", "Cura", "Dano", "Divino",
	"Luz", "Mágico", "Medo", "Mental", "Metabolismo", "Metamorfose", "Movimento",
	"Perda de Vida", "Sentidos", "Trevas", "Veneno",
}

var trailingNumber = regexp.MustCompile(pySpace + `*[0-9]{1,3}` + pySpace + `*$`)

func pyStrip(s string) string { return strings.TrimFunc(s, isSpace) }

// beforeFirst é o `re.split(pattern, s)[0]` do Python.
func beforeFirst(pattern *regexp.Regexp, s string) string {
	if loc := pattern.FindStringIndex(s); loc != nil {
		return s[:loc[0]]
	}
	return s
}

// rebuildEffectTypes recria os tipos de efeito a partir do texto da p228. A
// condição CITA o tipo ("Abalado ... Medo.") e o leitor não tinha para onde ir.
// O `id` é o nome DOBRADO, a chave com que as condições já os guardam (`tags`),
// e tag sem tipo correspondente PARA o script: elo que aponta para o vazio é
// pior que elo nenhum.
//
// O Python tinha esta função DUAS vezes, e vale a última: esta, que corta no
// "Nome." e não emenda o hífen de quebra.
func (b *book) rebuildEffectTypes() error {
	body := anySpaces.ReplaceAllString(b.rawPage(effectsPage), " ")
	if !strings.Contains(body, "Tipos de efeitos") {
		return fatal{fmt.Sprintf("a p%d não é mais a dos tipos de efeito", effectsPage)}
	}
	type mark struct {
		at   int
		name string
	}
	var marks []mark
	for _, name := range effectNames {
		at := strings.Index(body, name+".")
		if at < 0 {
			return fatal{fmt.Sprintf("não achei o tipo %s na p%d", pyfmt.Repr(name), effectsPage)}
		}
		marks = append(marks, mark{at, name})
	}
	sort.Slice(marks, func(i, j int) bool {
		if marks[i].at != marks[j].at {
			return marks[i].at < marks[j].at
		}
		return marks[i].name < marks[j].name
	})

	// CORTA NO RODAPÉ: a página é de duas colunas, e o texto extraído continua
	// na SEÇÃO SEGUINTE depois do número da página — "Veneno" saiu com 1.800
	// caracteres. O corte é pelo número DESTA página e não por um número
	// qualquer: as definições citam páginas ("veja a página 230").
	footer := regexp.MustCompile(pySpace + strconv.Itoa(effectsPage) + pySpace)
	kept := map[string]string{}
	for _, e := range transcript.EffectTypes {
		kept[e.ID] = e.UID
	}
	var effects []transcript.EffectType
	for i, m := range marks {
		end := len(body)
		if i+1 < len(marks) {
			end = marks[i+1].at
		}
		text := pyStrip(body[m.at+len(m.name)+1 : end])
		text = pyStrip(beforeFirst(footer, text))
		text = pyStrip(trailingNumber.ReplaceAllString(text, ""))
		// O CONTROLE do corte: uma definição do livro é uma ou duas frases.
		if n := utf8.RuneCountInString(text); n > 400 {
			return fatal{fmt.Sprintf("a definição de %s veio com %d caracteres — o corte falhou", pyfmt.Repr(m.name), n)}
		}
		id := classID(m.name)
		effects = append(effects, transcript.EffectType{ID: id, Name: m.name, Description: text,
			BookPage: effectsPage, UID: kept[id]})
	}

	tags := map[string]bool{}
	for _, c := range transcript.Conditions {
		for _, t := range c.Tags {
			tags[t] = true
		}
	}
	for _, e := range effects {
		delete(tags, e.ID)
	}
	if len(tags) > 0 {
		return fatal{"tags de condição sem tipo de efeito: " + pyStrList(sortedKeys(tags))}
	}
	transcript.EffectTypes = effects
	fmt.Printf("catálogo de tipos de efeito: %d · nenhuma tag órfã\n", len(effects))
	return nil
}
