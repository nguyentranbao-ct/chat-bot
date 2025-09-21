import { log } from '@/libs/logger';
import { Context, Next } from 'koa';
import { v7 as uuid } from 'uuid';
import { config } from '@/config';

const { requestBody, responseBody } = config.log;
const jsonType = 'application/json';

export const logRequest = async (ctx: Context, next: Next) => {
  const ts = Date.now();
  const { req, request, res } = ctx;
  if (request.path === '/health' || request.path === '/metrics') {
    return next();
  }
  const reqId =
    req.headers['x-request-id'] || req.headers['x-correlation-id'] || uuid();

  ctx.log = log.child({ corr_id: reqId });
  let err: Error | undefined;
  try {
    await next();
  } catch (e: unknown) {
    err = e as Error;
  }

  res.once('finish', () => {
    const duration = Date.now() - ts;
    const status = res.statusCode;
    const data: Record<string, unknown> = {
      method: req.method,
      status: res.statusCode,
      path: request.path,
      query: request.querystring ? request.query : undefined,
      latency_ms: duration,
      user_agent: req.headers['user-agent'],
      ip: request.ip,
    };
    if (requestBody && request.is(jsonType)) {
      data.request = request.body;
    }
    if (responseBody && ctx.response.is(jsonType)) {
      data.response = ctx.body;
    }
    if (err) {
      data.error = err.toString();
      if (err.stack) {
        data.error = err.stack;
      }
    }

    switch (true) {
      case status >= 500:
        ctx.log.error(data);
        break;
      case status >= 400:
        ctx.log.warn(data);
        break;
      default:
        ctx.log.info(data);
        break;
    }
  });

  // re-throw the error to be caught by the global error handler
  if (err) {
    throw err;
  }
};
