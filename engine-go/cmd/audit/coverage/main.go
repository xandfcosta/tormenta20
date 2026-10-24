// Command coverage diz o que o MOTOR aplica de cada verbete de raça, classe e
// origem (ALE-482).
//
//	go run ./cmd/audit/coverage              # os três perfis
//	go run ./cmd/audit/coverage classe       # um só
//	go run ./cmd/audit/coverage --destino    # só a contagem por issue
//
// Não é um auditor do LIVRO: confere o catálogo contra o MOTOR, e por isso não
// precisa do PDF.
//
// O denominador mora no DADO. A cobertura do capítulo 1 saiu errada quatro vezes
// por um instrumento que inferia o balde de fora; hoje o balde é DECLARADO no
// verbete, nos termos dos encantos (ALE-459):
//
//   - efeito no motor (`modifiers`, variante com `modifiers`, ou ativação com
//     `grant`) ou `modeled` → INTEIRO;
//   - só `powerUid` → APONTA: o balde é do verbete apontado. Contar o ponteiro
//     como efeito deu "inteiro" a 31 entradas cujo alvo não aplicava nada
//     (ALE-491);
//   - efeito E `unmodeled` → EM PARTE, com `modeled` obrigatório: sem ele a nota
//     parece falar do verbete inteiro;
//   - ativação que não é `passive` e não tem `grant` → GESTO;
//   - nada disso → SÓ NOTA, com `unmodeled` obrigatório.
//
// O DESTINO é a issue que a nota cita: agrupar por ela é o plano do capítulo. A
// nota que cita DUAS sai nas duas, então a soma dos grupos passa do número de
// notas; a que não cita issue sai em "sem issue" para ninguém a perder.
package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"

	"t20engine/domain/catalog/transcript"
)

var issueRef = regexp.MustCompile(`ALE-\d+`)

// originExpertise é a regra do benefício de PERÍCIA da origem, que não tem marca
// por verbete: são 94, e todos entram pela mesma forma — a regra vale para o
// TIPO, e mora aqui uma vez.
const originExpertise = "treina a perícia pelo modificador `trained` que o `originModifiers` gera (ALE-490)"

var profiles = []string{"raça", "classe", "origem"}

var bucketOrder = []string{"inteiro", "em parte", "gesto", "só nota", "aponta"}

// entry é o que o balde lê de um verbete, igual para os três perfis.
type entry struct {
	label, id     string
	hasModifiers  bool
	modeled       bool
	unmodeled     string
	powerUID      string
	variantEffect bool
}

var activations = func() map[string]transcript.Activation {
	out := map[string]transcript.Activation{}
	for _, a := range transcript.Activations {
		out[a.ID] = a
	}
	return out
}()

// bucket é o balde do verbete e a falha de marcação ("" quando não há).
func bucket(e entry) (string, string) {
	activation, hasActivation := activations[e.id]
	effect := e.hasModifiers || e.variantEffect || (hasActivation && activation.Grant != nil)
	if (effect || e.modeled) && e.unmodeled != "" {
		if !e.modeled {
			return "em parte", "aplica alguma coisa e não diz O QUÊ (`modeled`)"
		}
		return "em parte", ""
	}
	if effect || e.modeled {
		return "inteiro", ""
	}
	if e.powerUID != "" {
		return "aponta", ""
	}
	if hasActivation && activation.Kind != "passive" {
		return "gesto", ""
	}
	if e.unmodeled != "" {
		return "só nota", ""
	}
	return "só nota", "não tem efeito no motor NEM diz por quê (`unmodeled`)"
}

func entries(profile string) []entry {
	var out []entry
	switch profile {
	case "raça":
		for _, race := range transcript.RaceDefinitions {
			for _, a := range race.Abilities {
				variantEffect := false
				for _, v := range a.Variants {
					variantEffect = variantEffect || len(v.Modifiers) > 0
				}
				out = append(out, entry{race.Name + ": " + a.Name, a.ID, len(a.Modifiers) > 0,
					len(a.Modeled) > 0, a.Unmodeled, "", variantEffect})
			}
		}
	case "classe":
		for _, p := range transcript.ClassPowers {
			class, name := p.ClassName, p.Name
			if class == "" {
				class = "(genérico)"
			}
			if name == "" {
				name = p.ID
			}
			out = append(out, entry{class + ": " + name, p.ID, len(p.Modifiers) > 0,
				len(p.Modeled) > 0, p.Unmodeled, p.PowerUID, false})
		}
	case "origem":
		for _, o := range transcript.Origins {
			u := o.UniquePower
			out = append(out, entry{o.Name + ": " + u.Name, u.ID, len(u.Modifiers) > 0, false, u.Unmodeled, "", false})
		}
	}
	return out
}

