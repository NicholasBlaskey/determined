import time
import json

from common import api
import common.experimental.determined as det
from common.api import authentication, bindings, errors
import matplotlib.pyplot as plt


from determined.experimental import client

#master_url = "127.0.0.1:8080"
#client.login(master=master_url, user="determined", password="")
#TrialReference()
d = det.Determined()
#t = d.get_trial(5)
for i in d.get_trials_training_metrics([5]):
    print(i)
    break
for i in det.Determined().get_trial(5).training_metrics():
    print(i)
    break
    
