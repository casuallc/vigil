#!/bin/bash

# 测试SSH转发服务的脚本
echo "SSH Forwarding Service Test"
echo "=========================="
echo ""
echo "1. Starting SSH forwarding server..."
echo ""

# 为Windows系统创建临时目录
TEMP_DIR=$(echo "$TEMP" | sed 's/\\/\//g')
AUDIT_LOG="$TEMP_DIR/ssh-audit.log"

# 启动SSH转发服务器（在后台运行）
./bbx-ssh --host=localhost --port=2222 --target-host=localhost --target-port=22 --target-username=testuser --target-password=testpassword --audit-log="$AUDIT_LOG" &
SERVER_PID=$!

# 等待服务器启动
sleep 2

echo "2. Testing SSH connection to the forwarding server..."
echo ""

# 测试SSH连接（使用简单的nc命令）
echo "Testing connection to localhost:2222..."
nc localhost 2222 < /dev/null > /dev/null 2>&1 &
NC_PID=$!

sleep 1

# 检查nc是否成功连接
if kill -0 $NC_PID 2>/dev/null; then
    # nc仍在运行，说明连接成功
    kill $NC_PID 2>/dev/null
    echo "✓ SSH forwarding server is running on port 2222"
else
    echo "✗ Failed to connect to SSH forwarding server"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

echo ""
echo "3. Testing SSH command execution..."
echo ""

# 尝试执行简单命令（注意：这可能会失败，因为我们没有实际的SSH服务器在localhost:22上）
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -p 2222 testuser@localhost "echo 'Hello from SSH forwarding service'" 2>&1

# 检查命令执行结果
if [ $? -eq 0 ]; then
    echo ""
    echo "✓ SSH command execution successful"
else
    echo ""
    echo "⚠️  SSH command execution failed (expected if no SSH server on localhost:22)"
fi

echo ""
echo "4. Stopping SSH forwarding server..."
echo ""

# 停止服务器
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo "✓ SSH forwarding server stopped"
echo ""
echo "Test completed!"
