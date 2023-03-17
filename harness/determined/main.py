import time
import json

from common import api
from common.api import authentication, bindings, errors

master_url = "127.0.0.1:8080"
authentication.cli_auth = authentication.Authentication(master_url)
sess = api.Session(master_url, None, None, None)


def metrics_no_paging(trial_id):
    return bindings.get_MetricsNoPaging(sess, trialId=trial_id).steps

def metrics_limit_offset(trial_id, page_size=10):
    def get_with_offset(offset: int) -> bindings.v1MetricsLimitOffsetResponse:
        return bindings.get_MetricsLimitOffset(
            sess,
            offset=offset,
            limit=page_size,
            trialId=trial_id,
        )

    resps = api.read_paginated(get_with_offset, offset=0, pages="all")
    return [s for r in resps for s in r.steps]

def metrics_keyset(trial_ids, page_size):
    key = ""
    r = []
    while True:
        resp = bindings.get_MetricsKeyset(
            sess,
            key=key,
            size=page_size,
            trialId=trial_id,
        )
        r.extend(resp.steps)

        key = resp.nextPage
        if key == "":
            break
    return r

def metrics_streaming_response(trial_ids, page_size):
    r = []
    for res in bindings.get_MetricsStreaming(sess, trialId=trial_id, size=page_size):
        r.extend(res.steps)            
    return r

def metrics_no_paging_echo(trial_ids):
    resp = api.get("127.0.0.1:8080", f"/trials/{trial_id}/metrics/nopage")
    r = bindings.v1MetricsStreamingResponse.from_json(resp.json()["steps"])
    return r.steps


def metrics_stream_echo(trial_ids, page_size):
    start = time.time()

    r = []
    resp = api.get("127.0.0.1:8080", f"/trials/{trial_id}/metrics/stream?size={page_size}",
                   stream=False)
    for line in resp.iter_lines(): # chunk_size=len(resp.text)
        if line:
            r.extend(bindings.v1MetricsStreamingResponse.from_json(
                json.loads(line)).steps)



routes = [
    ("Metrics no paging", metrics_no_paging),
    ("Metrics limit offset", metrics_limit_offset),
    ("Metrics keyset", metrics_keyset),
    ("Metrics grpc stream", metrics_streaming_response),
    ("Metrics no paging echo", metrics_no_paging_echo),
    ("Metrics ndjson streaming", metrics_stream_echo), 
]

def main():
    trial_ids = [1]

    for r in routes:
        print(r[0])

        for trial_id in trial_ids:
            start = time.time()
            r = r[1](trial_id)
            print(f"Trial ID = {trial_id} took {time.time() - start} with # rows = {len(r)}")
        print("")


if __name__ == "__main__":
    main()
