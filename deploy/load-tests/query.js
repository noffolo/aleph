// k6 load test — POST /aleph.v1.QueryService/ExecuteQuery
//
// Tests the ConnectRPC unary ExecuteQuery endpoint. Simulates real
// DuckDB query execution against project data.
//
// Usage:
//   k6 run deploy/load-tests/query.js
//
// Env vars:
//   BASE_URL   — target URL (default: http://localhost:8080)
//   DURATION   — test duration (default: 30s)
//   VUS        — target virtual users (default: 500)
//   API_KEY    — X-Aleph-Api-Key header value (default: test-key)
//   PROJECT_ID — project to query (default: default)
//   OBJECT_TYPE — object type to query (default: "")

import { check, sleep } from 'k6';
import http from 'k6/http';

const BASE_URL    = __ENV.BASE_URL    || 'http://localhost:8080';
const DURATION    = __ENV.DURATION    || '30s';
const VUS         = parseInt(__ENV.VUS || '500', 10);
const API_KEY     = __ENV.API_KEY     || 'test-key';
const PROJECT_ID  = __ENV.PROJECT_ID  || 'default';
const OBJECT_TYPE = __ENV.OBJECT_TYPE || '';

export const options = {
  stages: [
    { target: Math.ceil(VUS * 0.2), duration: '10s' },
    { target: VUS,                   duration: '15s' },
    { target: VUS,                   duration: DURATION },
    { target: 0,                     duration: '10s' },
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed:   ['rate<0.01'],
  },
};

export default function () {
  const payload = JSON.stringify({
    object_type: OBJECT_TYPE,
    project_id:  PROJECT_ID,
    limit:       100,
  });

  const res = http.post(
    `${BASE_URL}/aleph.v1.QueryService/ExecuteQuery`,
    payload,
    {
      headers: {
        'Content-Type':     'application/json',
        'X-Aleph-Api-Key': API_KEY,
      },
      tags: { endpoint: 'query' },
    },
  );

  check(res, {
    'status is 200':              (r) => r.status === 200,
    'response has columns':       (r) => {
      try { return JSON.parse(r.body).columns !== undefined; }
      catch { return false; }
    },
    'response time < 1000ms':     (r) => r.timings.duration < 1000,
  });

  // Small think-time between requests to simulate realistic usage
  sleep(Math.random() * 0.5);
}
