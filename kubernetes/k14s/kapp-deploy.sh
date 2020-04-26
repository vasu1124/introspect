#!/bin/bash

kapp deploy -a introspect -f <(ytt -f .)
