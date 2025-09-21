import { log } from '@/libs/logger';
import { server } from '@/server/http';
import { io } from '@/server/socket';

io.listen(server);

let count = 0;
for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => {
    if (count++) {
      log.warn(`force closed by ${signal}`);
      process.exit(1);
    }
    server.close(err => {
      log.warn(`server stopped by ${signal}`);
      io.disconnectSockets();
      if (err) {
        log.error(`close error ${err}`);
      }
      process.exit(err ? 1 : 0);
    });
  });
}
