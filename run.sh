#!/bin/bash
pkill Xephyr
export DISPLAY=:0
Xephyr -screen 1280x720 -br -ac -noreset :1 &
sleep 1s
DISPLAY=:1 ./windowmanageragain