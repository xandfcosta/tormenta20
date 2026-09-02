package api

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TINTA QUE NÃO EXISTE NA FOLHA NÃO DESENHA — e não reclama (ALE-272, fatia 6).
//
// O `text-grimorio-ink` foi escrito na fatia 5 e nunca existiu: `grimorio-ink`
// não é token da paleta. O Tailwind não emite regra para uma classe que não
// conhece, o elemento fica com a cor HERDADA, e o crachá de contagem dos
// Efeitos saiu com número dourado sobre fundo dourado — 1,53:1, ilegível. Nada
// falha: o HTML tem a classe, o `templ generate` passa, o `go build` passa, e a
// folha simplesmente não tem a regra.
//
// O medidor de contraste do e2e pegou este caso, mas por sorte de ESTADO: o
// crachá só aparece quando há efeito ativo, e no dia em que a suíte rodou
// inteira o personagem medido não tinha nenhum. Guarda que depende de o dado
// certo estar no banco é guarda que mede quando quer.
//
// Ele varre a FONTE inteira — os `.templ` e os `.go` do piloto —, e não uma
// cena servida: a varredura é o que faz a convenção valer para a próxima tela
// também.
func TestEveryHouseTintExistsInTheStylesheet(t *testing.T) {
	folha, err := os.ReadFile(filepath.Join("piloto", "static", "piloto.css"))
	if err != nil {
		t.Fatalf("ler a folha compilada: %v", err)
	}
	arquivos := osFontesDoPiloto(t)
	usadas := map[string][]string{}
	for _, caminho := range arquivos {
		fonte, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		for _, tinta := range tintasDaCasaEm(semOsComentarios(string(fonte))) {
			usadas[tinta] = append(usadas[tinta], filepath.Base(caminho))
		}
	}
	// O DENOMINADOR: um padrão que parasse de casar daria zero tintas e este
	// guarda passaria verde afirmando nada.
	// O DENOMINADOR. Ele é o que separa "nenhuma tinta reprovou" de "não varri
	// nada", e o piso subiu com o kit entrando na lista (ALE-278, fatia 4): um
	// piso que o conjunto MENOR já satisfazia não denunciaria a perda do maior.
	//
	// MEDIDO, e o número desmente a intuição: são 21 tintas COM o kit na varredura
	// e 21 SEM ele. Todo token que o kit escreve já é escrito por alguma cena
	// também, então incluí-lo não comprou cobertura HOJE. Ele fica porque compra
	// AMANHÃ: no dia em que o kit for o único lugar de um token — e ele é o lugar
	// natural para isso, já que existe para as classes serem escritas uma vez —,
	// sem esta linha o token nasceria sem medição.
	//
	// O PISO SUBIU DE 20 PARA 40 na ALE-276, quando os papéis semânticos entraram
	// em `asPaletasDaCasa`: a varredura foi de 21 tintas para 43. O piso é o
	// mesmo argumento do parágrafo acima levado a sério — 20 é um número que o
	// conjunto ANTIGO satisfazia, e deixá-lo ali faria a volta acidental da lista
	// curta passar verde. O piso mede o conjunto de HOJE.
	if len(usadas) < 40 {
		t.Fatalf("a varredura achou %d tintas da casa, e são dezenas: o padrão parou de casar", len(usadas))
	}

	nomes := make([]string, 0, len(usadas))
	for nome := range usadas {
		nomes = append(nomes, nome)
	}
	sort.Strings(nomes)
	for _, nome := range nomes {
		if aFolhaConhece(string(folha), nome) {
			continue
		}
		t.Errorf("a tinta %q não existe na folha (usada em %s): o elemento sai com a cor herdada e ninguém reclama",
			nome, strings.Join(usadas[nome], ", "))
	}
}

// osFontesDoPiloto são os arquivos que escrevem classe: os `.templ`, que é o
// que o scanner do Tailwind lê, e os `.go` do piloto, onde uma classe escrita
// não passa pelo scanner e só existe se alguém a registrou no `@source inline`.
// O gerado (`_templ.go`) fica de fora porque repete o `.templ` ao lado.
//
// O KIT ENTROU NA LISTA na ALE-278 (fatia 4), e a razão é a forma da falha desta
// família: o kit saiu de `api/piloto_ui.templ` para `web/ui/kit.templ`, e o
// padrão `piloto_*` deixou de casar com ele. O guarda teria continuado VERDE
// medindo as cenas e ignorando o botão, o campo e a casca — que são justamente
// os arquivos onde uma tinta errada aparece em TODA tela.
//
// É o mesmo defeito que o `CLAUDE.md` da raiz descreve como "um guarda só mede o
// que ele VISITA", e desta vez o que sumiu da visita não foi uma cena esquecida:
// foi um arquivo que mudou de nome.
func osFontesDoPiloto(t *testing.T) []string {
	t.Helper()
	fora := []string{}
	interessa := func(caminho string) bool {
		if strings.HasSuffix(caminho, "_test.go") || strings.HasSuffix(caminho, "_templ.go") {
			return false
		}
		return strings.HasSuffix(caminho, ".templ") || strings.HasSuffix(caminho, ".go")
	}

	achados, err := filepath.Glob("piloto_*.templ")
	if err == nil {
		outros, _ := filepath.Glob("piloto_*.go")
		achados = append(achados, outros...)
	}
	for _, caminho := range achados {
		if interessa(caminho) {
			fora = append(fora, caminho)
		}
	}

	// O `web/` INTEIRO, e não um pacote por linha. A terceira cena a sair
	// (`web/grimoire`) caiu fora da lista enumerada e o denominador acusou na
	// hora — 21 tintas viraram 13, porque a folha de especificação é onde mais
	// tinta da casa é escrita. Enumerar faria a PRÓXIMA cena nascer sem medição.
	if err := filepath.WalkDir("../web", func(caminho string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entrada.IsDir() && interessa(caminho) {
			fora = append(fora, caminho)
		}
		return nil
	}); err != nil {
		t.Fatalf("varrer o web/: %v", err)
	}
	if len(fora) == 0 {
		t.Fatal("nenhuma fonte do piloto encontrada: este guarda mediria o vazio")
	}
	return fora
}

