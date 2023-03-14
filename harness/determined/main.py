import time

from common import api
from common.api import authentication, bindings, errors

master_url = "127.0.0.1:8080"
authentication.cli_auth = authentication.Authentication(master_url)
sess = api.Session(master_url, None, None, None)


def metrics_no_paging(trial_ids):
    for trial_id in trial_ids:
        start = time.time()
        
        r = bindings.get_MetricsNoPaging(sess, trialId=trial_id)
        print(f"Trial ID = {trial_id} took {time.time() - start} with # rows = {len(r.steps)}")

print("Metrics no paging")
#metrics_no_paging([1, 2, 3])

def metrics_limit_offset(trial_ids, page_size):
    for trial_id in trial_ids:
        def get_with_offset(offset: int) -> bindings.v1MetricsLimitOffsetResponse:
            return bindings.get_MetricsLimitOffset(
                sess,
                offset=offset,
                limit=page_size,
                trialId=trial_id,
            )

        start = time.time()        
        resps = api.read_paginated(get_with_offset, offset=0, pages="all")
        r = [s for r in resps for s in r.steps]
        print(f"Trial ID = {trial_id} took {time.time() - start} with # rows = {len(r)}")

print("\nMetrics limit offset")        
#metrics_limit_offset([1, 2, 3], page_size=250)

def metrics_keyset(trial_ids, page_size):
    for trial_id in trial_ids:
        start = time.time()
        
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

            #print(resp.prevPage, resp.nextPage)
            key = resp.nextPage
            if key == "":
                break

        print(f"Trial ID = {trial_id} took {time.time() - start} with # rows = {len(r)}")

print("\nMetrics keyset")                
#metrics_keyset([1, 2, 3], page_size=250)


def metrics_streaming_response(trial_ids, page_size):
    for trial_id in trial_ids:
        start = time.time()

        r = []
        for res in bindings.get_MetricsStreaming(sess, trialId=trial_id, size=page_size):
            r.extend(res.steps)            

        print(f"Trial ID = {trial_id} took {time.time() - start} with # rows = {len(r)}")        

print("\nMetrics streaming")
#metrics_streaming_response([1, 2, 3], page_size=250)        

# /:trial_id/metrics/nopage
# /:trial_id/metrics/stream

def metrics_no_paging_echo(trial_ids):
    for trial_id in trial_ids:
        start = time.time()

        resp = api.get("127.0.0.1:8080", f"/trials/{trial_id}/metrics/nopage")
        r = resp.json()
        print(f"Trial ID = {trial_id} took {time.time() - start} with # rows = {len(r)}")

print("\nEcho no paging")
metrics_no_paging_echo([1, 2, 3])        
