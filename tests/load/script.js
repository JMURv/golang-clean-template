import http from 'k6/http'
import {check, sleep} from 'k6'
import {Counter, Trend} from 'k6/metrics'
import {FormData} from "https://jslib.k6.io/formdata/0.0.2/index.js"

const BASE_URL = __ENV.K6_BACKEND_BASE_URL || 'http://localhost:8080'
const TEST_USER_PASSWORD = __ENV.TEST_PASSWORD || 'load-test-password'
const PARALLEL_USERS = __ENV.K6_PARALLEL_USERS || 1000

export const options = {
    http: {reuseConnections: true},
    stages: [
        {duration: '30s', target: 100},
        {duration: '1m', target: PARALLEL_USERS},
        {duration: '30s', target: 0},
    ],
    thresholds: {
        http_req_duration: [
            {threshold: 'p(95)<500', abortOnFail: true},
            {threshold: 'p(95)<300'},
            {threshold: 'p(95)<200'},
        ],
    }
}

const authSuccessCounter = new Counter('auth_success')
const authFailCounter = new Counter('auth_failures')
const authTimeTrend = new Trend('auth_time_trend')

const getMeSuccessCounter = new Counter('getMe_success')
const getMeFailCounter = new Counter('getMe_failures')
const getMeTimeTrend = new Trend('getMe_time_trend')

const listDevicesSuccessCounter = new Counter('listDevices_success')
const listDevicesFailCounter = new Counter('listDevices_failures')
const listDevicesTimeTrend = new Trend('listDevices_time_trend')

const updateMeSuccessCounter = new Counter('updateMe_success')
const updateMeFailCounter = new Counter('updateMe_failures')
const updateMeTimeTrend = new Trend('updateMe_time_trend')

const updateUserSuccessCounter = new Counter('updateUser_success')
const updateUserFailCounter = new Counter('updateUser_failures')
const updateUserTimeTrend = new Trend('updateUser_time_trend')


const refreshSuccessCounter = new Counter("refresh_success")
const refreshFailCounter = new Counter('refresh_failures')
const refreshTimeTrend = new Counter('refresh_time_trend')

const logoutSuccessCounter = new Counter("logout_success")
const logoutFailCounter = new Counter('logout_failures')
const logoutTimeTrend = new Counter('logout_time_trend')

const vuUserMap = new Map()

export function setup() {
    let users = []

    const listResp = http.get(`${BASE_URL}/users`)
    const usrList = listResp.json()
    if (usrList && usrList["data"].length > 0) {
        for (let j = 0; j < usrList["data"].length; j++) {
            usrList["data"][j]["password"] = TEST_USER_PASSWORD
            users.push(usrList["data"][j])
        }
        return {users}
    }

    const createUsers = []
    for (let i = 0; i < PARALLEL_USERS; i++) {
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
            const data = res.json()
            return {
                id: data.id,
                email: `loadtest${i}@example.com`,
                password: TEST_USER_PASSWORD
            }
        }
        return null
    }).filter(user => user !== null)

    return {users}
}

export default function (data) {
    const {users} = data
    if (!users?.length) return

    if (!vuUserMap.has(__VU)) {
        vuUserMap.set(__VU, users[__VU % users.length]);
    }
    const user = vuUserMap.get(__VU);

    // Authenticate
    const loginStart = Date.now()
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
                'User-Agent': `k6-VU-${__VU}-ITER-${__ITER}`,
                'X-Real-IP': '127.0.0.1',
            }
        }
    )
    const loginEnd = Date.now()
    authTimeTrend.add(loginEnd - loginStart)

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

    // getMe
    const getMeStart = Date.now()
    const getMeRes = http.get(`${BASE_URL}/users/me`)
    const getMeEnd = Date.now()
    getMeTimeTrend.add(getMeEnd - getMeStart)

    const getMeSuccess = check(getMeRes, {
        'getMe status is 200': (r) => r.status === 200,
        'getMe avatar is empty': (r) => {
            const data = r.json()
            return data?.avatar === ""
        }
    })

    if (!getMeSuccess) {
        getMeFailCounter.add(1)
        return
    }
    getMeSuccessCounter.add(1)

    sleep(Math.random() * 2)

    // listDevices
    const listDevicesStart = Date.now()
    const listDevicesRes = http.get(`${BASE_URL}/device`)
    const listDevicesEnd = Date.now()
    listDevicesTimeTrend.add(listDevicesEnd - listDevicesStart)

    const listDevicesSuccess = check(listDevicesRes, {
        'listDevices status is 200': (r) => r.status === 200,
        'listDevices has one or more devices': (r) => {
            const data = r.json()
            return data.length >= 1
        }
    })

    if (!listDevicesSuccess) {
        listDevicesFailCounter.add(1)
        return
    }
    listDevicesSuccessCounter.add(1)

    sleep(Math.random() * 2)

    // UpdateUser
    const fd = new FormData()
    fd.append("data", JSON.stringify({
        name: `User UPDATED`,
        email: user.email,
        password: TEST_USER_PASSWORD
    }))

    const updateUserStart = Date.now()
    const updateUserRes = http.put(`${BASE_URL}/users/${user.id}`, fd.body(), {
        headers: {
            'Content-Type': `multipart/form-data; boundary=${fd.boundary}`,
        }
    })
    const updateUserEnd = Date.now()
    updateUserTimeTrend.add(updateUserEnd - updateUserStart)

    const updateUserSuccess = check(updateUserRes, {
        'updateUser status is 200': (r) => r.status === 200,
    })

    if (!updateUserSuccess) {
        updateUserFailCounter.add(1)
        return
    }
    updateUserSuccessCounter.add(1)

    sleep(Math.random() * 2)

    // Check if data has changed
    const updateMeStart = Date.now()
    const updateMeRes = http.get(`${BASE_URL}/users/me`)
    const updateMeEnd = Date.now()
    updateMeTimeTrend.add(updateMeEnd - updateMeStart)

    const updateMeSuccess = check(updateMeRes, {
        'updateMe status is 200': (r) => r.status === 200,
        'updateMe username is updated': (r) => {
            const data = r.json()
            return data?.name === "User UPDATED"
        }
    })

    if (!updateMeSuccess) {
        updateMeFailCounter.add(1)
        return
    }
    updateMeSuccessCounter.add(1)

    sleep(Math.random() * 2)


    // Refresh
    const refreshStart = Date.now()
    const refreshRes = http.post(
        `${BASE_URL}/auth/jwt/refresh`,
        null,
        {
            headers: {
                'Content-Type': 'application/json',
                'User-Agent': `k6-VU-${__VU}-ITER-${__ITER}`,
                'X-Real-IP': '127.0.0.1',
            }
        }
    )
    const refreshEnd = Date.now()
    refreshTimeTrend.add(refreshEnd - refreshStart)

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
    const logoutStart = Date.now()
    const logoutRes = http.post(
        `${BASE_URL}/auth/logout`,
        null,
        {
            headers: {
                'Content-Type': 'application/json',
            }
        }
    )
    const logoutEnd = Date.now()
    logoutTimeTrend.add(logoutEnd - logoutStart)

    const logoutSuccess = check(logoutRes, {
        'logout status is 200': (r) => r.status === 200,
        'logout received cookies': (r) => {
            return logoutRes.cookies['access'][0].value !== undefined && logoutRes.cookies['refresh'][0].value !== undefined
        }
    })
    if (!logoutSuccess) {
        logoutFailCounter.add(1)
        return
    }
    logoutSuccessCounter.add(1)

}
