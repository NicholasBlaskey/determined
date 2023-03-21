import common.experimental.determined as det

trial_id = 41
# master_url = "127.0.0.1:8080"
# client.login(master=master_url, user="determined", password="")
# TrialReference()
d = det.Determined()

print("det training metrics")
for i in d.get_trials_training_metrics([trial_id]):
    print(i)
    break
print("det trial training metrics")
for i in det.Determined().get_trial(trial_id).training_metrics():
    print(i)
    break

print("det validation metrics")
for i in d.get_trials_validation_metrics([trial_id]):
    print(i)
    break
print("det trial validation metrics")
for i in det.Determined().get_trial(trial_id).validation_metrics():
    print(i)
    break
