package api

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// A TAG JSON É CONTRATO COM O CLIENTE, E ELA COMEÇA EM MINÚSCULA.
//
// O mecanismo que ele caça: um rename automático casa DENTRO da string da tag e
// capitaliza o campo. O efeito é silencioso — o cliente lê a grafia minúscula e
// recebe `undefined`, sem erro em lugar nenhum —, e nenhum teste de Go acusa,
// porque o mesmo laço renomeia as asserções junto. Teste que muda com a mudança
// não acusa a mudança.
//
// A regra é do glossário: **a fronteira fala inglês**, e o cliente foi escrito
// contra a grafia minúscula. Trocar a caixa quebra cliente e migração por zero.
//
// ELE CAMINHA A ÁRVORE, e não uma lista de pacotes: a versão que enumerava
// quatro perdeu as tags da Mesa no dia em que ela virou `web/table`, e só não
// seguiu verde medindo menos porque o piso existia. Enumerar é remendo; o que
// restaura a amostragem é a caminhada, e o pacote novo nasce medido.

// A ROTA TAMBÉM É FIO, E TAMBÉM COMEÇA EM MINÚSCULA.
//
// Mesma família (rename que varre uma STRING junto) e mesma resposta (conserte a
// string, não o guarda), e por isso mesmo arquivo — separá-los faria parecer que
// são dois problemas. O estrago é maior que o das tags: uma rota capitalizada
// vira 404, e o 404 chega à tela como funcionalidade que sumiu — inclusive em
// rotas que ninguém exercita no dia a dia, como redefinir senha.
//
// # Ele pergunta ao ROTEADOR, e a troca não foi de estilo (ALE-345)
//
// Aqui morava um regex sobre a fonte: `r.Post("(/…)"`. Ele só enxergava o
// registro cujo caminho é um literal COLADO no parêntese — e a forma dominante
// da cena da sessão é `base := …` seguido de `r.Post(base+"/iniciar", …)`, que
// nunca casou. O guarda media 98 rotas de 242, e as que faltavam eram
// exatamente as de um pacote só.
//
// O `chi.Walk` devolve o padrão RESOLVIDO: o `base` juntado, o `r.Route` pai
// concatenado com o filho, a constante expandida. É o mesmo caminho do
// `TestNoRouteCarriesACoordinateInThePath`, e é o que faz o piso abaixo poder
// ser o número de rotas do processo em vez de um chute.
func TestAWireRouteStartsLowercase(t *testing.T) {
	routes := walkTheRouter(t, newTestServer(t))

	for route := range routes {
		// O método vem colado no padrão (`"POST /campanhas/…"`), e ele é
		// MAIÚSCULO por definição — o que se mede é o caminho.
		path := route
		if slot := strings.IndexByte(route, ' '); slot >= 0 {
			path = route[slot+1:]
		}
		for _, chunk := range strings.Split(strings.Trim(path, "/"), "/") {
			// `{id}` é parâmetro e segue o nome do campo, não a rota; `*` é o
			// curinga do chi.
			if chunk == "" || chunk == "*" || strings.HasPrefix(chunk, "{") {
				continue
			}
			// Qualquer maiúscula, e não só a primeira: em `password-Reset` o
			// segmento COMEÇA minúsculo e a capitalização caiu depois do hífen.
			// Um guarda que olhasse só a inicial passaria verde sobre ele.
			if strings.ContainsAny(chunk, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
				t.Errorf("a rota %q tem segmento em MAIÚSCULA (%q).\n"+
					"O cliente chama a grafia minúscula; capitalizar vira 404, e o\n"+
					"404 chega à tela como funcionalidade que sumiu.",
					path, chunk)
			}
		}
	}

	// O DENOMINADOR é EXATO porque vem do roteador, e o piso denuncia um
	// roteador que deixou de montar — que é como este guarda ficaria inerte.
	if len(routes) < 150 {
		t.Fatalf("guarda cego: só %d rotas no roteador", len(routes))
	}
}

func TestAWireTagStartsLowercase(t *testing.T) {
	tag := regexp.MustCompile(`json:"([^",]+)`)

	visited, found := 0, 0
	{
		var files []string
		root, err := os.Getwd()
		if err != nil {
			t.Fatalf("achar a raiz: %v", err)
		}
		if err := filepath.WalkDir(filepath.Dir(filepath.Dir(root)), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			files = append(files, path)
			return nil
		}); err != nil {
			t.Fatalf("caminhar a árvore: %v", err)
		}
		for _, name := range files {
			raw, err := os.ReadFile(name)
			if err != nil {
				t.Fatalf("ler %s: %v", name, err)
			}
			visited++
			for row, text := range strings.Split(string(raw), "\n") {
				// Comentário não é contrato. Sem isto o guarda lê o próprio texto
				// explicativo — que cita a tag defeituosa — e falha para sempre
				// sobre si mesmo. Achado ao provar o vermelho: ele já estava
				// vermelho ANTES da sabotagem, o que denunciou o autoexame.
				if cut := strings.Index(text, "//"); cut >= 0 {
					text = text[:cut]
				}
				for _, m := range tag.FindAllStringSubmatch(text, -1) {
					field := m[1]
					found++
					if field == "" || field == "-" {
						continue
					}
					if field[0] >= 'A' && field[0] <= 'Z' {
						t.Errorf("%s:%d: a tag `json:%q` começa em MAIÚSCULA.\n"+
							"O cliente lê a grafia minúscula; capitalizar entrega o campo como\n"+
							"`undefined` sem erro nenhum. Se isto veio de um rename automático,\n"+
							"a expressão casou dentro da string da tag — conserte a tag, não o guarda.",
							name, row+1, field)
					}
				}
			}
		}
	}

	// Ausência não é aprovação. O piso é a METADE das tags medidas quando o
	// guarda foi escrito: folga para o código encolher, e barulho na hora em que
	// o padrão parar de casar. Um guarda que varre zero arquivos passa verde
	// sobre nada.
	if visited == 0 || found < 300 {
		t.Fatalf("guarda cego: %d arquivos, %d tags — o padrão parou de casar", visited, found)
	}
}
