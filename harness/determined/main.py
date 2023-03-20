import time
import json

from common import api
from common.api import authentication, bindings, errors
import matplotlib.pyplot as plt

page_size = 10000

master_url = "127.0.0.1:8080"
authentication.cli_auth = authentication.Authentication(master_url)
sess = api.Session(master_url, None, None, None)


def metrics_no_paging(trial_id):
    return bindings.get_MetricsNoPaging(sess, trialId=trial_id).steps

def metrics_limit_offset(trial_id, page_size=page_size):
    def get_with_offset(offset: int) -> bindings.v1MetricsLimitOffsetResponse:
        return bindings.get_MetricsLimitOffset(
            sess,
            offset=offset,
            limit=page_size,
            trialId=trial_id,
        )

    resps = api.read_paginated(get_with_offset, offset=0, pages="all")
    return [s for r in resps for s in r.steps]

def metrics_keyset(trial_id, page_size=page_size):
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

def metrics_streaming_response(trial_id, page_size=page_size):
    r = []
    for res in bindings.get_MetricsStreaming(sess, trialId=trial_id, size=page_size):
        r.extend(res.steps)            
    return r

def metrics_no_paging_echo(trial_id):
    resp = api.get("127.0.0.1:8080", f"/trials/{trial_id}/metrics/nopage")
    r = bindings.v1MetricsStreamingResponse.from_json(resp.json())
    return r.steps


def iter_lines(obj, chunk_size=512, decode_unicode=False, delimiter=None):
    """Iterates over the response data, one line at a time.  When
    stream=True is set on the request, this avoids reading the
    content at once into memory for large responses.

    .. note:: This method is not reentrant safe.
    """

    pending = None

    for chunk in obj.iter_content(
        chunk_size=chunk_size, decode_unicode=decode_unicode
    ):
        if pending is not None:
            chunk = pending + chunk

        if delimiter:
            lines = chunk.split(delimiter)
        else:
            lines = chunk.splitlines()

        if lines and lines[-1] and chunk and lines[-1][-1] == chunk[-1]:
            pending = lines.pop()
        else:
            pending = None

        yield from lines

    if pending is not None:
        yield pending

            
def metrics_stream_echo(trial_id, page_size=page_size):
    start = time.time()

    # /trials/6/metrics/stream?size=10000
    r = []
    resp = api.get("127.0.0.1:8080", f"/trials/{trial_id}/metrics/stream?size={page_size}",
                   stream=True)
    #resp.raw.chunked = True    
    #for line in resp.iter_lines(): # chunk_size=len(resp.text)

    # So the question is what do we do with something over the chunk_size limit???
    # We could split the steps but what if one step alone is too large???
    for line in resp.iter_lines(chunk_size=1024 * 1024): # 10,000 takes 30 seconds # 100,000 takes 10
        pass

    
    #for line in resp.iter_lines(chunk_size=1): # 10,000 takes 30 seconds # 100,000 takes 10
    #for line in iter_lines(resp, chunk_size=1): # 10,000 takes 30 seconds # 100,000 takes 10
        #print("LINE", c)
        #c += 1
        #pass
        #print(line)
    '''
    for chunk in resp.iter_content(chunk_size=10000):
        # this alone takes 30 seconds...
        L = [chunk[i:i+1] for i in range(len(chunk))]
        for i in L:    
            if i == b'\n':
                c += 1
                #print(i)
    '''

    '''
        cur = left_over + chunk
        split = cur.splitlines()

        if cur[-1] == b'\n':
            left_over = b''
        else:
            #print(len(split))
            left_over = split[-1]
            del split[-1]

        c += len(split)
        print(c)
    '''

    '''

        #print(type(chunk))
        #if "\n" in chunk:
        #    print("new line")
            
        
        #if line:
        #    r.extend(bindings.v1MetricsStreamingResponse.from_json(
        #        json.loads(line)).steps)
    print(c)
    '''
    return r


routes = [
    #("Metrics no paging", metrics_no_paging),
    #("Metrics limit offset", metrics_limit_offset),
    #("Metrics keyset", metrics_keyset),
    #("Metrics grpc stream", metrics_streaming_response),
    #("Metrics no paging echo", metrics_no_paging_echo),
    ("Metrics ndjson streaming", metrics_stream_echo), 
]

def main():
    #trial_ids = [1, 2, 3, 4, 5, 6, 7]
    trial_ids = [7]    
    #trial_ids = [1, 2, 3, 4, 5, 6, 7]
    #trial_ids = [6]

    x_y_labels = []
    for r in routes:
        print(r[0])

        x, y, labels = [], [], r[0]
        
        for trial_id in trial_ids:
            start = time.time()

            steps = r[1](trial_id)
            #print(steps[0].to_json())
            print(f"Trial ID = {trial_id} took {time.time() - start} with # rows = {len(steps)}")
            x.append(len(steps))
            y.append(time.time() - start)
            
            '''
            try:
                steps = r[1](trial_id)
                print(steps[0].to_json())
                print(f"Trial ID = {trial_id} took {time.time() - start} with # rows = {len(steps)}")
                x.append(len(steps))
                y.append(time.time() - start)
            except:
                print("FAILURE")
                print("Sleeping for 20")
                time.sleep(20)
                print("Done sleeping for 10")                
            '''
        print("")

        x_y_labels.append([x, y, labels])
        
    for line in x_y_labels:
        plt.plot(line[0], line[1], label=line[2])
    plt.legend()
    plt.xlabel("rows in metrics")
    plt.ylabel("response time (seconds)")        
    #plt.show()
    

    
if __name__ == "__main__":
    main()
