package engine

// VitalContextFor builds the VitalContext the front assembles in enginePools:
// parsed ownership blobs + attrTotals from the base item/race effects (no
// conditionals). The server-side vitals recompute (backend buildVitalContext +
// vitals-sync) uses it to re-derive PV/PM máximos after a level/class change.
func (r *Ruleset) VitalContextFor(ch Character) VitalContext {
	// Sem as somas de atributo: elas não miram vital nenhum, e resolvê-las aqui
	// seria uma passada a mais para nada.
	//
	// Sem a FORMA SELVAGEM: ela é fonte da coleta e muda Força e Destreza, e o máximo de PV e PM é o do
	// druida, que o animal não toca (p62).
	ch.WildShape = nil
	plain, _ := splitAttributeSums(r.ActiveItemsFor(ch))
	effects := resolveInWorld(plain, nil)
	attrTotals := map[string]int{}
	for _, a := range AttributeKeys {
		attrTotals[a] = effectiveAttribute(ch, a, effects)
	}
	level := 0
	classes := make([]ClassEntry, len(ch.Classes))
	for i, ce := range ch.Classes {
		classes[i] = ClassEntry{ClassName: ce.ClassName, Level: ce.Level}
		level += ce.Level
	}
	raceID := ""
	if len(ch.Races) > 0 {
		raceID = ch.Races[0].Race
	}
	return VitalContext{
		Level:              level,
		Classes:            classes,
		RaceID:             raceID,
		RaceAbilityChoices: parseChoiceSet(ch.RaceAbilityChoices).list,
		RaceAncestry:       parseRaceAttributeChoices(ch.RaceAttributeChoices).ancestry,
		PowerIDs:           parseChoiceSet(ch.ClassPowers).list,
		ClassChoices:       parseClassChoices(ch.ClassChoices),
		GodPowers:          parseChoiceSet(ch.GodPowers).list,
		Origin:             ch.Origin,
		OriginChoices:      parseChoiceSet(ch.OriginChoices).list,
		AttrTotals:         attrTotals,
		ItemGrants:         append(r.itemVitalGrants(ch), r.ruleVitalGrants(ch)...),
	}
}

// VitalsForCharacter re-derives PV/PM máximos for a raw Character — the API's
// server-side vitals recompute after a level or class-level change.
func (r *Ruleset) VitalsForCharacter(ch Character) VitalPools {
	return r.ComputeVitals(r.VitalContextFor(ch))
}
