import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const resolutionLatency = new Trend('resolution_latency');
const cacheHitRate = new Rate('cache_hits');

// Test configuration
export let options = {
    stages: [
        { duration: '1m', target: 50 },
        { duration: '3m', target: 500 },
        { duration: '2m', target: 1000 },
        { duration: '3m', target: 1000 },
        { duration: '1m', target: 0 },
    ],

    thresholds: {
        http_req_duration: ['p(99)<100'],
        http_req_failed: ['rate<0.01'],
        errors: ['rate<0.01'],
        resolution_latency: ['p(95)<50'],
        cache_hits: ['rate>0.8'],  // Target >80% cache hit rate
    },
};

// Test DIDs with different methods
const testDIDs = [
    // did:key (should be cached permanently)
    'did:key:z6MkpTHR8VNsBxYAAWHut2Geadd9jSwuBV8xRoAnwWsdvktH',
    'did:key:z6MkhaXgBZDvotDkL5257faiztiGiC2QtKLGpbnnEGta2doK',
    'did:key:z6MkjchhfUsD6mmvni8mCdXHw216Xrm9bQe2mBH1P5RDjVJG',

    // did:web (should be cached for 1 hour)
    'did:web:localhost:8888',
    'did:web:example.com',
];

// Distribution: 80% did:key (high cache hit), 20% did:web
function selectDID() {
    const rand = Math.random();
    if (rand < 0.8) {
        // did:key (indices 0-2)
        return testDIDs[Math.floor(Math.random() * 3)];
    } else {
        // did:web (indices 3-4)
        return testDIDs[3 + Math.floor(Math.random() * 2)];
    }
}

export default function () {
    const did = selectDID();

    // Test DID resolution via challenge endpoint
    const start = Date.now();
    const res = http.get(
        `http://${__ENV.GATEWAY_HOST || 'localhost:8080'}/v1/auth/challenge?did=${did}`,
        {
            tags: {
                name: 'did_resolution',
                did_method: did.split(':')[1],
            },
        }
    );

    const latency = Date.now() - start;
    resolutionLatency.add(latency);

    const ok = check(res, {
        'status 200': (r) => r.status === 200,
        'latency < 100ms': (r) => r.timings.duration < 100,
    });

    if (!ok) {
        errorRate.add(1);
    } else {
        // Check if response indicates cache hit (via custom header if added)
        // Or infer from latency: <10ms likely cache hit
        if (latency < 10) {
            cacheHitRate.add(1);
        } else {
            cacheHitRate.add(0);
        }
    }

    // Vary request rate to simulate real traffic
    sleep(Math.random() * 0.5);
}

export function setup() {
    console.log('Starting DID resolution load test...');
    console.log('Target cache hit rate: >80%');
    return {};
}

export function teardown(data) {
    console.log('DID resolution test complete');
}
