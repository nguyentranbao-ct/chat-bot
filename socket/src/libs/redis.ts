import { config } from '@/config';
import { createClient } from 'redis';

const {
  redis: { host, port, user, pass, db },
} = config;

const rdb = createClient({
  url: `redis://${user}:${pass}@${host}:${port}/${db}`,
});

try {
  await rdb.connect();
} catch (e: unknown) {
  throw new Error(`connect redis: ${e}`);
}

export { rdb };
