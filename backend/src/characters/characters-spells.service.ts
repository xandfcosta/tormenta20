import {
  BadRequestException,
  ConflictException,
  Injectable,
  NotFoundException,
} from '@nestjs/common';
import { SPELL_CATALOG } from '@tormenta20/t20-data';
import { PrismaService } from '../prisma/prisma.service';
import { CharactersService } from './characters.service';

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
}
