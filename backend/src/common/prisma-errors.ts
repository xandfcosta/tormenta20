/**
 * Structural detection of Prisma's `P2002` unique-constraint violation.
 * Kept off `@prisma/client` runtime types so callers don't couple to the
 * generated client — a shared kernel helper for translating a raced
 * insert into a domain `ConflictException`.
 *
 * @example
 * try { await prisma.user.create(...) }
 * catch (err) { if (isPrismaUniqueViolation(err)) throw new ConflictException(...) }
 */
export function isPrismaUniqueViolation(err: unknown): boolean {
  return (
    typeof err === 'object' &&
    err !== null &&
    'code' in err &&
    (err as { code: unknown }).code === 'P2002'
  );
}
