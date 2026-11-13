// Light load smoke for FixPhone.
//
// Target: 50 concurrent users, p95 < 500 ms, failure rate < 1%.
// The scenario is read-only so it can run against staging without polluting
// business data. Login happens once in setup(); each VU reuses the token.
//
// Usage:
//   k6 run -e BASE_URL=http://127.0.0.1:8080/api/v1 \
//          -e EMAIL=owner@toko.id -e PASSWORD=Rahasia123 scripts/load/k6-smoke.js
//
// Optional: install k6 from https://k6.io/docs/get-started/installation/

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';

const BASE_URL = (__ENV.BASE_URL || 'http://127.0.0.1:8080/api/v1').replace(/\/$/, '');
const EMAIL = __ENV.EMAIL || 'owner@toko.id';
const PASSWORD = __ENV.PASSWORD || 'Rahasia123';

const readTrend = new Trend('read_duration', true);

export const options = {
  scenarios: {
    smoke: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 50 },
        { duration: '60s', target: 50 },
        { duration: '15s', target: 0 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<500'],
    read_duration: ['p(95)<500'],
  },
};

export function setup() {
  const response = http.post(
    `${BASE_URL}/auth/login`,
    JSON.stringify({ email: EMAIL, password: PASSWORD }),
    { headers: { 'Content-Type': 'application/json' } },
  );
  const ok = check(response, { 'login 200': (res) => res.status === 200 });
  if (!ok) {
    throw new Error(`login gagal (${response.status}): ${response.body}`);
  }
  return { token: response.json('data.access_token') };
}

const ENDPOINTS = [
  '/dashboard/summary',
  '/dashboard/charts?range=7',
  '/service-orders?page=1&per_page=20',
  '/invoices?page=1&per_page=20',
  '/customers?page=1&per_page=20',
  '/spareparts?page=1&per_page=20',
];

export default function (data) {
  const headers = { Authorization: `Bearer ${data.token}` };
  const path = ENDPOINTS[Math.floor(Math.random() * ENDPOINTS.length)];
  const response = http.get(`${BASE_URL}${path}`, { headers, tags: { endpoint: path } });

  readTrend.add(response.timings.duration);
  check(response, {
    'status 200': (res) => res.status === 200,
    'envelope success': (res) => {
      try {
        return res.json('success') === true;
      } catch (err) {
        return false;
      }
    },
  });

  sleep(Math.random() * 2 + 0.5);
}
