import { Gauge, Histogram, register } from 'prom-client';

const buckets = [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10];

export const requestDurationSeconds = new Histogram({
  name: 'request_duration_seconds',
  help: 'The HTTP request latencies in seconds.',
  labelNames: ['code', 'method', 'path'],
  buckets,
});

export const socketActiveConnections = new Gauge({
  name: 'socket_active_connections',
  help: 'A gauge of the number of active connections.',
  labelNames: ['project', 'platform'],
});

export const getMetrics = () => {
  return register.metrics();
};
