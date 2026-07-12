import {
  BadRequestException,
  ConflictException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { SPELL_CATALOG, getCatalogItem } from '@tormenta20/t20-data';
import type { Modifier, SpellSchool } from '@tormenta20/t20-data';
import { PrismaService } from '../prisma/prisma.service';
import { isPrismaUniqueViolation } from '../common/prisma-errors';
import { CharactersService } from './characters.service';

/**
 * Real T20 splits caster classes into two families:
 *  - **preparadas** need to flag which magias they hold for the day:
 *    Clérigo, Druida, Arcanista/Mago.
 *  - **livres** cast anything they know without prep:
 *    Bardo, Arcanista/Bruxo, Arcanista/Feiticeiro, Paladino.
 *
 * Clérigo/Druida are simple — any level triggers the flag. Arcanista
 * splits by `classChoices.Arcanista.caminho` (bruxo|feiticeiro|mago);
 * only the 'mago' path needs prep.
 */
const ALWAYS_PREPARE_CLASSES = new Set(['Clérigo', 'Druida']);

function parseClassChoices(raw: string | null | undefined): Record<
  string,
  { caminho?: string; devoto?: string }
> {
  if (!raw) return {};
  try {
    const parsed = JSON.parse(raw);
    return typeof parsed === 'object' && parsed !== null ? parsed : {};
  } catch {
    return {};
  }
}

function requiresPreparation(
  classes: readonly { className: string }[],
  classChoicesRaw: string | null | undefined,
): boolean {
  for (const c of classes) {
    if (ALWAYS_PREPARE_CLASSES.has(c.className)) return true;
  }
  const choices = parseClassChoices(classChoicesRaw);
  const arcanista = classes.find((c) => c.className === 'Arcanista');
  if (arcanista && choices.Arcanista?.caminho === 'mago') return true;
  return false;
}

/**
 * PDF p171 — limite de PM por magia = ½ nível do conjurador, mínimo 1.
 * Plus any item-based `pmLimit` modifiers on equipped items (catalog
 * base mods + improvements + material). Race/origin/class powers can
 * also target pmLimit in the frontend derived layer, but replicating
 * that full pipeline server-side would drag a large surface — we
 * pin the check to items since those flow through the CharacterItem
 * table and can be validated against the catalog authoritatively.
 */
function pmLimitFromItems(
  items: readonly {
    catalogId: string | null;
    equipped: string | null;
    improvements: string;
    material: string | null;
  }[],
): number {
  let total = 0;
  for (const it of items) {
    if (it.equipped === null || !it.catalogId) continue;
    const mods: Modifier[] = [];
    const base = getCatalogItem(it.catalogId);
    if (base?.modifiers) mods.push(...base.modifiers);
    let improvementIds: string[] = [];
    try {
      const parsed = JSON.parse(it.improvements ?? '[]');
      if (Array.isArray(parsed)) {
        improvementIds = parsed.filter(
          (v): v is string => typeof v === 'string',
        );
      }
    } catch {
      // Malformed JSON on the row — skip the improvements; the base
      // catalog mods still apply.
    }
    for (const id of improvementIds) {
      const imp = getCatalogItem(id);
      if (imp?.modifiers) mods.push(...imp.modifiers);
    }
    if (it.material) {
      const mat = getCatalogItem(it.material);
      if (mat?.modifiers) mods.push(...mat.modifiers);
    }
    for (const m of mods) {
      if (m.target.k === 'pmLimit') total += m.amount;
    }
  }
  return total;
}

function pmLimitFor(
  level: number,
  items: readonly {
    catalogId: string | null;
    equipped: string | null;
    improvements: string;
    material: string | null;
  }[],
): number {
  return Math.max(1, Math.floor(level / 2)) + pmLimitFromItems(items);
}

/**
 * Spellbook persistence — pairs a Character row to a set of SPELL_CATALOG
 * entries via the `CharacterSpell` join. Kept as its own service so
 * `characters.service.ts` (already 890 LOC) doesn't grow further while
 * the spell engine consumer lands in slices.
 *
 * Ownership is enforced by delegating to `CharactersService.findOne`
 * (which throws Forbidden if the caller doesn't own the character).
 */
@Injectable()
export class CharactersSpellsService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly characters: CharactersService,
  ) {}

  private assertSpellExists(catalogSpellId: string) {
    if (!SPELL_CATALOG[catalogSpellId]) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Unknown spell "${catalogSpellId}"`,
        fieldErrors: { catalogSpellId: ['Not in catalog'] },
      });
    }
  }

  /**
   * Add a spell to the character's known list. Duplicates 409 so the
   * frontend can distinguish "already learned" from other errors.
   */
  async learnSpell(
    ownerId: number,
    characterId: number,
    catalogSpellId: string,
  ) {
    await this.characters.findOne(ownerId, characterId);
    this.assertSpellExists(catalogSpellId);
    try {
      return await this.prisma.characterSpell.create({
        data: {
          characterId,
          catalogSpellId,
          prepared: false,
        },
      });
    } catch (e) {
      // P2002: unique(characterId, catalogSpellId) violated → already known.
      if (isPrismaUniqueViolation(e)) {
        throw new ConflictException({
          statusCode: 409,
          error: 'Conflict',
          message: `Spell "${catalogSpellId}" already known`,
          fieldErrors: { catalogSpellId: ['Already learned'] },
        });
      }
      throw e;
    }
  }

  /**
   * Idempotent unlearn — deleting an already-absent row is fine
   * (returns { count: 0 }) so the UI can fire without a pre-check.
   */
  async unlearnSpell(
    ownerId: number,
    characterId: number,
    catalogSpellId: string,
  ) {
    await this.characters.findOne(ownerId, characterId);
    const result = await this.prisma.characterSpell.deleteMany({
      where: { characterId, catalogSpellId },
    });
    return { catalogSpellId, removed: result.count };
  }

  /**
   * Toggle prepared state. Requires the spell to be learned; returning
   * NotFound (not 400) so the frontend can surface "aprenda primeiro".
   */
  async setSpellPrepared(
    ownerId: number,
    characterId: number,
    catalogSpellId: string,
    prepared: boolean,
  ) {
    await this.characters.findOne(ownerId, characterId);
    const row = await this.prisma.characterSpell.findUnique({
      where: {
        characterId_catalogSpellId: { characterId, catalogSpellId },
      },
    });
    if (!row) {
      throw new NotFoundException(
        `Spell "${catalogSpellId}" not in character's spellbook`,
      );
    }
    return this.prisma.characterSpell.update({
      where: { id: row.id },
      data: { prepared },
    });
  }

  /**
   * Cast a spell. Validates: learned, prepared (if any of the char's
   * caster classes require preparation), augments (existence, stack
   * rules, truque bans), pm cost ≤ per-spell limit, pm cost ≤ current
   * mp. On success debits `mpCurrent` by the total pm cost.
   *
   * Returns the updated character row so the frontend can update the
   * PM ticker in one hop.
   */
  async castSpell(
    ownerId: number,
    characterId: number,
    catalogSpellId: string,
    augmentPicks: readonly { augmentIndex: number; stacks: number }[],
  ) {
    const character = await this.characters.findOne(ownerId, characterId);
    const spell = SPELL_CATALOG[catalogSpellId];
    if (!spell) {
      throw new BadRequestException(
        `Unknown spell "${catalogSpellId}"`,
      );
    }

    const learned = await this.prisma.characterSpell.findUnique({
      where: {
        characterId_catalogSpellId: { characterId, catalogSpellId },
      },
    });
    if (!learned) {
      throw new NotFoundException(
        `Spell "${catalogSpellId}" not in character's spellbook`,
      );
    }

    const needsPrep = requiresPreparation(
      character.classes,
      character.classChoices,
    );
    if (needsPrep && !learned.prepared) {
      throw new ForbiddenException(
        `Spell "${catalogSpellId}" must be prepared before casting`,
      );
    }

    const augmentPm = this.validateAugments(spell, augmentPicks);
    const rawTotalPm = spell.circle === 0 ? 0 : this.baseCircleCost(spell.circle) + augmentPm;

    // Catalisador consumption: if the character has an active
    // scene-scoped effect that carries a `catalyst:<school>` modifier
    // matching this spell's school, apply the discount and mark the
    // effect for deletion after the cast succeeds.
    const catalyst = spell.circle === 0
      ? null
      : this.findCatalystForSchool(
          character.activeEffects ?? [],
          spell.school,
        );
    const catalystDiscount = catalyst?.amount ?? 0;
    const totalPm = Math.max(0, rawTotalPm + catalystDiscount);

    const limit = pmLimitFor(character.level, character.items ?? []);
    if (spell.circle > 0 && totalPm > limit) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `PM cost ${totalPm} exceeds per-spell limit ${limit}`,
        fieldErrors: { augments: [`Limite PM excedido (${limit})`] },
      });
    }

    if (totalPm > character.mpCurrent) {
      throw new BadRequestException({
        statusCode: 400,
        error: 'Bad Request',
        message: `Insufficient PM (need ${totalPm}, have ${character.mpCurrent})`,
        fieldErrors: { mpCurrent: [`Sem PM suficiente`] },
      });
    }

    // Consume the catalyst first (idempotent — if the delete fails we
    // still want the cast to proceed rather than double-charge PM).
    if (catalyst) {
      try {
        await this.prisma.activeEffect.delete({
          where: { id: catalyst.effectId },
        });
      } catch {
        // Effect vanished concurrently — safe to ignore.
      }
    }

    if (totalPm === 0) {
      return this.characters.findOne(ownerId, characterId);
    }

    await this.prisma.character.update({
      where: { id: characterId },
      data: { mpCurrent: character.mpCurrent - totalPm },
    });
    return this.characters.findOne(ownerId, characterId);
  }

  /**
   * Scan the character's ActiveEffect rows for a scene-scoped catalyst
   * whose modifier list contains a `catalyst:<school>` entry matching
   * the spell's school. Returns the first match (multiple catalysts of
   * the same school stack across days, but per T20 you consume one at
   * a time — first-in-first-out via the `id ASC` include order).
   */
  private findCatalystForSchool(
    activeEffects: readonly {
      id: number;
      scope: string;
      modifiers: string;
    }[],
    school: SpellSchool,
  ): { effectId: number; amount: number } | null {
    for (const effect of activeEffects) {
      if (effect.scope !== 'scene') continue;
      let mods: Modifier[] = [];
      try {
        const parsed = JSON.parse(effect.modifiers);
        if (Array.isArray(parsed)) mods = parsed as Modifier[];
      } catch {
        continue;
      }
      for (const m of mods) {
        if (m.target.k === 'catalyst' && m.target.school === school) {
          return { effectId: effect.id, amount: m.amount };
        }
      }
    }
    return null;
  }

  private baseCircleCost(circle: 0 | 1 | 2 | 3 | 4 | 5): number {
    return { 0: 0, 1: 1, 2: 3, 3: 6, 4: 10, 5: 15 }[circle];
  }

  private validateAugments(
    spell: (typeof SPELL_CATALOG)[string],
    picks: readonly { augmentIndex: number; stacks: number }[],
  ): number {
    if (picks.length === 0) return 0;
    if (spell.circle === 0) {
      throw new BadRequestException(
        `Truques cannot receive aprimoramentos`,
      );
    }
    const seenIndexes = new Set<number>();
    let total = 0;
    for (const p of picks) {
      if (
        !Number.isInteger(p.augmentIndex) ||
        p.augmentIndex < 0 ||
        p.augmentIndex >= spell.augments.length
      ) {
        throw new BadRequestException(
          `Invalid augmentIndex ${p.augmentIndex}`,
        );
      }
      if (seenIndexes.has(p.augmentIndex)) {
        throw new BadRequestException(
          `Duplicate augmentIndex ${p.augmentIndex} — combine stacks in one pick`,
        );
      }
      seenIndexes.add(p.augmentIndex);
      if (!Number.isInteger(p.stacks) || p.stacks < 1) {
        throw new BadRequestException(
          `stacks must be an integer ≥ 1 (got ${p.stacks})`,
        );
      }
      const a = spell.augments[p.augmentIndex];
      if (a.kind === 'muda' && p.stacks > 1) {
        throw new BadRequestException(
          `'muda' augment cannot stack (index ${p.augmentIndex})`,
        );
      }
      total += a.pmCost * p.stacks;
    }
    return total;
  }
}
