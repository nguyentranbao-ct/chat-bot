import { log } from '@/libs/logger';
import { rdb } from '@/libs/redis';
import { getUserKeyRoom, getUserPlatformRoom } from '@/libs/socket';
import { authSocket, socketMetrics } from '@/middlewares/socket';
import { ISocket } from '@/types/socket';
import { createAdapter } from '@socket.io/redis-adapter';
import { Server as SocketServer } from 'socket.io';
import { config } from '@/config';
const logSocketConnection = config.log.socketConnection;

const io = new SocketServer({
  path: '/ws',
  transports: ['websocket'],
});

const pub = rdb.duplicate();
const sub = rdb.duplicate();
try {
  await pub.connect();
  await sub.connect();
} catch (e: unknown) {
  throw new Error(`connect redis: ${e}`);
}
io.adapter(
  createAdapter(pub, sub, {
    key: 'ws',
  }),
);

const allowedNamespaces = new Set(['/events']);
io.use((socket, next) => {
  if (!allowedNamespaces.has(socket.nsp.name)) {
    return next(new Error(`unknown namespace: ${socket.nsp.name}`));
  }
  next();
});

const ioEvents = io.of('/events');
ioEvents.use(authSocket);
ioEvents.on('connection', async (s) => {
  const socket = <ISocket>s;
  const profile = {
    sid: socket.id,
    project_id: socket.$projectId,
    platform: socket.$platform,
    user_id: socket.$userId,
    device_id: socket.$deviceId,
    fingerprint: socket.$fingerprint,
  };

  if (logSocketConnection) {
    log.info(profile, 'connected');
  }
  socket.on('disconnect', () => {
    if (logSocketConnection) {
      log.info(profile, 'disconnected');
    }
  });
  socketMetrics(socket);

  const selfRoom = getUserKeyRoom(socket.$projectId, socket.$userKey);
  const platformRoom = getUserPlatformRoom(
    socket.$projectId,
    socket.$userKey,
    socket.$platform,
  );
  socket.join([selfRoom, platformRoom]);
});

export { io, ioEvents };
