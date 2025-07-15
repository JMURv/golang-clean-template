import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js"

export const options = {
    stages: [
        { duration: '30s', target: 100 },
        { duration: '1m', target: 500 },
        { duration: '30s', target: 0 },
    ],
    thresholds: {
        http_req_duration: ['p(95)<200'],    // 95% запросов должны быть быстрее 500мс
        http_req_failed: ['rate<0.01'],      // Менее 1% ошибок
    },
};

const requestCounter = new Counter('request_count');
const responseTimeTrend = new Trend('response_time_trend');
const authSuccessCounter = new Counter('auth_success');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const TEST_USER_PASSWORD = __ENV.TEST_PASSWORD || 'load-test-password';

let users = [];

export function setup() {
    // TODO: if users exists, GET to list EP
    const createUsers = [];
    const userCount = 1000;

    for (let i = 0; i < userCount; i++) {
        const email = `loadtest${i}@example.com`;

        const fd = new FormData()
        fd.append("data", JSON.stringify({
            name: `User ${i}`,
            email: email,
            password: TEST_USER_PASSWORD
        }))

        createUsers.push({
            method: 'POST',
            url: `${BASE_URL}/users`,
            body: fd.body(),
            params: {
                headers: {
                    'User-Agent': 'k6-load-test/1.0',
                    "Content-Type": `multipart/form-data; boundary=${fd.boundary}`
                }
            }
        });
    }

    const responses = http.batch(createUsers);
    users = responses.map((res, i) => {
        if (res.status === 201) {
            return {
                email: `loadtest${i}@example.com`,
                password: TEST_USER_PASSWORD
            };
        }
        return null;
    }).filter(user => user !== null);

    return { users };
}

export default function (data) {
    const {users} = data;
    if (!users || users.length === 0) return;

    // TODO: get users by order
    const user = users[Math.floor(Math.random() * users.length)];

    // Authenticate
    // TODO: refresh_tokens_token_hash_key duplicates
    const loginRes = http.post(
        `${BASE_URL}/auth/jwt`,
        JSON.stringify({
            email: user.email,
            password: user.password,
            token: "test-token"
        }),
        {
            headers: {
                'Content-Type': 'application/json',
                'User-Agent': 'k6-load-test/1.0',
                'X-Real-IP': '127.0.0.1',
            }
        }
    );

    // Check success authentication
    // TODO: regex to get values after access=...
    const authSuccess = check(loginRes, {
        'auth status is 200': (r) => r.status === 200,
        'received cookies': (r) => {
            const cookies = parseCookies(r.headers);
            return cookies['refresh'] !== undefined && cookies['access'] !== undefined;
        }
    });

    if (!authSuccess) {
        return;
    }

    authSuccessCounter.add(1);

}


function parseCookies(headers) {
    const cookies = {};
    const cookieHeader = headers['Set-Cookie'];

    if (cookieHeader) {
        cookieHeader.forEach(cookieStr => {
            cookieStr.split(';').forEach(part => {
                const [key, value] = part.trim().split('=');
                if (key && value) {
                    cookies[key] = decodeURIComponent(value);
                }
            });
        });
    }

    return cookies;
}