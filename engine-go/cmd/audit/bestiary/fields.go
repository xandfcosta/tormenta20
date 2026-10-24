package main

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

const signed = `([+–−-]?\d+)`
const attr = `([+–−-]?\d+|—)`
const sep = bookpdf.PySpace + `*,?` + bookpdf.PySpace + `*`

var (
	initiativeRe = regexp.MustCompile(`Iniciativa` + bookpdf.PySpace + `*` + signed + bookpdf.PySpace + `*,` + bookpdf.PySpace + `*Percepção` + bookpdf.PySpace + `*` + signed)
	defenseRe    = regexp.MustCompile(`Defesa` + bookpdf.PySpace + `*(\d+)` + bookpdf.PySpace + `*,` + bookpdf.PySpace + `*Fort` + bookpdf.PySpace + `*` + signed +
		bookpdf.PySpace + `*,` + bookpdf.PySpace + `*Ref` + bookpdf.PySpace + `*` + signed + bookpdf.PySpace + `*,` + bookpdf.PySpace + `*Von` + bookpdf.PySpace + `*` + signed)
	hpRe         = regexp.MustCompile(`Pontos de Vida` + bookpdf.PySpace + `*(\d+)`)
	pmRe         = regexp.MustCompile(`Pontos de Mana` + bookpdf.PySpace + `*(\d+)`)
	attributesRe = regexp.MustCompile(`For` + bookpdf.PySpace + `*` + attr + sep + `Des` + bookpdf.PySpace + `*` + attr + sep + `Con` + bookpdf.PySpace + `*` + attr + sep +
		`Int` + bookpdf.PySpace + `*` + attr + sep + `Sab` + bookpdf.PySpace + `*` + attr + sep + `Car` + bookpdf.PySpace + `*` + attr)
	// Até o FIM: o valor pode ter ponto dentro ("T$ 2.500"), e o bloco já
	// termina na frase do tesouro.
	treasureRe = regexp.MustCompile(`Tesouro` + bookpdf.PySpace + `+(.+)$`)
	skillRe    = regexp.MustCompile(`^([^,+]+?(?:\([^)]*\))?)` + bookpdf.PySpace + `*([+–−-]\d+)` + bookpdf.PySpace + `*(?:\(([^)]*)\))?$`)
)

var attributeFields = []string{"forca", "destreza", "constituicao", "inteligencia", "sabedoria", "carisma"}

// readBlock são os campos lidos de um bloco. Um campo AUSENTE não está no mapa;
// um presente com nil é o "—" do livro, o `None` do script de origem.
type readBlock struct {
	numbers      map[string]*int
	skills       []transcript.CreatureSkill
	hasSkills    bool
	equipment    string
	hasEquipment bool
	treasure     string
	hasTreasure  bool
}

func (r readBlock) has(field string) bool {
	switch field {
	case "skills":
		return r.hasSkills
	case "equipamento":
		return r.hasEquipment
	case "tesouro":
		return r.hasTreasure
	}
	_, ok := r.numbers[field]
	return ok
}

func (r readBlock) number(field string) *int { return r.numbers[field] }

// equals é o `lido.get(campo) == valor` do script de origem.
func (r readBlock) equals(field string, v int) bool {
	n := r.numbers[field]
	return n != nil && *n == v
}

func mustNum(t string) *int {
	n, err := bookpdf.Num(t)
	if err != nil {
		panic("número do livro ilegível: " + t + " (esperado algo como +2, –1 ou 13)")
	}
	return &n
}

// readFields lê os campos do bloco. Os da CAUDA (atributos, Perícias,
// Equipamento, Tesouro) são a ÚLTIMA ocorrência: quando um bloco engole
// conteúdo de outra criatura, a primeira linha de atributos é a do intruso — foi
// assim que o Otyugh quase ganhou os 17 de Força do Sacerdote de Aharadak.
func readFields(b creatureBlock) readBlock {
	text := bookpdf.Join(b.body)
	r := readBlock{numbers: map[string]*int{}}
	if m := initiativeRe.FindStringSubmatch(text); m != nil {
		r.numbers["iniciativa"], r.numbers["percepcao"] = mustNum(m[1]), mustNum(m[2])
	}
	if m := defenseRe.FindStringSubmatch(text); m != nil {
		r.numbers["defesa"] = mustNum(m[1])
		r.numbers["fortitude"], r.numbers["reflexos"], r.numbers["vontade"] = mustNum(m[2]), mustNum(m[3]), mustNum(m[4])
	}
	if m := hpRe.FindStringSubmatch(text); m != nil {
		r.numbers["hp"] = mustNum(m[1])
	}
	if m := pmRe.FindStringSubmatch(text); m != nil {
		r.numbers["pm"] = mustNum(m[1])
	}
	if all := attributesRe.FindAllStringSubmatch(text, -1); all != nil {
		m := all[len(all)-1]
		for i, name := range attributeFields {
			if m[i+1] == "—" {
				r.numbers[name] = nil
			} else {
				r.numbers[name] = mustNum(m[i+1])
			}
		}
	}
	if g, ok := lastLazyField(text, "Perícias", []string{"Equipamento", "Tesouro"}, true); ok {
		r.skills, r.hasSkills = readSkills(g), true
	}
	if g, ok := lastLazyField(text, "Equipamento", []string{"Tesouro"}, false); ok {
		r.equipment, r.hasEquipment = strings.TrimRight(bookpdf.Strip(g), "."), true
	}
	if all := treasureRe.FindAllStringSubmatch(text, -1); all != nil {
		// Sem o ponto final: é pontuação da frase, não parte do valor.
		r.treasure, r.hasTreasure = strings.TrimRight(bookpdf.Strip(all[len(all)-1][1]), "."), true
	}
	return r
}

