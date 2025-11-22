#!/bin/bash
# 设置测试环境

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "==> Building Docker images..."
cd "$PROJECT_ROOT/tests/docker"
docker-compose build

echo ""
echo "==> Starting containers..."
docker-compose up -d

echo ""
echo "==> Waiting for SSH services to be ready..."
sleep 5

# 检查 SSH 是否就绪
for i in {1..30}; do
    if docker exec ansigo-control ssh -o ConnectTimeout=2 testuser@172.28.0.11 echo "SSH ready" 2>/dev/null; then
        echo "✓ target1 SSH ready"
        break
    fi
    echo "Waiting for target1 SSH... ($i/30)"
    sleep 1
done

for i in {1..30}; do
    if docker exec ansigo-control ssh -o ConnectTimeout=2 testuser@172.28.0.12 echo "SSH ready" 2>/dev/null; then
        echo "✓ target2 SSH ready"
        break
    fi
    echo "Waiting for target2 SSH... ($i/30)"
    sleep 1
done

for i in {1..30}; do
    if docker exec ansigo-control ssh -o ConnectTimeout=2 testuser@172.28.0.13 echo "SSH ready" 2>/dev/null; then
        echo "✓ target3 SSH ready"
        break
    fi
    echo "Waiting for target3 SSH... ($i/30)"
    sleep 1
done

echo ""
echo "==> Test environment is ready!"
echo ""
echo "To enter the control node:"
echo "  docker exec -it ansigo-control bash"
echo ""
echo "To run tests:"
echo "  docker exec -it ansigo-control bash -c 'cd /workspace/tests && ./scripts/run-phase1-tests.sh'"
echo ""
echo "To stop the environment:"
echo "  cd $PROJECT_ROOT/tests/docker && docker-compose down"
