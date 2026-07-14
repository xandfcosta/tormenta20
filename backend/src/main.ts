import { ValidationPipe } from '@nestjs/common';
import { NestFactory } from '@nestjs/core';
import compression from 'compression';
import cookieParser from 'cookie-parser';
import { AppModule } from './app.module';
import { validationExceptionFactory } from './common/validation-exception.factory';

async function bootstrap() {
  const app = await NestFactory.create(AppModule);
  // Gzip every response. Character-aggregate JSON (5–15 KB) compresses ~70–80%;
  // transparent to clients (Accept-Encoding negotiated). Cheapest transfer win
  // before any per-endpoint delta shaping.
  app.use(compression());
  app.use(cookieParser());
  app.useGlobalPipes(
    new ValidationPipe({
      whitelist: true,
      forbidNonWhitelisted: true,
      transform: true,
      exceptionFactory: validationExceptionFactory,
    }),
  );
  app.enableCors({
    origin: process.env.CORS_ORIGIN ?? 'http://localhost:5173',
    credentials: true,
  });
  await app.listen(process.env.PORT ?? 3000);
}
bootstrap();
