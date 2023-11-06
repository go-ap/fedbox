import {check, group, sleep} from 'k6';
import http from 'k6/http';
import {Rate} from 'k6/metrics';

const errors = new Rate('error_rate');

export const options = {
    thresholds: {
        'http_req_duration': ['p(95)<200'],
        'error_rate': [{threshold: 'rate < 0.01', abortOnFail: true, delayAbortEval: '1s'}],
        'error_rate{errorType:responseStatusError}': [{threshold: 'rate < 0.1'}],
        'error_rate{errorType:contentTypeError}': [{threshold: 'rate < 0.1'}],
        'error_rate{errorType:bodySizeError}': [{threshold: 'rate < 0.1'}],
        'error_rate{errorType:ActivityPubError}': [{threshold: 'rate < 0.1'}],
    },
    scenarios: {
        slam: {
            executor: 'ramping-arrival-rate',
            exec: 'slam',
            startRate: 0,
            timeUnit: '30s',
            preAllocatedVUs: 100,
            maxVUs: 100,
            stages: [
                { target: 200, duration: '120s'},
                { target: 20, duration: '10s'},
            ],
        },
    },
    maxRedirects: 0,
}

const BASE_URL = __ENV.TEST_HOST;

const actors = [
    {
        id: `${BASE_URL}/`,
        name: 'self',
        type: 'Service',
    }
]

export function setup() {
    // Do not try to run the setup on remote servers
    if (!BASE_URL.endsWith('.local')) return;

    for (let u of actors) {
        if (u.hasOwnProperty('id')) {
            // actor exists
            check(http.get(u.id), {'is ActivityPub': isActivityPub});
        } else {

        }
    }
}

function ActivityPubChecks() {
    return {
        'status 200': isOK,
        'is ActivityPub': isActivityPub,
    }
}

function hasActivityPubType(types) {
    if (!Array.isArray(types )) types = [types];
    return (r) => {
        const status = types.findIndex((e) => e === r.json('type')) > 0;
        errors.add(!status, {errorType: 'ActivityPubError'});
        return status
    }
}

function CollectionChecks() {
    return {
        'has correct Type': hasActivityPubType(['Collection', 'OrderedCollection']),
    }
}

function CollectionPageChecks() {
    return {
        'has correct Type': hasActivityPubType(['CollectionPage', 'OrderedCollectionPage']),
    }
}

function isOK(r) {
    const status = r.status === 200;
    errors.add(!status, {errorType: 'responseStatusError'});
    return status;
}

function isActivityPub(r) {
    const ct = contentType(r);
    const contentTypeStatus = (
        ct.startsWith('application/json')
        || ct.startsWith('application/activity+json')
        || ct.startsWith('application/ld+json')
    );
    const bodyLengthStatus = r.body.length > 0;

    errors.add(!contentTypeStatus, {errorType: 'contentTypeError'});
    errors.add(!bodyLengthStatus, {errorType: 'bodySizeError'});
    return contentTypeStatus && bodyLengthStatus;
}

function actorChecks(u) {
    return Object.assign(
        ActivityPubChecks(),
        isSpecificActor(u),
    );
}

function collectionChecks() {
    return Object.assign(
        ActivityPubChecks(),
        CollectionChecks(),
    );
}

function collectionPageChecks() {
    return Object.assign(
        ActivityPubChecks(),
        CollectionPageChecks(),
    );
}

function isSpecificActor(u) {
    let result = {
        'has body':  (r) => r.body.length > 0,
    };
    for (let prop in u) {
        const propName = `property ${prop.toUpperCase()}`;
        result[propName] = (r) => {
            const ob = r.json();
            return !ob.hasOwnProperty(prop) || ob[prop] === u[prop]
        };
    }
    return result;
}

function getHeader(hdr) {
    return (r) => r.headers.hasOwnProperty(hdr) ? r.headers[hdr].toLowerCase() : '';
}

const contentType = getHeader('Content-Type');

const objectCollections = ['likes', 'shares', 'replies'];
const actorCollections = ['inbox', 'outbox', 'following', 'followers', 'liked', 'likes', 'shares', 'replies'];

function aggregateActorCollections(actor) {
    let collections = [];
    for (let i in actorCollections) {
        const col = actorCollections[i];
        if (actor.hasOwnProperty(col)) {
            collections.push(actor[col])
        }
    }
    if (actor.hasOwnProperty('streams')) {
        collections.push(...actor['streams'])
    }
    return collections;
}

function runSuite(actors, sleepTime = 0) {
    return () => {
        for (let u of actors) {
            if (!u.hasOwnProperty('id')) {
                console.error('invalid actor to test, missing "id" property');
                continue;
            }
            group(u.id, function () {
                const r = http.get(u.id)
                check(r, actorChecks(u));

                const actor = r.json();
                group('collections', function () {
                    for (const colIRI of aggregateActorCollections(actor)) {
                        group(colIRI, function () {
                            const r = http.get(colIRI)
                            check(r, collectionChecks());

                            let col = r.json();
                            let next, pageCount = 0;

                            if (col.hasOwnProperty('first') && col['first'] !== col['id']) {
                                next = col['first']
                            }
                            while (true) {
                                if (col.hasOwnProperty('next') && col['next'] !== col['id']) {
                                    next = col['next']
                                }
                                if (next === '') break;

                                group(`[${pageCount}]${next}`, function () {
                                    const r = http.get(next)
                                    !check(r, collectionPageChecks());
                                    col = r.json();
                                });
                                next = '';
                                pageCount++;
                            }
                        });
                    }
                });
                sleep(sleepTime);
            });
        }
    }
}

export function slam() {
    group('actors', runSuite(actors));
}
