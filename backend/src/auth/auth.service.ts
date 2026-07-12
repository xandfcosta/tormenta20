import {
  ConflictException,
  Injectable,
  UnauthorizedException,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import * as bcrypt from 'bcrypt';
import { PrismaService } from '../prisma/prisma.service';
import { isPrismaUniqueViolation } from '../common/prisma-errors';
import {
  authUserSelect,
  type AuthUser,
  type JwtPayload,
} from './auth-user.type';

const BCRYPT_ROUNDS = 12;

export type { AuthUser, JwtPayload };

@Injectable()
export class AuthService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly jwt: JwtService,
  ) {}

  async register(input: {
    email: string;
    password: string;
    name?: string;
  }): Promise<AuthUser> {
    const existing = await this.prisma.user.findUnique({
      where: { email: input.email },
    });
    if (existing) {
      throw new ConflictException(`Email already registered: ${input.email}`);
    }
    const passwordHash = await bcrypt.hash(input.password, BCRYPT_ROUNDS);
    try {
      return await this.prisma.user.create({
        data: { email: input.email, passwordHash, name: input.name ?? null },
        select: authUserSelect,
      });
    } catch (err) {
      // Two registrations racing the same email pass the pre-check above
      // and collide on the unique index — translate P2002 to a domain
      // Conflict instead of leaking a 500.
      if (isPrismaUniqueViolation(err)) {
        throw new ConflictException(`Email already registered: ${input.email}`);
      }
      throw err;
    }
  }

  async validate(email: string, password: string): Promise<AuthUser> {
    const user = await this.prisma.user.findUnique({ where: { email } });
    if (!user) throw new UnauthorizedException('Invalid credentials');
    const ok = await bcrypt.compare(password, user.passwordHash);
    if (!ok) throw new UnauthorizedException('Invalid credentials');
    return { id: user.id, email: user.email, name: user.name };
  }

  signToken(user: AuthUser): string {
    const payload: JwtPayload = { sub: user.id, email: user.email };
    return this.jwt.sign(payload);
  }

  async findById(id: number): Promise<AuthUser | null> {
    return this.prisma.user.findUnique({
      where: { id },
      select: authUserSelect,
    });
  }
}
