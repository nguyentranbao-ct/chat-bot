import { requestDurationSeconds } from '@/libs/metrics';
import { Context, Next } from 'koa';

export const metrics = (ctx: Context, next: Next) => {
  const startTime = Date.now();
  const { res } = ctx;
  res.once('finish', () => {
    const { request, res, _matchedRoute } = ctx;
    const ms = Date.now() - startTime;
    const path = _matchedRoute;
    requestDurationSeconds
      .labels(res.statusCode.toString(), request.method, path)
      .observe(ms / 1000);
  });

  return next();
};
