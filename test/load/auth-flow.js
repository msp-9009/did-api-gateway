import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const errorRate = new Rate('errors');
const challengeLatency = new Trend('challenge_latency');
const verifyLatency = new Trend('verify_latency');

// Load test configuration
export let options = {
    stages: [
        { duration: '2m', target: 100 },     // Warm up
        { duration: '5m', target: 1000 },    // Baseline load
        { duration: '2m', target: 5000 },    // Ramp to peak
        { duration: '5m', target: 5000 },    // Sustained peak load
        { duration: '2m', target: 10000 },   // Spike test
        { duration: '1m', target: 10000 },   // Hold spike
        { duration: '2m', target: 1000 },    // Scale down
        { duration: '1m', target: 0 },       // Cool down
    ],

    thresholds: {
        http_req_duration: ['p(99)<100'],     // 99% of requests < 100ms
        http_req_failed: ['rate<0.01'],       // Error rate < 1%
        errors: ['rate<0.01'],                // Custom error rate < 1%
        challenge_latency: ['p(95)<50'],      // 95% < 50ms
        verify_latency: ['p(95)<150'],        // 95% < 150ms (includes VC verification)
    },
};

// Test DIDs (pre-generated for performance)
const testDIDs = [
    'did:key:z6MkpTHR8VNsBxYAAWHut2Geadd9jSwuBV8xRoAnwWsdvktH',
    'did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK',
    'did:key:z6MkjchhfUsD6mmvni8mCdXHw216Xrm9bQe2mBH1P5RDjVJG',
    'did:web:localhost:8888',
];

export default function () {
    // Select a random DID
    const did = testDIDs[Math.floor(Math.random() * testDIDs.length)];

    // 1. Request challenge
    const challengeStart = Date.now();
    const challengeRes = http.get(
        `http://${__ENV.GATEWAY_HOST || 'localhost:8080'}/v1/auth/challenge?did=${did}`,
        {
            tags: { name: 'challenge' },
        }
    );

    challengeLatency.add(Date.now() - challengeStart);

    const challengeOk = check(challengeRes, {
        'challenge status 200': (r) => r.status === 200,
        'challenge has required fields': (r) => {
            if (r.status !== 200) return false;
            const body = JSON.parse(r.body);
            return body.challenge && body.nonce && body.expires_at;
        },
        'challenge latency < 50ms': (r) => r.timings.duration < 50,
    });

    if (!challengeOk) {
        errorRate.add(1);
        return;
    }

    // Parse challenge response
    let challenge, nonce;
    try {
        const body = JSON.parse(challengeRes.body);
        challenge = body.challenge;
        nonce = body.nonce;
    } catch (e) {
        errorRate.add(1);
        return;
    }

    // Simulate signing delay (client-side operation)
    sleep(0.05);

    // 2. Verify (with mock signature and credential)
    const verifyStart = Date.now();
    const verifyPayload = {
        did: did,
        challenge: challenge,
        signature: 'mock_signature_' + randomString(64), // In real scenario, sign with private key
        credential: 'mock_vc_jwt_' + randomString(100),  // Mock JWT-VC
        scopes: ['basic'],
    };

    const verifyRes = http.post(
        `http://${__ENV.GATEWAY_HOST || 'localhost:8080'}/v1/auth/verify`,
        JSON.stringify(verifyPayload),
        {
            headers: { 'Content-Type': 'application/json' },
            tags: { name: 'verify' },
        }
    );

    verifyLatency.add(Date.now() - verifyStart);

    const verifyOk = check(verifyRes, {
        'verify status 200 or 401': (r) => r.status === 200 || r.status === 401,
        'verify latency < 150ms': (r) => r.timings.duration < 150,
    });

    if (!verifyOk) {
        errorRate.add(1);
    }

    // Note: In production load test, verify would succeed with real signatures
    // For testing gateway performance, we accept 401 (invalid signature) as valid response

    // Simulate user think time
    sleep(0.1);
}

// Setup function (runs once per VU)
export function setup() {
    console.log('Starting load test...');
    console.log(`Target: ${__ENV.GATEWAY_HOST || 'localhost:8080'}`);
    return {};
}

// Teardown function (runs once at end)
export function teardown(data) {
    console.log('Load test complete');
}
