// k6 load test — GET /api/v1/healthz
//
// This is the only unauthenticated endpoint. It serves as a baseline
// for the server's raw throughput without auth overhead.
//
// Usage:
//   k6 run deploy/load-tests/health.js
//
// Env vars:
//   BASE_URL   — target URL (default: http://localhost:8080)
//   DURATION   — test duration (default: 30s)
//   VUS        — target virtual users (default: 500)

import { check } from 'k6';
import http from 'k6/http';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const DURATION = __ENV.DURATION || '30s';
const VUS = parseInt(__ENV.VUS || '500', 10);

export const options = {
  stages: [
    { target: Math.ceil(VUS * 0.2), duration: '10s' },  // ramp-up to 20%
    { target: VUS,                   duration: '15s' },  // ramp to target
    { target: VUS,                   duration: DURATION },// sustained load
    { target: 0,                     duration: '10s' },  // ramp-down
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95th percentile < 1s
    http_req_failed:   ['rate<0.01'],  // < 1% errors
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/api/v1/healthz`, {
    tags: { endpoint: 'healthz' },
  });

  check(res, {
    'status is 200':        (r) => r.status === 200,
    'body has ok status':   (r) => r.json('status') === 'ok',
    'response time < 200ms': (r) => r.timings.duration < 200,
  });
}
