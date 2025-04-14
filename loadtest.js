import http from 'k6/http';
import { check, sleep, fail } from 'k6';
import { Rate } from 'k6/metrics';


const BASE_URL = 'http://localhost:8080';
const PRODUCT_TYPES = ['электроника', 'одежда', 'обувь'];



const RPS_COUNTER = new Rate('http_reqs_rate');

export const options = {

    executor: 'ramping-arrival-rate',


    timeUnit: '1s',


    preAllocatedVUs: 50,

    maxVUs: 500,


    stages: [
        { duration: '30s', target: 200 },
        { duration: '1m', target: 200 },
        { duration: '30s', target: 500 },
        { duration: '1m', target: 500 },
        { duration: '30s', target: 0 },
    ],

    thresholds: {
        // NFR цели:
        'http_req_duration': ['p(95)<100'],
        'http_req_failed': ['rate<0.0001'],
        'checks': ['rate>0.99'],


    },
};


export function setup() {
    console.log('=== Running Setup ===');
    console.log('Fetching authentication tokens...');


    function getAuthToken(role) {
        const payload = JSON.stringify({ role: role });
        const params = { headers: { 'Content-Type': 'application/json' } };
        const res = http.post(`${BASE_URL}/dummyLogin`, payload, params);
        if (res.status !== 200 || !res.json() || !res.json().token) {
            console.error(`SETUP FAILED: Could not get token for role: ${role}. Status: ${res.status}, Body: ${res.body}`);
            return null;
        }
        console.log(`Token obtained for role: ${role}`);
        return res.json().token;
    }

    const modToken = getAuthToken('moderator');
    const empToken = getAuthToken('employee');

    if (!modToken || !empToken) {
        fail("Critical setup failure: Could not retrieve authentication tokens.");
    }

    const pvzIdFromEnv = __ENV.PVZ_ID || null;
    if (!pvzIdFromEnv) {
        console.warn("WARN: PVZ_ID environment variable is not set. POST /products requests will be skipped.");
    }

    console.log('Authentication tokens obtained successfully.');
    console.log(`Using PVZ_ID: ${pvzIdFromEnv || 'Not Provided'}`);
    console.log('=== Setup Complete ===');

    return {
        moderatorToken: modToken,
        employeeToken: empToken,
        pvzId: pvzIdFromEnv
    };
}




export default function (data) {

    if (!data || !data.moderatorToken || !data.employeeToken) {

        return;
    }

    const modToken = data.moderatorToken;
    const empToken = data.employeeToken;
    const pvzId = data.pvzId;


    const pvzParams = {
        headers: { 'Authorization': `Bearer ${modToken}` },
        tags: { name: 'GetPVZList' },
    };
    const pvzRes = http.get(`${BASE_URL}/pvz`, pvzParams);
    check(pvzRes, {
        '[GET /pvz] Status is 200': (r) => r.status === 200,
    }, { name: 'GetPVZList' });
    RPS_COUNTER.add(1);




    if (pvzId) {
        const randomProductType = PRODUCT_TYPES[Math.floor(Math.random() * PRODUCT_TYPES.length)];
        const productPayload = JSON.stringify({
            type: randomProductType,
            pvzId: pvzId,
        });
        const productParams = {
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${empToken}`,
            },
            tags: { name: 'AddProduct' },
        };
        const productRes = http.post(`${BASE_URL}/products`, productPayload, productParams);
        const productCheck = check(productRes, {
            '[POST /products] Status is 201 (Created)': (r) => r.status === 201,
        }, { name: 'AddProduct' });
        RPS_COUNTER.add(1);


        if (!productCheck && productRes.status !== 201) {

            console.log(`[VU: ${__VU}] POST /products failed. Status: ${productRes.status}`);
        }
    }

}