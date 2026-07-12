/**
 * Identity contract shared across bounded contexts — the shape of the
 * authenticated user and the JWT payload. Lives outside `auth.service`
 * so consumers (controllers, the WS gateway) can name the current user
 * without importing the service and dragging its dependency graph along.
 */
export type AuthUser = {
  id: number;
  email: string;
  name: string | null;
};

export type JwtPayload = {
  sub: number;
  email: string;
};

/**
 * Prisma `select` that projects exactly an {@link AuthUser}. One source
 * of truth for the shape — reuse it wherever a user is read as identity
 * so the projection can't drift field-by-field across call sites.
 */
export const authUserSelect = {
  id: true,
  email: true,
  name: true,
} as const;
