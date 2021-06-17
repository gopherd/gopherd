#!/bin/bash

openssl ecparam -genkey -name prime256v1 -out ../etc/ec256.p1
openssl pkcs8 -in ../etc/ec256.p1 -topk8 -out ../etc/ec256.p8 -nocrypt
openssl ec -in ../etc/ec256.p8 -pubout -out ../etc/ec256.pub.p8
rm ../etc/ec256.p1