// semOsComentarios tira as linhas de comentário antes da varredura: uma
// docstring que CITA a classe errada — como a que explica este guarda — não é
// tinta escrita em elemento nenhum, e cobrá-la faria o guarda acusar prosa.
func semOsComentarios(fonte string) string {
	linhas := strings.Split(fonte, "\n")
	fora := make([]string, 0, len(linhas))
	for _, linha := range linhas {
		if strings.HasPrefix(strings.TrimSpace(linha), "//") {
			continue
		}
		fora = append(fora, linha)
	}
	return strings.Join(fora, "\n")
}

// asPaletasDaCasa são os prefixos de token DESTE projeto. A paleta embutida do
// Tailwind fica de fora de propósito: ela existe sempre, e incluí-la só traria
// o ruído das variantes sem prender defeito nenhum.
//
// OS PAPÉIS SEMÂNTICOS ENTRARAM NA ALE-276, e a ausência deles era um buraco do
// mesmo formato que o do kit: `text-destructive-foreground` foi escrito em três
// sítios e o token NUNCA existiu na paleta. Este guarda passou por cima porque
// `destructive` não estava nesta lista — ele media `grimorio-*` e os quatro
// papéis de `-ink`, e a família semântica inteira (`primary`, `muted`, `card`,
// `popover`, `destructive`…) estava fora da varredura.
//
// O sintoma foi o de sempre: o botão "Excluir a sessão" herdou o `--foreground`
// do diálogo, saiu com tinta de pergaminho sobre crimson a 4,33:1, e nada
// falhou — nem o `templ generate`, nem o `go build`, nem este guarda. Quem
// acusou foi um caso de e2e que ainda não existia.
//
// A lista é a dos `--color-*` que o `@theme` do `index.css` declara, menos o
// que a paleta embutida do Tailwind já cobre. Token semântico novo entra aqui
// junto com a linha do `@theme` — as duas metades do mesmo ato.
var asPaletasDaCasa = []string{
	"grimorio", "arcane", "penalty", "hp", "mp", "terreno",
	"accent", "background", "bonus", "border", "card", "destructive",
	"foreground", "input", "marker", "muted", "popover", "primary",
	"ring", "secondary", "warning",
}

var oUtilitarioDeCor = regexp.MustCompile(
	`\b(?:text|bg|border|ring|outline|fill|stroke|decoration|shadow|from|via|to)-([a-z]+(?:-[a-z0-9]+)*)`)

// tintasDaCasaEm devolve os NOMES de token da casa citados no arquivo.
//
// O nome, e não a classe inteira, porque a folha escreve o mesmo token de
// várias formas — `.text-arcane-ink`, `.text-arcane-ink\/80`, `--arcane-ink` —
// e cobrar uma forma específica cobraria a implementação do Tailwind.
func tintasDaCasaEm(fonte string) []string {
	fora := []string{}
	vistos := map[string]bool{}
	for _, achado := range oUtilitarioDeCor.FindAllStringSubmatch(fonte, -1) {
		nome := achado[1]
		if !daCasa(nome) || vistos[nome] {
			continue
		}
		vistos[nome] = true
		fora = append(fora, nome)
	}
	return fora
}

func daCasa(nome string) bool {
	for _, paleta := range asPaletasDaCasa {
		if nome == paleta || strings.HasPrefix(nome, paleta+"-") {
			return true
		}
	}
	return false
}

// aFolhaConhece procura o nome do token seguido de algo que NÃO continue o
// nome. Sem essa borda, `arcane` passaria por causa de `arcane-ink` — e um
// token inventado que fosse prefixo de um real nunca seria pego.
func aFolhaConhece(folha, nome string) bool {
	borda := regexp.MustCompile(regexp.QuoteMeta(nome) + `([^a-z0-9-]|$)`)
	return borda.MatchString(folha)
}
