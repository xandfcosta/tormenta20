import {
  BadRequestException,
  ConflictException,
  ForbiddenException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { SPELL_CATALOG } from '@tormenta20/t20-data';
import { PrismaService } from '../prisma/prisma.service';
import { CharactersService } from './characters.service';

/**
 * MVP prepared-list rule. Real T20 splits caster classes into two
 * families: **preparadas** (Clérigo, Druida, Arcanista/Mago) need to
 * flag which magias they hold for the day; **livres** (Bardo,
 * Arcanista/Bruxo, Arcanista/Feiticeiro, Paladino) cast anything they
 * know without prep. Detecting Arcanista path (Bruxo/Feiticeiro/Mago)
 * needs class-choice data we don't wire yet — deferred. Ship the
 * conservative version: any Clérigo or Druida level on the char
 * forces the prepared flag.
 */
const PREPARE_REQUIRING_CLASSES = new Set(['Clérigo', 'Druida']);

/**
 * PDF p171 — limite de PM por magia = ½ nível do conjurador, mínimo 1.
 * Item-based modifiers (`pmLimit` stat) live in the frontend derived
 * layer; keeping the backend authoritative check to the book base so
 * we can't be over-permissive without shipping the full stat compiler
 * server-side too.
 */
function pmLimitBase(characterLevel: number): number {
  return Math.max(1, Math.floor(characterLevel / 2));
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
      const code = (e as { code?: string })?.code;
      if (code === 'P2002') {
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

    const needsPrep = character.classes.some((c) =>
      PREPARE_REQUIRING_CLASSES.has(c.className),
    );
    if (needsPrep && !learned.prepared) {
      throw new ForbiddenException(
        `Spell "${catalogSpellId}" must be prepared before casting`,
      );
    }

    const augmentPm = this.validateAugments(spell, augmentPicks);
    const totalPm = spell.circle === 0 ? 0 : this.baseCircleCost(spell.circle) + augmentPm;

    const limit = pmLimitBase(character.level);
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

    if (totalPm === 0) {
      return character;
    }

    await this.prisma.character.update({
      where: { id: characterId },
      data: { mpCurrent: character.mpCurrent - totalPm },
    });
    return this.characters.findOne(ownerId, characterId);
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
