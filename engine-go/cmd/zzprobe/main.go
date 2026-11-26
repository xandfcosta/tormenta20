package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"

	"t20engine/domain/catalog/transcript"
)

func main() {
	f, _ := os.Open("/tmp/claude-1000/unmeasured.txt")
	byName := map[string]transcript.Item{}
	for _, it := range transcript.Items {
		byName[it.Name] = it
	}
	type row struct {
		band, cat, name string
		page            int
		price           float64
	}
	var rows []row
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		n := sc.Text()
		it, ok := byName[n]
		if !ok {
			rows = append(rows, row{"??", "?", n, 0, 0})
			continue
		}
		band := "outro"
		switch {
		case it.BookPage >= 138 && it.BookPage <= 167:
			band = "cap3"
		case it.BookPage >= 324:
			band = "cap8"
		case it.BookPage == 0:
			band = "sem-pagina"
		}
		rows = append(rows, row{band, it.Category, n, it.BookPage, it.Price})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].band != rows[j].band {
			return rows[i].band < rows[j].band
		}
		if rows[i].cat != rows[j].cat {
			return rows[i].cat < rows[j].cat
		}
		return rows[i].page < rows[j].page
	})
	count := map[string]int{}
	for _, r := range rows {
		count[r.band+"/"+r.cat]++
	}
	keys := []string{}
	for k := range count {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%-40s %d\n", k, count[k])
	}
	fmt.Println("---")
	for _, r := range rows {
		if r.band != "cap8" {
			fmt.Printf("%s | %s | p%d | T$%g | %s\n", r.band, r.cat, r.page, r.price, r.name)
		}
	}
}
