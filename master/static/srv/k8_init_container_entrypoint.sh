#!/bin/bash

# Script takes args: NUM_FILES SRC DST and un-tars files from SRC to DST.

if [ $1 -le 0 ]
then
    exit 0
fi

for var in $(seq 1 1 $1)
do
    IDX=$(( $var - 1 ))
    SRC=$2/$IDX.tar.gz
    DST=$3/$IDX
    mkdir $DST

    if [ ! -f $SRC ]; then
        echo "Not found $SRC"
        continue
    fi
    
    if [ $(id -u) == "0" ]
    then
        tar --same-owner -xvf $SRC -C $DST
    else
        tar -xvf $SRC -C $DST
    fi
done
echo "Done"
