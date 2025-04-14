import http from 'k6/http';
import { check, sleep, fail } from 'k6';
import { Rate } from 'k6/metrics'; // Импортируем Rate для RPS метрики

// --- Конфигурация теста ---
const BASE_URL = 'http://localhost:8080'; // Адрес API
const PRODUCT_TYPES = ['электроника', 'одежда', 'обувь'];
// PVZ_ID получаем из переменной окружения в setup

// Создаем кастомную метрику для отслеживания реального RPS
const RPS_COUNTER = new Rate('http_reqs_rate');

export const options = {
    // Используем executor для контроля частоты запросов
    executor: 'ramping-arrival-rate',

    // Определяем единицу времени для rate
    timeUnit: '1s',

    // Начальное количество VUs, k6 будет создавать больше при необходимости
    preAllocatedVUs: 50,
    // Максимальное количество VUs, которое k6 может создать
    maxVUs: 500, // Можно увеличить, если k6 будет жаловаться, что не хватает VUs

    // Этапы нагрузки: пытаемся достичь ~500 итераций/сек
    // Каждая итерация делает 2 запроса, цель ~1000 запросов/сек (RPS)
    stages: [
        { duration: '30s', target: 200 },  // Разгон до 200 итераций/сек ( ~400 RPS) за 30 сек
        { duration: '1m', target: 200 },  // Держим 200 итераций/сек 1 минуту
        { duration: '30s', target: 500 },  // Разгон до 500 итераций/сек (~1000 RPS) за 30 сек
        { duration: '1m', target: 500 },  // Держим 500 итераций/сек 1 минуту
        { duration: '30s', target: 0 },    // Снижение до 0 за 30 сек
    ],

    thresholds: {
        // NFR цели:
        'http_req_duration': ['p(95)<100'], // SLI времени ответа p95 < 100ms
        'http_req_failed': ['rate<0.0001'], // SLI успешности 99.99% (ошибок < 0.01%)
        'checks': ['rate>0.99'], // Большинство проверок должно проходить
        // Порог для реального RPS (опционально, т.к. может не достигаться)
        // 'http_reqs_rate': ['rate>=1000'], // Проверка, что достигли 1000 RPS
    },
};

// --- Функция Setup: выполняется 1 раз перед тестом ---
export function setup() {
    console.log('=== Running Setup ===');
    console.log('Fetching authentication tokens...');

    // Обертка для получения токена
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


// --- Основная логика теста (выполняется каждым VU) ---
// Принимает данные из setup() как аргумент 'data'
export default function (data) {

    if (!data || !data.moderatorToken || !data.employeeToken) {
        // Это не должно происходить, если setup отработал, но на всякий случай
        return;
    }

    const modToken = data.moderatorToken;
    const empToken = data.employeeToken;
    const pvzId = data.pvzId;

    // --- Сценарий 1: Получение списка ПВЗ (модератор) ---
    const pvzParams = {
        headers: { 'Authorization': `Bearer ${modToken}` },
        tags: { name: 'GetPVZList' },
    };
    const pvzRes = http.get(`${BASE_URL}/pvz`, pvzParams);
    check(pvzRes, {
        '[GET /pvz] Status is 200': (r) => r.status === 200,
    }, { name: 'GetPVZList' });
    RPS_COUNTER.add(1); // Учитываем запрос в кастомном счетчике RPS

    // Убрали sleep для максимальной частоты запросов
    // sleep(0.5);

    // --- Сценарий 2: Добавление товара (сотрудник) ---
    if (pvzId) { // Выполняем, только если PVZ_ID был передан
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
        RPS_COUNTER.add(1); // Учитываем запрос в кастомном счетчике RPS

        // Логируем ошибку, если проверка не прошла и статус не 201
        if (!productCheck && productRes.status !== 201) {
            // Логируем только статус и VU ID, чтобы не перегружать вывод при высокой нагрузке
            console.log(`[VU: ${__VU}] POST /products failed. Status: ${productRes.status}`);
        }
    }
    // Убрали sleep для максимальной частоты запросов
    // sleep(1);
}