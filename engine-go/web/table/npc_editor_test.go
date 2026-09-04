package table

import (
	"encoding/json"
	"strings"
	"testing"
)

// Os guardas do EDITOR DE BLOCO (ALE-269).
//
// A forma do bloco tem guarda em `creature/block.go` (a validação) e a cópia do
// livro tem o oráculo do `piloto_verbete_para_bloco`. O que se prende aqui é o
// que só existe DESDE o editor: que o rascunho atravessa os gestos de forma sem
// perder o que estava digitado, que a AUSÊNCIA de mana sobrevive à ida e à volta
// pelo formulário, que a recusa fala DENTRO do editor, e que o id do rascunho não
// alcança o elenco de outra campanha.

// bodyDraft monta o corpo de sinais que o navegador mandaria.
//
// Escrito como TEXTO e não `json.Marshal` de um `npcDraft`: marshalar a
// struct faria o teste mandar exatamente o que o servidor espera, e um campo
// renomeado passaria verde nos dois lados. Aqui o teste fala a língua do FIO.
func bodyDraft(dentro string) string {
	return `{"rascunho":{` + dentro + `}}`
}

const blocoMinimo = `"nd":1,"tipo":"humanoide","size":"medio","hp":10,"defesa":10,` +
	`"deslocamento":"9m (6q)","attacks":[],"skills":[],"specialAbilities":[]`

// responseDraft extrai o rascunho do quadro de sinais do SSE.
//
// Ler a CHAVE e não procurar o texto solto na resposta, e isto custou uma
// sabotagem para descobrir: `Contains(resposta, "Ogro Capitão")` passa verde com
// o sinal renomeado, porque o nome continua no corpo — ligado a coisa nenhuma. O
// que a tela precisa é do valor sob `rascunho`, e é isso que se afirma.
func responseDraft(t *testing.T, resposta string) map[string]any {
	t.Helper()
	const marca = "data: signals "
	i := strings.Index(resposta, marca)
	if i < 0 {
		t.Fatalf("a resposta não trouxe sinais:\n%s", resposta)
	}
	linha := resposta[i+len(marca):]
	if fim := strings.IndexByte(linha, '\n'); fim >= 0 {
		linha = linha[:fim]
	}
	var sinais struct {
		Rascunho map[string]any `json:"rascunho"`
	}
	if err := json.Unmarshal([]byte(linha), &sinais); err != nil {
		t.Fatalf("os sinais não são JSON: %v\n%s", err, linha)
	}
	if sinais.Rascunho == nil {
		t.Fatalf("a resposta não trouxe `rascunho`:\n%s", linha)
	}
	return sinais.Rascunho
}
