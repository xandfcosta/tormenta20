package book

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// O LIVRO NÃO CONHECE NEM O HTTP NEM A TELA (ALE-278).
//
// Irmão dos `fronteira_test.go` do `aovivo`, do `tabuleiro`, da `plataforma`, do
// `events`, do `sheet` e do `creature`. Este é o mais importante da série, e a
// razão é aritmética: TREZE famílias de arquivo leem o livro, então ele vai ser
// importado por quase todo pacote de cena que nascer.
//
// No dia em que ele importar o `api`, todas as cenas alcançam HTTP de graça — e
// o guarda de fronteira de cada uma continua VERDE, porque cada guarda só olha
// os imports dele. É a mesma armadilha que o `events` documenta, com treze vezes
// mais alcance.
//
// # A lista, e por que cada um está nela
var permitidos = map[string]bool{
	// O catálogo é a FONTE: o livro tipa o que o `catalog/data/*.json` guarda.
	"t20engine/catalog": true,
	// Os tipos do motor aparecem nas entradas que carregam regra — o item que
	// tem efeito, a magia que tem custo.
	"t20engine/engine": true,
	// O bloco de criatura é forma de dado e já é folha (ver o guarda de lá).
	"t20engine/creature": true,
	// A busca é FOLHA também, e ela entrou aqui apagando uma cópia: o `Fold`
	// que desacentua morava em dois lugares porque o livro não podia importar o
	// `api`, e a segunda cópia estava errada (ver `search`). Um import de um
	// pacote sem dependência nenhuma não alarga fronteira de ninguém.
	"t20engine/search": true,
}

func TestTheBookKnowsNoHTTPAndNoScreen(t *testing.T) {
	arquivos, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ler o pacote: %v", err)
	}

	conjunto := token.NewFileSet()
	visitados := 0
	for _, entrada := range arquivos {
		nome := entrada.Name()
		if !strings.HasSuffix(nome, ".go") {
			continue
		}
		visitados++
		arquivo, err := parser.ParseFile(conjunto, nome, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ler %s: %v", nome, err)
		}
		for _, imp := range arquivo.Imports {
			caminho := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(caminho, "t20engine/") || permitidos[caminho] {
				continue
			}
			t.Errorf("%s importa %q — o livro é consultado por TREZE famílias.\n"+
				"Um import daqui dá %q a todas elas de graça, e o guarda de fronteira\n"+
				"de cada uma continua verde porque só olha os imports dela.",
				nome, caminho, caminho)
		}
	}

	// Sem isto, apagar o pacote deixaria o guarda VERDE — ausência lida como
	// aprovação, que é a família que o `CLAUDE.md` da raiz descreve.
	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}

// A CHAVE DE ENDEREÇO desacentua, e este caso existe porque ela parou de fazer
// isso no meio da extração.
//
// O `KeyOfName` monta o endereço que um ELO aponta: a classe cita "Atuação" e o
// link tem de ir para `?entrada=atuacao`. Durante a mudança, a cópia local do
// desacentuador foi escrita chamando a função errada — a que só faz `ToLower` —
// e o resultado foi "atuação", com acento. Nenhum erro, nenhum panic: um elo
// apontando para um endereço que não existe.
//
// Quem pegou foi um teste de CENA, dois pacotes acima. Este aqui prende a regra
// onde ela mora, que é o que o guia chama de "uma regra, uma camada".
func TestTheAddressKeyDropsAccents(t *testing.T) {
	casos := map[string]string{
		"Atuação":      "atuacao",
		"Anão":         "anao",
		"Luta":         "luta",
		"Jogo de Sina": "jogo-de-sina",
	}
	for nome, esperado := range casos {
		if obtido := KeyOfName(nome); obtido != esperado {
			t.Errorf("KeyOfName(%q) = %q, esperado %q", nome, obtido, esperado)
		}
	}
}
