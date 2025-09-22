import { log } from '@/libs/logger';
import {
  requestDurationSeconds,
  socketActiveConnections,
} from '@/libs/metrics';
import { getHeader, getQuery } from '@/libs/socket';
import { ISocket } from '@/types/socket';
import { Event, Socket } from 'socket.io';
import jwt from 'jsonwebtoken';

const JWT_SECRET =
  process.env.JWT_SECRET || 'your-jwt-secret-key-change-this-in-production';

export const authSocket = (s: Socket, next: (err?: Error) => void) => {
  const socket = <ISocket>s;

  // Extract JWT token from Authorization header or query parameter
  const token =
    getHeader(socket, 'authorization')?.replace('Bearer ', '') ||
    getQuery(socket, 'token');

  const meta: Record<string, string | undefined> = {
    sid: socket.id,
    platform: getHeader(socket, 'x-platform') || getQuery(socket, 'platform'),
    device_id: getQuery(socket, 'device_id'),
    fingerprint: getQuery(socket, 'fingerprint'),
  };

  const withErr = (msg: string): void => {
    log.warn(meta, `connect error: ${msg}`);
    next(new Error(msg));
  };

  // Validate JWT token
  if (!token) {
    withErr('missing JWT token');
    return;
  }

  try {
    const decoded = jwt.verify(token, JWT_SECRET) as any;
    meta.user_id = decoded.user_id || decoded.sub || decoded.id;

    if (!meta.user_id) {
      withErr('invalid JWT token: missing user id');
      return;
    }
  } catch (err: any) {
    withErr(`invalid JWT token: ${err?.message || String(err)}`);
    return;
  }

  if (!meta.platform || meta.platform.trim() === '') {
    withErr('invalid platform');
    return;
  }
  if (!meta.fingerprint) {
    withErr('invalid fingerprint');
    return;
  }
  if (!meta.device_id) {
    withErr('invalid device id');
    return;
  }

  socket.$platform = meta.platform;
  socket.$fingerprint = meta.fingerprint;
  socket.$deviceId = meta.device_id;
  socket.$userId = meta.user_id;
  socket.$userKey = meta.user_id; // Use user_id as the primary key for rooms
  next();
};

export const socketMetrics = (socket: ISocket) => {
  const labels = {
    platform: socket.$platform,
  };
  socketActiveConnections.inc(labels);
  socket.on('disconnect', () => {
    socketActiveConnections.dec(labels);
  });
  socket.use(async (events: Event, next: (err?: Error) => void) => {
    const eventName = events[0];
    const startTime = Date.now();
    let status = 'success';
    try {
      next();
    } catch {
      status = 'error';
    }
    const duration = Date.now() - startTime;
    requestDurationSeconds
      .labels(status, `s:${eventName}`)
      .observe(duration / 1000);
  });
};
