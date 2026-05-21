#!/bin/sh
set -e
# 修复挂载卷权限（宿主机创建的目录可能不属于 app 用户）
chown -R app:app /app/data
exec su-exec app /app/minimax-studio "$@"
