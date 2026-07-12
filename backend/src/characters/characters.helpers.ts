import { BadRequestException } from '@nestjs/common';
import {
  CAMINHOS,
  DEUS_BY_ID,
  familyFor,
  getCatalogItem,
  type ClassChoiceBlob,
  type ClassChoices,
} from '@tormenta20/t20-data';
import type { CreateCharacterDto } from './dto/character.dto';

/**
 * Pure helpers extracted from CharactersService so they can be unit-tested
 * without a Prisma instance. Keep this file dependency-free (no @Injectable,
 * no class properties). Service methods call them directly.
 */

export function assertCharacterRules(dto: CreateCharacterDto): void {
  const fieldErrors: Record<string, string[]> = {};
  if (dto.hpCurrent > dto.hpMax) {
    fieldErrors.hpCurrent = ['HP current cannot exceed HP max'];
  }
  if (dto.mpCurrent > dto.mpMax) {
    fieldErrors.mpCurrent = ['MP current cannot exceed MP max'];
  }
  const seen = new Set<string>();
  dto.classes.forEach((entry, index) => {
    if (seen.has(entry.className)) {
      fieldErrors[`classes.${index}.className`] = [
        `Class "${entry.className}" already added — combine levels in one entry instead`,
      ];
    } else {
      seen.add(entry.className);
    }
  });
  if (Object.keys(fieldErrors).length === 0) return;
  throw new BadRequestException({
    statusCode: 400,
    error: 'Bad Request',
    message: 'Validation failed',
    fieldErrors,
  });
}

/**
 * Validate a classChoices blob against the t20-data catalogs:
 *  - `devoto` value must be a known deus id.
 *  - `caminho` value must be one of CAMINHOS[className].
 * Bogus keys (unknown className) are accepted silently because the frontend
 * only proposes legal keys; they become inert at evaluation time.
 */
export function sanitizeClassChoices(input: unknown): ClassChoices {
  if (!input || typeof input !== 'object') return {};
  const fieldErrors: Record<string, string[]> = {};
  const out: ClassChoices = {};
  for (const [className, raw] of Object.entries(input)) {
    if (!raw || typeof raw !== 'object') continue;
    const blob: ClassChoiceBlob = {};
    const r = raw as { devoto?: unknown; caminho?: unknown };
    if (r.devoto !== undefined && r.devoto !== '') {
      if (typeof r.devoto !== 'string' || !DEUS_BY_ID[r.devoto]) {
        (fieldErrors[`classChoices.${className}.devoto`] ??= []).push(
          `Unknown deus id "${String(r.devoto)}"`,
        );
      } else {
        blob.devoto = r.devoto;
      }
    }
    if (r.caminho !== undefined && r.caminho !== '') {
      const options = CAMINHOS[className];
      if (
        typeof r.caminho !== 'string' ||
        !options?.some((c) => c.id === r.caminho)
      ) {
        (fieldErrors[`classChoices.${className}.caminho`] ??= []).push(
          `Caminho "${String(r.caminho)}" not valid for ${className}`,
        );
      } else {
        blob.caminho = r.caminho;
      }
    }
    if (blob.devoto || blob.caminho) out[className] = blob;
  }
  if (Object.keys(fieldErrors).length > 0) {
    throw new BadRequestException({
      statusCode: 400,
      error: 'Bad Request',
      message: 'Validation failed',
      fieldErrors,
    });
  }
  return out;
}

export function assertSlotsMultiple(slots: number): void {
  if (!Number.isFinite(slots) || !Number.isInteger(slots * 2)) {
    throw new BadRequestException({
      statusCode: 400,
      error: 'Bad Request',
      message: 'Validation failed',
      fieldErrors: { slots: ['Slots must be a multiple of 0.5'] },
    });
  }
}

/**
 * Validate that every improvement / material id refers to a known catalog
 * overlay AND that its `appliesTo` family list includes the base item's
 * family. Custom items (no catalogId) cannot have overlays.
 */
export function assertOverlaysCompatible(
  catalogId: string | null | undefined,
  improvements: string[] | undefined,
  material: string | null | undefined,
): void {
  const hasOverlays =
    (improvements && improvements.length > 0) ||
    (material !== undefined && material !== null && material !== '');
  if (!hasOverlays) return;
  if (!catalogId) {
    throw new BadRequestException({
      statusCode: 400,
      error: 'Bad Request',
      message: 'Validation failed',
      fieldErrors: {
        improvements: ['Custom items cannot have overlays'],
      },
    });
  }
  const base = getCatalogItem(catalogId);
  if (!base) {
    throw new BadRequestException(`Unknown base item "${catalogId}"`);
  }
  if (
    base.category === 'consumable' ||
    base.category === 'meal' ||
    base.category === 'catalyst' ||
    base.category === 'animal' ||
    base.category === 'vehicle'
  ) {
    throw new BadRequestException({
      statusCode: 400,
      error: 'Bad Request',
      message: 'Validation failed',
      fieldErrors: {
        improvements: [`${base.category} items cannot have overlays`],
      },
    });
  }
  const family = familyFor(base);
  const fieldErrors: Record<string, string[]> = {};
  for (const id of improvements ?? []) {
    const overlay = getCatalogItem(id);
    if (!overlay || overlay.category !== 'improvement') {
      (fieldErrors.improvements ??= []).push(`Unknown improvement "${id}"`);
      continue;
    }
    if (!overlay.appliesTo?.some((f) => f === 'any' || f === family)) {
      (fieldErrors.improvements ??= []).push(
        `"${overlay.name}" does not apply to ${family}`,
      );
    }
  }
  if (material) {
    const overlay = getCatalogItem(material);
    if (!overlay || overlay.category !== 'material') {
      (fieldErrors.material ??= []).push(`Unknown material "${material}"`);
    } else if (!overlay.appliesTo?.some((f) => f === 'any' || f === family)) {
      (fieldErrors.material ??= []).push(
        `"${overlay.name}" does not apply to ${family}`,
      );
    }
  }
  if (Object.keys(fieldErrors).length > 0) {
    throw new BadRequestException({
      statusCode: 400,
      error: 'Bad Request',
      message: 'Validation failed',
      fieldErrors,
    });
  }
}

/**
 * Deterministic average roll used for AI-rolled consumables when the client
 * does not supply a player-rolled value. Treats `NdF` as `N * (F+1)/2` —
 * the arithmetic mean of the dice, not a book rule (so there's no page to
 * cite); it's just the no-roll fallback value. Accepts plain integers
 * ('0', '5') as a flat-bonus shorthand.
 */
export function rollAverage(dice: string, bonus = 0): number {
  const trimmed = dice.trim();
  if (trimmed === '' || trimmed === '0') return bonus;
  const flat = Number(trimmed);
  if (!Number.isNaN(flat)) return flat + bonus;
  const match = /^(\d+)d(\d+)$/i.exec(trimmed);
  if (!match) {
    throw new BadRequestException(`Invalid dice expression: "${dice}"`);
  }
  const n = Number(match[1]);
  const f = Number(match[2]);
  return Math.floor((n * (f + 1)) / 2) + bonus;
}
