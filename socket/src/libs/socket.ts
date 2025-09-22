import { Socket } from 'socket.io';

export const getHeader = (socket: Socket, key: string) => {
  const header = socket.handshake.headers[key] || socket.request.headers[key];
  return Array.isArray(header) ? header[0] : header;
};

export const getQuery = (socket: Socket, key: string) => {
  const query = socket.handshake.query[key];
  return Array.isArray(query) ? query[0] : query;
};

export const getUserKeyRoom = (userKey: string) => {
  return `uk:${userKey}`;
};

export const getUserPlatformRoom = (
  userKey: string,
  platform: string,
) => {
  return `uk:${userKey}:plf:${platform}`;
};
