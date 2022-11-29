import re
import yaml

with open("master-config.yaml") as f:
    conf = f.read()
    conf = conf.split("master.yaml: |")[1]

    conf = re.sub(": {{.*?}}", ": value", conf)
    conf = re.sub("{{.*?}}", "", conf)

    conf = conf.replace("|", "value")
    #conf = conf.replace("\n\t\n", "\n")
    #conf = conf.replace("\n", "!")
    #conf = conf.replace("    ", "@")
    #conf = conf.replace("\n\n", "")    
    #conf = conf.replace("    \n", "")

    conf = conf.replace("\n    ", "\n")

    #conf = re.sub(":", ": value", conf)

    print(conf)
    yaml_conf = yaml.safe_load(conf)
    print(yaml_conf)
