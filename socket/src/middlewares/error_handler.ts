import { Context, Next } from 'koa';

export const errorHandler = async (ctx: Context, next: Next) => {
  try {
    await next();
  } catch (e) {
    if (ctx.respond) {
      return;
    }
    const body = {
      success: false,
      message: 'Internal Server Error',
    };
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const err = e as any;
    ctx.status = err?.statusCode || err?.status || 500;
    if (err?.message && ctx.status < 500) {
      body.message = err.message;
    }
    ctx.body = body;
  }
};
