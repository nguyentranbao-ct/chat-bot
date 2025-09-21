import io from 'socket.io-client';

const socketServer = process.env.SOCKET_SERVER_URL || 'http://localhost:7070/events';
const apiKey = process.env.SOCKET_API_KEY || 'x';
const projectId = process.env.PROJECT_ID || 'p';
const userId = process.env.USER_ID || 'u';
const platform = process.env.PLATFORM || 'web';
const fingerprint = process.env.FINGERPRINT || 'f';
const deviceId = process.env.DEVICE_ID || 'd';
main(socketServer);

function main(socketServer) {
  const socket = io(socketServer, {
    path: '/ws',
    transports: ['websocket'],
    query: {
      api_key: apiKey,
      user_id: userId,
      fingerprint,
      device_id: deviceId,
    },
    extraHeaders: {
      'x-project-id': projectId,
      'x-platform': platform,
    },
  });

  socket.onAny((event, data) => {
    console.log('receive event', event, data);
  });

  socket.on('connect_error', (err) => {
    console.log('connect error', err);
  });

  socket.on('disconnect', () => {
    console.log('disconnected');
  });

  socket.on('connect', () => {
    console.log('connected');
  });
}