type kindCount struct {
	kind string
	n    int
}

// originBenefitKinds conta os benefícios de origem por tipo — o denominador do
// resto —, do mais frequente ao menos, empate na ordem em que apareceram.
func originBenefitKinds() ([]kindCount, int) {
	var counts []kindCount
	total := 0
	for _, o := range transcript.Origins {
		for _, b := range o.Benefits {
			kind := b.Kind
			switch {
			case b.PowerUID != "":
				kind = "aponta para poder geral"
			case b.PowerPick != "":
				kind = "poder livre"
			}
			i := slices.IndexFunc(counts, func(c kindCount) bool { return c.kind == kind })
			if i < 0 {
				counts = append(counts, kindCount{kind, 0})
				i = len(counts) - 1
			}
			counts[i].n++
			total++
		}
	}
	sort.SliceStable(counts, func(i, j int) bool { return counts[i].n > counts[j].n })
	return counts, total
}

type noted struct {
	profile, label, bucket, reason string
}

func audit(profile string, byDestination map[string][]noted) int {
	count := map[string]int{}
	var failures []string
	list := entries(profile)
	for _, e := range list {
		b, failure := bucket(e)
		count[b]++
		if failure != "" {
			failures = append(failures, e.label+": "+failure)
		}
		if e.unmodeled == "" {
			continue
		}
		// Em TODA issue que ela cita: a do Lefou é tipo de criatura (ALE-469) e
		// efeito condicional (ALE-484), e cada fatia tem de achá-la.
		destinations := issueRef.FindAllString(e.unmodeled, -1)
		sort.Strings(destinations)
		destinations = slices.Compact(destinations)
		if len(destinations) == 0 {
			destinations = []string{"sem issue"}
		}
		for _, d := range destinations {
			byDestination[d] = append(byDestination[d], noted{profile, e.label, b, e.unmodeled})
		}
	}
	total := len(list)
	fmt.Printf("\n%s — %d verbetes\n", strings.ToUpper(profile), total)
	for _, b := range bucketOrder {
		fmt.Printf("  %-9s %4d\n", b, count[b])
	}
	if profile == "origem" {
		kinds, n := originBenefitKinds()
		parts := make([]string, len(kinds))
		for i, k := range kinds {
			parts[i] = fmt.Sprintf("%d %s", k.n, k.kind)
		}
		fmt.Printf("  e os %d benefícios: %s\n", n, strings.Join(parts, ", "))
		fmt.Printf("    perícia: %s\n", originExpertise)
	}
	fmt.Printf("  FALHAS: %d   (medidos: %d)\n", len(failures), total)
	for _, f := range failures {
		fmt.Printf("    %s\n", f)
	}
	return len(failures)
}

func main() {
	// À mão, e não pelo `flag`: os perfis são posicionais e podem vir antes ou
	// depois do `--destino`, e uma opção desconhecida é ignorada — como no
	// script que este substitui.
	var chosen []string
	onlyDestination := false
	for _, a := range os.Args[1:] {
		if a == "--destino" {
			onlyDestination = true
		}
		if !strings.HasPrefix(a, "--") {
			chosen = append(chosen, a)
		}
	}
	if len(chosen) == 0 {
		chosen = profiles
	}
	var unknown []string
	for _, p := range chosen {
		if !slices.Contains(profiles, p) {
			unknown = append(unknown, "'"+p+"'")
		}
	}
	if len(unknown) > 0 {
		fmt.Printf("não conheço [%s]; os perfis são raça, classe e origem\n", strings.Join(unknown, ", "))
		os.Exit(1)
	}

	byDestination := map[string][]noted{}
	failures := 0
	for _, p := range chosen {
		failures += audit(p, byDestination)
	}
	fmt.Println("\nPOR DESTINO — a nota de cada verbete, agrupada pela issue que ela cita")
	var destinations []string
	for d := range byDestination {
		destinations = append(destinations, d)
	}
	sort.Slice(destinations, func(i, j int) bool {
		li, lj := destinations[i] == "sem issue", destinations[j] == "sem issue"
		if li != lj {
			return lj
		}
		return destinations[i] < destinations[j]
	})
	for _, d := range destinations {
		lines := byDestination[d]
		fmt.Printf("\n  %s: %d\n", d, len(lines))
		if onlyDestination {
			continue
		}
		for _, l := range lines {
			fmt.Printf("    [%s] %s  (%s)\n", l.profile, l.label, l.bucket)
			fmt.Printf("        %s\n", l.reason)
		}
	}
	if failures > 0 {
		os.Exit(1)
	}
}
