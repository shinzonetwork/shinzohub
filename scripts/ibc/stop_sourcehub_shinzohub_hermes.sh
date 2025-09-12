#!/bin/sh

set -e

# Kill running processes
killall sourcehubd 2>/dev/null || true
killall shinzohubd 2>/dev/null || true
killall hermes 2>/dev/null || true

# Cleanup directories
rm -rf sourcehub.log
rm -rf shinzohub.log
rm -rf hermes.log