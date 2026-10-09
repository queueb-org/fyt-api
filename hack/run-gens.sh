#!/bin/bash
# See the README for the generator versions used by this module.

project_dir="$(dirname $0)/.."
gena run \
  register-gen \
  deepcopy-gen \
  conversion-gen \
  defaulter-gen \
  -v=4 -d "$project_dir"
