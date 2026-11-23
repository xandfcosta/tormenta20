package engine

// carismaLossFromPowers mirrors tormenta.ts: the N-th power costs
// 1 + floor((N-1)/2) Carisma. Sequence: 1, 2, 4, 6, 9, 12…
func carismaLossFromPowers(powerCount int) int {
	total := 0
	for k := 1; k <= powerCount; k++ {
		total += 1 + (k-1)/2
	}
	return total
}
