package convention

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TODA CENA QUE DESENHA PÁGINA TEM APARÊNCIA MEDIDA (ALE-320).
//
// Contraste e tipografia só existem no NAVEGADOR — converter oklch para sRGB e
// saber em que tamanho a Cinzel de fato foi desenhada são perguntas que nenhuma
// outra camada responde. Por isso a medição roda no Playwright; o problema era
// que ela morava lá ENUMERADA: dezoito endereços escritos um a um, catorze
// cópias do mesmo caso, e nada cobrando a cena que nascesse amanhã.
//
// É o regime que o `CLAUDE.md` chama de remendo na ALE-252 ("cada cena nova
// continuava precisando da própria linha"), e o molde do conserto é o da
// ALE-295: o guarda não pergunta "esta cena está na lista?", ele FORÇA a
// varredura.
//
// # Os dois lados
//
// Um é DERIVADO do código: cena que desenha página inteira é a que chama
// `WritePage`. O outro é o registro em `web/appearance_scenes.json`, que o spec
// de aparência percorre. Cena no primeiro e ausente no segundo é tela que
// ninguém abre com medidor.
//
// A primeira execução acusou o LEITOR (`/livro/ler`): página desde a ALE-264, e
// nunca uma medição de aparência sequer. Ninguém o omitiu de uma lista — a lista
// era escrita à mão, e o que não entra nela não existe.
//
// # POR QUE O REGISTRO MORA EM `web/` E NÃO EM `e2e/`
//
// Porque o `go test` CACHEIA por arquivo do módulo, e um registro em `e2e/` está
// fora dele. Medido, e ele mentiu: a primeira versão lia
// `e2e/tests/support/measured-scenes.ts`, e a sabotagem que tirou o leitor do
// registro passou VERDE — `ok (cached)` —, aparecendo só com `-count=1`. Um
// guarda que pode servir resultado velho sobre um repositório que ele não leu é
// pior que guarda nenhum: ele afirma.
//
// A lista também pertence aqui por direito. Quais cenas existem e em que
// endereço é fato do APP, e o app é Go; o e2e é consumidor.
//
// # O que ele NÃO mede
//
// Que o endereço registrado seja o certo, nem que a medição de fato rode nele. O
// outro lado disso é o denominador do próprio medidor: `medeOContraste` devolve
// `{falhas, medidos}` e o spec recusa um `medidos` baixo demais — é assim que "a
// cena não carregou" deixa de parecer "nada reprovou".
const appearanceScenesFile = "web/appearance_scenes.json"

type appearanceScene struct {
	// Visits são os endereços a medir — plural porque uma cena desenha telas
	// diferentes e, em duas delas, para PAPÉIS diferentes. A Mesa é o caso: o
	// mestre e o jogador recebem metades distintas do mesmo endereço, e medir só
	// uma é medir metade (ALE-276).
	Visits []appearanceVisit `json:"visits"`
}

type appearanceVisit struct {
	Address string `json:"address"`
	Session string `json:"session"`
	// MayBeAbsent é a cena que esta bancada pode não servir, e a ausência é
	// legítima: sem `LIVRO_PDF` o leitor devolve 404 de propósito. O spec PULA
	// com motivo visível, que é diferente de medir e diferente de reprovar.
	MayBeAbsent bool `json:"mayBeAbsent,omitempty"`
}

