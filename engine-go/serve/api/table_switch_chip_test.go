package api

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"
)

// O CHIP DA MESA é o mesmo gesto do interruptor da aba Efeitos, e o sinal que ele escreve é o que o
// servidor lê. Escrito com outro nome, todo chip de situação era recusado com "o gesto não disse
// qual efeito situacional alternar", e a recusa só existe na resposta.
func TestTheSwitchChipAtTheTableSendsTheSignalTheServerReads(t *testing.T) {
	f, id, _ := barbarianOnTurn(t)
	seedPowerChoices(t, f.s, id, `["estilo-de-duas-armas"]`, "{}")
	chips := acoesDaMesa(t, f)
	found := regexp.MustCompile(`\$(\w+) = (?:&#34;|&quot;|")flag:estilo-de-duas-armas`).FindStringSubmatch(chips)
	if found == nil {
		t.Fatal("as Ações não têm o chip do Estilo de Duas Armas")
	}
	target := fmt.Sprintf("/personagens/%d/efeitos/situacao", id)
	body := fmt.Sprintf(`{%q:"flag:estilo-de-duas-armas"}`, found[1])
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, body).Body.String()); refusal != "" {
		t.Errorf("o chip escreve $%s, e o servidor recusou: %q", found[1], refusal)
	}
}
