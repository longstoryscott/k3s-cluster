#! /bin/bash

for f in ./*/**/*.sh; do
    chmod +x "$f"
done
