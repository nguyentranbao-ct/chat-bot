export const parseBool = (
  value: string | undefined,
  defaultValue = false,
): boolean => {
  if (!value) return defaultValue;
  return ['true', '1'].includes(value.toLowerCase());
};

export const config = {
  log: {
    level: process.env.LOG_LEVEL || 'info',
    requestBody: parseBool(process.env.LOG_REQUEST_BODY, true),
    responseBody: parseBool(process.env.LOG_RESPONSE_BODY, true),
    socketConnection: parseBool(process.env.LOG_SOCKET_CONNECTION, false),
  },
  http: {
    host: process.env.HTTP_HOST || 'localhost',
    port: +(process.env.HTTP_PORT || 7070),
  },
  redis: {
    host: process.env.REDIS_HOST || 'localhost',
    port: +(process.env.REDIS_PORT || 6379),
    pass: process.env.REDIS_PASS || '',
    user: process.env.REDIS_USER || '',
    db: +(process.env.REDIS_DB || 0),
  },
};
