#!/bin/bash

mv yd-qgo /usr/bin/
mkdir -p /usr/share/yd-qgo/icons/dark
mkdir -p /usr/share/yd-qgo/icons/light
cp icons/yd* /usr/share/yd-qgo/icons
cp icons/dark/* /usr/share/yd-qgo/icons/dark/
cp icons/light/* /usr/share/yd-qgo/icons/light/
#cp README.md /usr/share/yd-qgo/