func TestEveryPageSceneIsMeasuredForAppearance(t *testing.T) {
	drawn := scenesThatDrawAPage(t)
	// O DENOMINADOR do lado do CÓDIGO. Se o casamento de `WritePage` parar de
	// valer — um renome do método, por exemplo —, a lista vem vazia e o guarda
	// passaria verde sem ter olhado cena nenhuma.
	if len(drawn) < 9 {
		t.Fatalf("achei %d cenas que desenham página, e são pelo menos nove: "+
			"o `WritePage` foi renomeado ou o caminho de `web/` mudou", len(drawn))
	}

	registered := registeredScenes(t)
	// O DENOMINADOR do lado do REGISTRO: arquivo trocado de formato e registro
	// vazio se parecem na lista de falhas.
	if len(registered) < 9 {
		t.Fatalf("o registro %s tem %d cenas, e são pelo menos nove: "+
			"ou ele mudou de formato, ou de lugar", appearanceScenesFile, len(registered))
	}

	var unmeasured []string
	for _, scene := range drawn {
		if _, ok := registered[scene]; !ok {
			unmeasured = append(unmeasured, scene)
		}
	}
	sort.Strings(unmeasured)

	if len(unmeasured) > 0 {
		t.Errorf("%d cena(s) desenham página inteira e NINGUÉM mede a aparência delas: %s\n"+
			"Acrescente o endereço de cada uma em engine-go/%s — o spec de aparência\n"+
			"percorre o registro, então a linha nova já entra medida por contraste e tipografia.\n"+
			"Cena fora do registro não é cena sem defeito: é cena sem medição.\n"+
			"(medidas: %d cenas desenham página, %d estão no registro)",
			len(unmeasured), strings.Join(unmeasured, ", "), appearanceScenesFile,
			len(drawn), len(registered))
	}

	// O CONTRÁRIO também é defeito: uma entrada que não corresponde a cena
	// nenhuma manda o spec abrir um endereço órfão, e um endereço órfão devolve
	// 404 — que o medidor lê como "nada reprovou" numa página de erro.
	var orphans []string
	drawnSet := map[string]bool{}
	for _, scene := range drawn {
		drawnSet[scene] = true
	}
	for scene := range registered {
		if !drawnSet[scene] {
			orphans = append(orphans, scene)
		}
	}
	sort.Strings(orphans)
	if len(orphans) > 0 {
		t.Errorf("%d entrada(s) do registro não são cena de `web/`: %s\n"+
			"Ou a cena foi apagada e a linha ficou, ou o nome está errado.",
			len(orphans), strings.Join(orphans, ", "))
	}
}

// scenesThatDrawAPage lista os pacotes de `web/` que chamam `WritePage`, que é o
// que separa uma CENA de um pedaço de cena: o buscador e o kit desenham
// fragmento, e medi-los seria medir de novo a página que os contém.
func scenesThatDrawAPage(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join("..", "web"))
	if err != nil {
		t.Fatalf("ler web/: %v", err)
	}
	var scenes []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if packageCallsWritePage(t, filepath.Join("..", "web", entry.Name())) {
			scenes = append(scenes, entry.Name())
		}
	}
	sort.Strings(scenes)
	return scenes
}

func packageCallsWritePage(t *testing.T, dir string) bool {
	t.Helper()
	sources, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("listar %s: %v", dir, err)
	}
	for _, path := range sources {
		if strings.HasSuffix(path, "_templ.go") || strings.HasSuffix(path, "_test.go") {
			continue
		}
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ler %s: %v", path, err)
		}
		if strings.Contains(string(body), "WritePage(") {
			return true
		}
	}
	return false
}

// registeredScenes lê o registro que o spec de aparência percorre.
func registeredScenes(t *testing.T) map[string]appearanceScene {
	t.Helper()
	// RASTREADO, e não só presente: um registro que existe na bancada e não no
	// índice é um guarda que a CI nunca executa — a forma mais cara de verde.
	tracked, err := exec.Command("git", "ls-files", "--cached", filepath.Join("..", appearanceScenesFile)).Output()
	if err != nil {
		t.Fatalf("git ls-files %s: %v", appearanceScenesFile, err)
	}
	if strings.TrimSpace(string(tracked)) == "" {
		t.Fatalf("%s não está rastreado pelo git.\n"+
			"Registro só na bancada é guarda que a CI nunca roda: `git add` nele.",
			appearanceScenesFile)
	}

	body, err := os.ReadFile(filepath.Join("..", appearanceScenesFile))
	if err != nil {
		t.Fatalf("ler %s: %v\nEle é o outro lado deste guarda: sem ele não há o que comparar",
			appearanceScenesFile, err)
	}
	var scenes map[string]appearanceScene
	if err := json.Unmarshal(body, &scenes); err != nil {
		t.Fatalf("%s não é JSON válido: %v", appearanceScenesFile, err)
	}
	for name, scene := range scenes {
		if len(scene.Visits) == 0 {
			t.Errorf("a entrada %q do registro não tem visita nenhuma: "+
				"uma cena registrada sem endereço é uma cena que ninguém abre", name)
			continue
		}
		for i, visit := range scene.Visits {
			if visit.Address == "" || visit.Session == "" {
				t.Errorf("a visita %d de %q está incompleta: address=%q session=%q",
					i, name, visit.Address, visit.Session)
			}
			if visit.Session != "gm" && visit.Session != "player" && visit.Session != "none" {
				t.Errorf("a visita %d de %q pede a sessão %q, e só existem gm, player e none",
					i, name, visit.Session)
			}
		}
	}
	return scenes
}
