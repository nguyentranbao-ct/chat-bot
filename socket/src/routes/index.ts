import { getMetrics } from '@/libs/metrics';
import { getUserKeyRoom, getUserPlatformRoom } from '@/libs/socket';
import { ioEvents } from '@/server/socket';
import Router from '@koa/router';
import { Type } from '@sinclair/typebox';
import Ajv from 'ajv';

const ajv = new Ajv();

const router = new Router();

const validateSendEvents = ajv.compile(
  Type.Object(
    {
      events: Type.Array(
        Type.Object({
          project_id: Type.String(),
          user_key: Type.String(),
          platform: Type.Optional(Type.String()),
          name: Type.String(),
          data: Type.Any(),
        }),
        { minItems: 1 },
      ),
    },
    { additionalProperties: false },
  ),
);

router
  .get('/health', ctx => {
    ctx.body = { status: 'ok' };
  })
  .get('/metrics', async ctx => {
    ctx.body = await getMetrics();
  })
  .post('/v1/events', async ctx => {
    const valid = validateSendEvents(ctx.request.body);
    if (!valid) {
      ctx.status = 400;
      ctx.body = {
        success: false,
        error: ajv.errorsText(validateSendEvents.errors),
      };
      return;
    }

    const { events } = ctx.request.body;
    for (const e of events) {
      const room = e.platform
        ? getUserPlatformRoom(e.project_id, e.user_key, e.platform)
        : getUserKeyRoom(e.project_id, e.user_key);
      ioEvents.to(room).emit(e.name, e.data);
    }
    ctx.body = { success: true };
  });

export default router;
