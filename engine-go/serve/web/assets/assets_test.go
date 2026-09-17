package assets

import (
	"strings"
	"testing"
)

// O dígito decide invalidação de cache: se variasse entre dois boots do MESMO
// binário, todo reinício jogaria fora o cache de todo mundo — e o clarão da
// folha rebaixada voltaria uma vez por deploy, sem ninguém entender por quê.
func TestTheDigestIsStableBetweenReads(t *testing.T) {
	if a, b := digest(), digest(); a != b {
		t.Errorf("o dígito mudou entre duas leituras: %q e %q", a, b)
	}
	if len(version) == 0 {
		t.Fatal("o dígito saiu vazio: toda URL versionada viraria `?v=` e o cache nunca casaria")
	}
	// E ele tem de VIR do conteúdo: um dígito constante casaria com ele mesmo
	// para sempre e serviria folha velha como imutável depois de um deploy.
	if strings.Trim(version, "0") == "" {
		t.Errorf("o dígito é %q, que não parece hash de conteúdo", version)
	}
}

// O CONTROLE de que o embed alcançou alguma coisa: uma árvore vazia daria um
// dígito estável e um `Handler` que responde 404 para tudo, e os dois passariam
// calados.
func TestTheEmbeddedTreeHasTheFilesThePageAsksFor(t *testing.T) {
	for _, file := range []string{"static/app.css", "static/scene.js", "static/favicon.svg"} {
		if _, err := files.ReadFile(file); err != nil {
			t.Errorf("%s não está embutido: %v", file, err)
		}
	}
}

func TestTheURLCarriesTheVersion(t *testing.T) {
	if got := URL("scene.js"); got != "/static/scene.js?v="+version {
		t.Errorf("URL(scene.js) = %q", got)
	}
}
