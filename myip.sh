#!/bin/sh
echo $(ifconfig eth0 | grep broadcast | awk '{print $2}')