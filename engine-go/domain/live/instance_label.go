package live

// COMO SE NUMERA O SEGUNDO OGRO.
//
// Mora aqui e não no tabuleiro porque as DUAS superfícies numeram: a fila da
// iniciativa e o mapa. Uma regra, dois consumidores — e o tabuleiro importa o
// regime porque ele só existe DENTRO de uma sessão ao vivo, nunca o contrário.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// instanceSuffix é o número que a mesa usa para contar iguais: "Zumbi 3".
// O número no MEIO do nome ("Recruta Nv1 Simples") NÃO é instância — separar
// errado faria a cópia nascer com um nome que o desenho colore como outra
// espécie.
var instanceSuffix = regexp.MustCompile(`^(.*\S)\s+(\d{1,3})$`)

// speciesOf separa a espécie do número da instância.
func speciesOf(label string) (string, int) {
	match := instanceSuffix.FindStringSubmatch(strings.TrimSpace(label))
	if match == nil {
		return strings.TrimSpace(label), 0
	}
	numero, err := strconv.Atoi(match[2])
	if err != nil {
		return strings.TrimSpace(label), 0
	}
	return match[1], numero
}

// NextInstanceLabelAmong é a regra sozinha, sobre uma lista de rótulos — o
// tabuleiro passa as peças e a FILA passa os combatentes. Extraída porque
// adicionar quatro ogros à iniciativa tem exatamente o mesmo problema que
// duplicar um zumbi no mapa, e resolvê-lo duas vezes é como as duas telas passam
// a numerar diferente.
func NextInstanceLabelAmong(usados []string, label string) string {
	especie, _ := speciesOf(label)
	ocupados := map[int]bool{}
	for _, outro := range usados {
		outra, numero := speciesOf(outro)
		if outra != especie {
			continue
		}
		if numero == 0 {
			numero = 1 // quem está sem número ocupa o 1
		}
		ocupados[numero] = true
	}
	for numero := 1; ; numero++ {
		if !ocupados[numero] {
			return fmt.Sprintf("%s %d", especie, numero)
		}
	}
}

// Species separa a espécie do número da instância, para quem DESENHA.
//
// Exportada porque o desenho da peça precisa da MESMA separação que a numeração:
// a cor e as letras vêm da espécie, e o número vira selo. Derivá-la uma segunda
// vez é como "Zumbi 1" e "Zumbi 2" — a mesma criatura — passam a sair em cores
// sem relação nenhuma, perdendo justamente o número, que é a única coisa que
// distingue as três peças na mesa.
//
// Devolve 0 quando não há sufixo. "Recruta Nv1 Simples" não tem instância: o
// número está no MEIO do nome, e separar por qualquer dígito faria a cópia
// nascer como outra espécie.
func Species(rotulo string) (string, int) { return speciesOf(rotulo) }
