#!/bin/bash

while true
do
  http https://introspect.example.com/mandelbrot >/dev/null 2>&1
  echo -n .
done
