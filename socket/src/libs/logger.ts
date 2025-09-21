import { config } from '@/config';
import { formatISO } from 'date-fns';
import pino from 'pino';

const {
  log: { level },
} = config;

export const log = pino({
  level,
  base: null,
  timestamp: () => `,"ts":"${formatISO(new Date())}"`,
  formatters: {
    level: label => ({ level: label }),
  },
});
