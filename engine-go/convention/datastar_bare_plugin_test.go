package convention

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// NENHUM GANCHO DE CSS USA O NOME DE UM PLUGIN DO DATASTAR.
//
// Um atributo como `data-text` sem valor, posto para o CSS achar a linha, é
// para o Datastar o plugin que escreve texto no elemento — e sem valor ele lança
// `ValueRequired` e PARA de avaliar a página. Nada mais acende: na mesa do
// mestre, o `$surface` deixou de valer e as três superfícies do celular
// apareceram empilhadas. Nenhum teste de Go viu, e o e2e inteiro passou verde,
// porque nenhum caso abria aquela cena naquele estado.
//
// Os nomes vêm do BUNDLE embutido e não de uma lista escrita aqui: uma versão
// nova do Datastar que traga um plugin novo passa a valer sozinha, e um gancho
// que hoje é inocente reprova no dia em que deixar de ser.

var datastarPluginName = regexp.MustCompile(`name:"([a-z][a-z-]*)"`)

// bareDataAttribute é o `data-x` SEM valor e sem chave: nem `=`, nem `:`, nem
// modificador `__`. É a forma do gancho de CSS.
var bareDataAttribute = regexp.MustCompile(`\sdata-([a-z][a-z-]*)(?:\s|>|/>|$)`)

func TestNoStyleHookIsNamedAfterADatastarPlugin(t *testing.T) {
	root := filepath.Join("..", "..")
	bundle, err := os.ReadFile(filepath.Join("..", "serve", "web", "assets", "static", "datastar.js"))
	if err != nil {
		t.Fatalf("ler o bundle do Datastar: %v", err)
	}
	plugins := map[string]bool{}
	for _, m := range datastarPluginName.FindAllStringSubmatch(string(bundle), -1) {
		plugins[m[1]] = true
	}
	// O CONTROLE: sem os plugins que o app usa em toda cena, o leitor do bundle
	// ficou cego e "nenhum gancho reprovou" não diria nada.
	for _, known := range []string{"text", "show", "signals", "on", "bind"} {
		if !plugins[known] {
			t.Fatalf("o bundle do Datastar não trouxe o plugin %q — o formato do `name:\"…\"` mudou, "+
				"e este guarda precisa aprender a lê-lo antes de valer alguma coisa", known)
		}
	}

	output, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "*.templ").Output()
	if err != nil {
		t.Fatalf("git ls-files em %s: %v", root, err)
	}
	filesRead, hooksRead := 0, 0
	var offenders []string
	for _, relative := range strings.Split(strings.TrimRight(string(output), "\x00"), "\x00") {
		if relative == "" {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		filesRead++
		for n, line := range strings.Split(string(body), "\n") {
			// O comentário que ENSINA a armadilha cita a forma errada de propósito.
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for _, m := range bareDataAttribute.FindAllStringSubmatch(line, -1) {
				hooksRead++
				if plugins[m[1]] {
					offenders = append(offenders, relative+":"+strconv.Itoa(n+1)+" — data-"+m[1])
				}
			}
		}
	}
	if filesRead == 0 || hooksRead == 0 {
		t.Fatalf("leu %d .templ e %d ganchos `data-x` sem valor: a varredura não viu nada", filesRead, hooksRead)
	}
	if len(offenders) > 0 {
		t.Errorf("gancho de CSS com o nome de um plugin do Datastar — %d de %d ganchos, em %d .templ:\n  %s\n"+
			"Sem valor, o plugin lança `ValueRequired` e o Datastar para de avaliar a página inteira. "+
			"Dê ao gancho um nome que não seja de plugin (ex.: `data-sentence` no lugar de `data-text`).",
			len(offenders), hooksRead, filesRead, strings.Join(offenders, "\n  "))
	}
}
