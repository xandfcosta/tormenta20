import {
  ConflictException,
  Injectable,
  UnauthorizedException,
} from '@nestjs/common';
import { JwtService } from '@nestjs/jwt';
import * as bcrypt from 'bcrypt';
import { PrismaService } from '../prisma/prisma.service';

const BCRYPT_ROUNDS = 12;

export type AuthUser = {
  id: number;
  email: string;
  name: string | null;
};

export type JwtPayload = {
  sub: number;
  email: string;
};

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
    const user = await this.prisma.user.create({
      data: { email: input.email, passwordHash, name: input.name ?? null },
      select: { id: true, email: true, name: true },
    });
    return user;
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
      select: { id: true, email: true, name: true },
    });
  }
}
