import { Socket } from 'socket.io';

export interface ISocket extends Socket {
  $userId?: string
  $platform: string
  $deviceId: string
  $fingerprint: string
  $userKey: string
}
