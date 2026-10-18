#!/usr/bin/env bash
# lab/down.sh — удалить стенд (вместе с ним исчезают все интерфейсы и
правила)
for n in srv cli inet; do ip netns del $n 2>/dev/null; done; true
