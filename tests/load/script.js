import http from 'k6/http'
import { check, sleep } from 'k6'
import { Counter, Trend } from 'k6/metrics'
import { FormData } from "https://jslib.k6.io/formdata/0.0.2/index.js"



export const options = {
    http: { reuseConnections: true },
    stages: [
        { duration: '30s', target: 100 },
        { duration: '1m', target: 300 },
        { duration: '30s', target: 0 },
    ],
    thresholds: {
        http_req_duration: [
            { threshold: 'p(95)<500', abortOnFail: true },
            { threshold: 'p(90)<350' },
            { threshold: 'p(85)<200' },
        ],
    }
}

const responseTimeTrend = new Trend('response_time_trend')
const authSuccessCounter = new Counter('auth_success')
const authFailCounter = new Counter('auth_failures')
const refreshSuccessCounter = new Counter("refresh_success")
const refreshFailCounter = new Counter('refresh_failures')
const logoutSuccessCounter = new Counter("logout_success")
const logoutFailCounter = new Counter('logout_failures')

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080'
const TEST_USER_PASSWORD = __ENV.TEST_PASSWORD || 'load-test-password'

let users = []

const vuUserMap = new Map();


export function setup() {
    const listResp = http.get(`${BASE_URL}/users`)
    const usrList = listResp.json()
    if (usrList && usrList["data"].length > 0) {
        for (let j = 0; j < usrList["data"].length; j++) {
            usrList["data"][j]["password"] = TEST_USER_PASSWORD
            users.push(usrList["data"][j])
        }
        return { users }
    }

    const createUsers = []
    const userCount = 1000
    for (let i = 0; i < userCount; i++) {
        const email = `loadtest${i}@example.com`

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
        })
    }

    const responses = http.batch(createUsers)
    users = responses.map((res, i) => {
        if (res.status === 201) {
            return {
                email: `loadtest${i}@example.com`,
                password: TEST_USER_PASSWORD
            }
        }
        return null
    }).filter(user => user !== null)

    return { users }
}

export default function (data) {
    const {users} = data

    if (!vuUserMap.has(__VU)) {
        vuUserMap.set(__VU, users[__VU % users.length]);
    }
    const user = vuUserMap.get(__VU);

    // Authenticate
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
    )

    const authSuccess = check(loginRes, {
        'auth status is 200': (r) => r.status === 200,
        'auth received cookies': (r) => {
            return loginRes.cookies['access'][0].value !== undefined && loginRes.cookies['refresh'][0].value !== undefined
        }
    })

    if (!authSuccess) {
        authFailCounter.add(1)
        return
    }

    authSuccessCounter.add(1)
    const jar = http.cookieJar()
    jar.set(`${BASE_URL}`, 'access', loginRes.cookies['access'][0].value)
    jar.set(`${BASE_URL}`, 'refresh', loginRes.cookies['refresh'][0].value)

    sleep(Math.random() * 2)
    // Refresh

    const refreshRes = http.post(
        `${BASE_URL}/auth/jwt/refresh`,
        null,
        {
            headers: {
                'Content-Type': 'application/json',
                'User-Agent': 'k6-load-test/1.0',
                'X-Real-IP': '127.0.0.1',
            }
        }
    )

    const refreshSuccess = check(refreshRes, {
        'refresh status is 200': (r) => r.status === 200,
        'refresh received cookies': (r) => {
            return refreshRes.cookies['access'][0].value !== undefined && refreshRes.cookies['refresh'][0].value !== undefined
        }
    })

    if (!refreshSuccess) {
        refreshFailCounter.add(1)
        return
    }

    refreshSuccessCounter.add(1)
    jar.clear()
    jar.set(`${BASE_URL}`, 'access', refreshRes.cookies['access'][0].value)
    jar.set(`${BASE_URL}`, 'refresh', refreshRes.cookies['refresh'][0].value)
    sleep(Math.random() * 2)

    // Log out
    const logoutRes = http.post(
        `${BASE_URL}/auth/logout`,
        null,
        {
            headers: {
                'Content-Type': 'application/json',
            }
        }
    )

    const logoutSuccess = check(logoutRes, {
        'refresh status is 200': (r) => r.status === 200,
        'received cookies': (r) => {
            return logoutRes.cookies['access'][0].value !== undefined && logoutRes.cookies['refresh'][0].value !== undefined
        }
    })
    if (!logoutSuccess) {
        logoutFailCounter.add(1)
        return
    }

    logoutSuccessCounter.add(1)

}