// readSkills lê "Furtividade +4 (+14 em pântanos), Ofício (armeiro) +2". Os
// dois parênteses querem coisas OPOSTAS e a diferença é a posição: antes do
// bônus é parte do NOME (`Ofício (armeiro)`); depois, é uma condição, que vai
// para a nota.
func readSkills(line string) []transcript.CreatureSkill {
	out := []transcript.CreatureSkill{}
	for _, part := range splitOutsideParens(strings.TrimRight(bookpdf.Strip(line), ".")) {
		m := skillRe.FindStringSubmatch(bookpdf.Strip(part))
		if m == nil {
			continue
		}
		skill := transcript.CreatureSkill{Name: bookpdf.Strip(m[1]), Bonus: *mustNum(m[2])}
		if m[3] != "" {
			skill.Note = bookpdf.Strip(m[3])
		}
		out = append(out, skill)
	}
	return out
}

// splitOutsideParens é o `re.split(r',\s*(?![^(]*\))')` do script de origem —
// a vírgula separa perícias, menos a que está DENTRO de parênteses. O RE2 não
// tem lookahead, então a condição é lida à mão: depois da vírgula, um `)` que
// chega antes de qualquer `(` quer dizer que ela está dentro de um.
func splitOutsideParens(s string) []string {
	var out []string
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] != ',' {
			continue
		}
		j := i + 1
		for j < len(s) {
			r, size := utf8.DecodeRuneInString(s[j:])
			if !bookpdf.IsPySpace(r) {
				break
			}
			j += size
		}
		if k := strings.IndexAny(s[j:], "()"); k >= 0 && s[j+k] == ')' {
			continue
		}
		out = append(out, s[last:i])
		last = j
	}
	return append(out, s[last:])
}

// lastLazyField é a ÚLTIMA ocorrência de `keyword\s+(.+?)(?=\s*(stops)\b|\.$|$)`
// — o lookahead que o RE2 não tem. O grupo é o menor trecho depois da palavra
// que termina antes de uma das `stops` (com fronteira de palavra Unicode), antes
// do ponto final do texto quando `dotEnd`, ou no fim.
func lastLazyField(text, keyword string, stops []string, dotEnd bool) (string, bool) {
	group, found := "", false
	for pos := 0; pos < len(text); {
		at := strings.Index(text[pos:], keyword)
		if at < 0 {
			break
		}
		at += pos
		start, spaces, lastSpace := at+len(keyword), 0, 0
		for start < len(text) {
			r, size := utf8.DecodeRuneInString(text[start:])
			if !bookpdf.IsPySpace(r) {
				break
			}
			lastSpace, spaces, start = start, spaces+1, start+size
		}
		if spaces > 1 && start == len(text) {
			// `.+?` precisa de um caractere: o `\s+` devolve o último espaço.
			start = lastSpace
		}
		if spaces == 0 || start == len(text) {
			pos = at + 1
			continue
		}
		_, size := utf8.DecodeRuneInString(text[start:])
		end := start + size
		for end < len(text) && !stopsAt(text, end, stops, dotEnd) {
			_, size := utf8.DecodeRuneInString(text[end:])
			end += size
		}
		group, found = text[start:end], true
		pos = end
	}
	return group, found
}

func stopsAt(text string, p int, stops []string, dotEnd bool) bool {
	if dotEnd && text[p] == '.' && p+1 == len(text) {
		return true
	}
	q := p
	for q < len(text) {
		r, size := utf8.DecodeRuneInString(text[q:])
		if !bookpdf.IsPySpace(r) {
			break
		}
		q += size
	}
	for _, stop := range stops {
		if strings.HasPrefix(text[q:], stop) && !wordAt(text, q+len(stop)) {
			return true
		}
	}
	return false
}

// isWord é o `\w` do Python: letra, número ou sublinhado, em Unicode.
func isWord(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) }

// wordAt diz se o caractere que começa em `i` é de palavra.
func wordAt(s string, i int) bool {
	if i >= len(s) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return isWord(r)
}

// wordBefore diz se o caractere que termina em `i` é de palavra.
func wordBefore(s string, i int) bool {
	if i <= 0 {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return isWord(r)
}
