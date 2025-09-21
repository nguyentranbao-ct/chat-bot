import { config } from '@/config';
import { log } from '@/libs/logger';
import { errorHandler } from '@/middlewares/error_handler';
import { logRequest } from '@/middlewares/log_request';
import { metrics } from '@/middlewares/metrics';
import router from '@/routes';
import bodyParser from '@koa/bodyparser';
import Koa from 'koa';

const {
  http: { host, port },
} = config;

const app = new Koa();
app
  .use(bodyParser())
  .use(metrics)
  .use(logRequest)
  .use(errorHandler)
  .use(router.routes())
  .use(router.allowedMethods());

log.warn(`start listening on ${host}:${port}`);
const server = app.listen({ host, port });
export { app, server };
