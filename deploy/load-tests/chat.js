// k6 load test — POST /aleph.v1.QueryService/Chat (streaming)
//
// Tests the ConnectRPC server-streaming Chat endpoint. Simulates
// sending messages to an AI agent and receiving streaming tokens.
//
// Usage:
//   k6 run deploy/load-tests/chat.js
//
// Env vars:
//   BASE_URL   — target URL (default: http://localhost:8080)
//   DURATION   — test duration (default: 30s)
//   VUS        — target virtual users (default: 100)
//   API_KEY    — X-Aleph-Api-Key header value (default: test-key)
//   PROJECT_ID — project to use (default: default)
//   AGENT_ID   — agent ID to chat with (default: "")

import { check } from 'k6';
import http from 'k6/http';

const BASE_URL    = __ENV.BASE_URL    || 'http://localhost:8080';
const DURATION    = __ENV.DURATION    || '30s';
const VUS         = parseInt(__ENV.VUS || '100', 10);  // lower default — streaming is heavier
const API_KEY     = __ENV.API_KEY     || 'test-key';
const PROJECT_ID  = __ENV.PROJECT_ID  || 'default';
const AGENT_ID    = __ENV.AGENT_ID    || '';

// Sample prompts to vary the payload across iterations
const PROMPTS = [
  'Summarize the latest data trends.',
  'What are the top anomalies in the dataset?',
  'Show me a forecast for next quarter.',
  'Explain the recent spike in metric X.',
  'Compare current values with last month.',
];

export const options = {
  stages: [
    { target: Math.ceil(VUS * 0.2), duration: '15s' },
    { target: VUS,                   duration: '20s' },
    { target: VUS,                   duration: DURATION },
    { target: 0,                     duration: '15s' },
  ],
  thresholds: {
    http_req_duration: ['p(95)<5000'], // streaming — allow up to 5s
    http_req_failed:   ['rate<0.02'],  // < 2% errors (streaming is more fragile)
  },
};

export default function () {
  const prompt = PROMPTS[Math.floor(Math.random() * PROMPTS.length)];

  const payload = JSON.stringify({
    message:     prompt,
    project_id:  PROJECT_ID,
    agent_id:    AGENT_ID,
  });

  const res = http.post(
    `${BASE_URL}/aleph.v1.QueryService/Chat`,
    payload,
    {
      headers: {
        'Content-Type':     'application/json',
        'X-Aleph-Api-Key': API_KEY,
      },
      tags: { endpoint: 'chat' },
    },
  );

  check(res, {
    'status is 200':        (r) => r.status === 200,
    'response is non-empty': (r) => r.body.length > 0,
    'response time < 5000ms': (r) => r.timings.duration < 5000,
  });
}
