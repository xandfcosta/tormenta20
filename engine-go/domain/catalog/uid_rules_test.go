package catalog_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"t20engine/domain/catalog"
)

// O `uid` É A IDENTIDADE QUE NÃO MUDA (ALE-402).
//
// O `id` do catálogo é o nome em kebab-case, e ele é ótimo para o que é
// público: a URL o mostra, o `grep` o acha, a fixture se lê. O que ele não é é
// estável — renomear o verbete renomeia o id, e o banco guarda referência em
// dezessete colunas de nove tabelas.
//
// O `uid` fica com o papel de identidade. Ele é opaco DE PROPÓSITO: um
// identificador derivável do nome volta a mudar quando alguém "corrige" a
// derivação, e aí ele não identifica nada.
//
// Estes três casos são o que faz a promessa valer. Sem eles o `uid` é só mais
// um campo, e um campo que ninguém confere é um campo que apodrece.

const baseline = "testdata/uid_baseline.txt"

type verbete struct {
	arquivo    string
	identidade string
	uid        string
}

// TestEveryCatalogEntryCarriesAUid — cobertura, medida pela BASE e não por uma
// segunda tabela de escopo.
//
// Quais arquivos e quais caminhos têm verbete é declarado uma vez só, no
// `scripts/mint-uids.py`. Repetir essa tabela aqui criaria exatamente a segunda
// verdade que o uid existe para remover — e ela já divergiu uma vez: a versão
// anterior deste caso usava "objeto com nome e descrição", e reprovou 126 itens
// que a cunhagem não alcançava porque `"modifiers": []` é falso em Python e
// presente em Go.
//
// Aqui a pergunta é outra e não precisa de escopo: TODO par da base ainda tem
// uid, e ele é o mesmo. Um verbete que perdesse o campo apareceria com uid
// vazio.
func TestEveryCatalogEntryCarriesAUid(t *testing.T) {
	todos := lerVerbetes(t)
	if len(todos) < 1400 {
		t.Fatalf("só %d verbetes com uid — eram 1506 quando isto foi escrito, e o "+
			"caso não estaria medindo nada", len(todos))
	}
}

// TestNoUidNamesTwoDifferentThings — unicidade. Um uid que aponte para dois
// conceitos é pior que nenhum: ele faz o apontamento resolver para o errado em
// silêncio.
//
// Repetir o MESMO uid em linhas do mesmo conceito é esperado e não é erro, e
// isso acontece de duas formas:
//
//   - dentro de um arquivo — o `divine-powers.json` tem 80 linhas para 72
//     poderes, porque "Coragem Total" aparece uma vez por deus que a concede;
//   - ENTRE arquivos — `origins.json` (benefícios derivados) e
//     `origins-source.json` (a transcrição do livro) descrevem as mesmas 35
//     origens, com campos diferentes. Uma origem, um uid.
//
// Por isso a comparação é pelo NOME normalizado e não pelo caminho do arquivo:
// `origins.json/Acólito` e `origins-source.json/acolito` são a mesma coisa, e
// tratá-los como dois faria o guarda reprovar o desenho que ele existe para
// proteger.
func TestNoUidNamesTwoDifferentThings(t *testing.T) {
	todos := lerVerbetes(t)
	identidades := map[string]map[string]bool{}
	for _, v := range todos {
		if identidades[v.uid] == nil {
			identidades[v.uid] = map[string]bool{}
		}
		identidades[v.uid][chaveDoConceito(v.identidade)] = true
	}
	for uid, quais := range identidades {
		if len(quais) == 1 {
			continue
		}
		nomes := make([]string, 0, len(quais))
		for q := range quais {
			nomes = append(nomes, q)
		}
		sort.Strings(nomes)
		t.Errorf("o uid %q nomeia %d conceitos diferentes: %s", uid, len(quais),
			strings.Join(nomes, ", "))
	}
}

// TestNoUidEverChanges — a promessa inteira em uma frase.
//
// A linha de base guarda o par (identidade pública → uid) de cada verbete que
// já existia. Se um uid mudar, o banco perde a referência: uma ficha deixa de
// achar o poder dela, sem erro em lugar nenhum.
//
// Verbete NOVO não reprova — ele nasce com uid próprio e entra na base quando
// alguém a regerar. O que reprova é um uid EXISTENTE virar outro, que é
// exatamente o que a cunhagem se recusa a fazer e este caso confirma.
func TestNoUidEverChanges(t *testing.T) {
	todos := lerVerbetes(t)
	agora := map[string]string{}
	for _, v := range todos {
		agora[v.arquivo+"\t"+v.identidade] = v.uid
	}

	f, err := os.Open(baseline)
	if err != nil {
		t.Fatalf("linha de base: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("fechar a linha de base: %v", err)
		}
	}()

	conferidos, sumiram := 0, 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		partes := strings.Split(scanner.Text(), "\t")
		if len(partes) != 3 {
			continue
		}
		chave, queria := partes[0]+"\t"+partes[1], partes[2]
		tem, existe := agora[chave]
		if !existe {
			sumiram++
			t.Errorf("%s/%s perdeu o `uid`, ou saiu do catálogo. As duas caem aqui "+
				"porque o caso só enxerga quem TEM uid: se o verbete ainda existe, "+
				"rode `python3 scripts/mint-uids.py --aplicar`; se foi apagado de "+
				"propósito, tire a linha da base", partes[0], partes[1])
			continue
		}
		conferidos++
		if tem != queria {
			t.Errorf("%s/%s trocou de uid: era %q e agora é %q. O uid é a identidade "+
				"que o BANCO guarda — trocá-lo faz a ficha perder o verbete sem erro "+
				"nenhum", partes[0], partes[1], queria, tem)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("ler a linha de base: %v", err)
	}
	// O CONTROLE: a base tinha o que conferir. Um arquivo vazio, ou um caminho
	// errado, passaria verde sobre nada.
	if conferidos+sumiram < 1000 {
		t.Fatalf("a linha de base rendeu %d pares e ela tinha 1017 quando isto foi "+
			"escrito — o arquivo não foi lido", conferidos+sumiram)
	}
}

func lerVerbetes(t *testing.T) (todos []verbete) {
	t.Helper()
	arquivos, err := filepath.Glob("data/*.json")
	if err != nil || len(arquivos) == 0 {
		t.Fatalf("não achei os catálogos em data/*.json: %v", err)
	}
	for _, caminho := range arquivos {
		raw, ok := catalog.Resource(strings.TrimSuffix(filepath.Base(caminho), ".json"))
		if !ok {
			continue
		}
		var arvore any
		if json.Unmarshal(raw, &arvore) != nil {
			continue
		}
		for _, no := range comUid(arvore) {
			todos = append(todos, verbete{
				arquivo:    filepath.Base(caminho),
				identidade: identidadeDe(no),
				uid:        no["uid"].(string),
			})
		}
	}
	return todos
}

// comUid: todo objeto que TEM uid, onde quer que esteja na árvore. Não há
// predicado de escopo aqui de propósito — quem decide o escopo é a cunhagem, e
// o guarda confere o que ela produziu contra a base.
func comUid(no any) []map[string]any {
	fora := []map[string]any{}
	switch v := no.(type) {
	case map[string]any:
		if _, tem := v["uid"].(string); tem {
			fora = append(fora, v)
		}
		chaves := make([]string, 0, len(v))
		for k := range v {
			chaves = append(chaves, k)
		}
		sort.Strings(chaves)
		for _, k := range chaves {
			fora = append(fora, comUid(v[k])...)
		}
	case []any:
		for _, item := range v {
			fora = append(fora, comUid(item)...)
		}
	}
	return fora
}

// chaveDoConceito normaliza a identidade pública para comparar COISAS: sem
// acento e sem caixa, porque o mesmo conceito aparece como "Acólito" num
// arquivo e "acolito" no outro.
func chaveDoConceito(identidade string) string {
	semAcento := norm.NFD.String(strings.ToLower(identidade))
	var b strings.Builder
	for _, r := range semAcento {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if r == '-' || r == ' ' || r == '_' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func identidadeDe(no map[string]any) string {
	if id, ok := no["id"].(string); ok {
		return id
	}
	nome, _ := no["name"].(string)
	return nome
}
